package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/opd-ai/dndbot/srv/components"
	"github.com/opd-ai/dndbot/srv/generator"
)

func (ui *GeneratorUI) handleHome(w http.ResponseWriter, r *http.Request) {
	components.Layout().Render(r.Context(), w)
	components.GeneratorForm().Render(r.Context(), w)
}

func (ui *GeneratorUI) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	prompt := r.FormValue("prompt")
	if prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// Check for existing valid session
	var sessionID string
	if cookie, err := r.Cookie("session_id"); err == nil && isValidSession(cookie.Value) {
		sessionID = cookie.Value
	} else {
		// Create new session if none exists or invalid
		sessionID = uuid.New().String()
	}

	// Set or update session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	progress := &generator.GenerationProgress{
		SessionID: sessionID,
		Done:      make(chan bool),
		StartTime: time.Now(),
		State:     generator.StateInitialized,
		IsActive:  true,
	}

	ui.sessionsM.Lock()
	ui.sessions[sessionID] = progress
	// Initialize message history if it doesn't exist
	if _, exists := ui.msgHistory[sessionID]; !exists {
		ui.msgHistory[sessionID] = &MessageHistory{
			Messages: make([]generator.WSMessage, 0),
		}
	}
	ui.sessionsM.Unlock()

	// Initialize message history for new session
	if _, exists := ui.msgHistory[sessionID]; !exists {
		ui.msgHistory[sessionID] = &MessageHistory{
			Messages: make([]generator.WSMessage, 0),
		}
	}

	go func() {
		defer func() {
			ui.cleanupSession(sessionID, progress)
		}()

		progress.UpdateState(generator.StateGenerating)
		if err := generator.GenerateAdventure(progress, prompt); err != nil {
			progress.UpdateState(generator.StateError)
			progress.Error = err
			return
		}

		progress.UpdateState(generator.StateCompleted)
	}()

	components.GenerationStatus(sessionID).Render(r.Context(), w)
}

func (ui *GeneratorUI) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Printf("No session cookie found: %v", err)
		http.Error(w, "No session found", http.StatusBadRequest)
		return
	}

	sessionID := cookie.Value
	if !isValidSession(sessionID) {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	ui.sessionsM.RLock()
	progress, exists := ui.sessions[sessionID]
	ui.sessionsM.RUnlock()

	if !exists {
		// Check cache for completed session
		if cachedProgress, found := ui.cache.Get(sessionID); found {
			progress = cachedProgress.(*generator.GenerationProgress)
		} else {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection: %v", err)
		}
		progress.Lock()
		progress.WSConn = nil
		progress.Unlock()
	}()

	conn.SetReadLimit(4096)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	// Send historical messages
	if history, exists := ui.msgHistory[sessionID]; exists {
		history.mu.RLock()
		for _, msg := range history.Messages {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("Failed to send historical message: %v", err)
			}
		}
		history.mu.RUnlock()
	}

	progress.Lock()
	if progress.WSConn != nil {
		if err := progress.WSConn.Close(); err != nil {
			log.Printf("Error closing existing WebSocket connection: %v", err)
		}
	}
	progress.WSConn = conn
	progress.Unlock()

	updates := make(chan generator.WSMessage, 10)
	defer close(updates)

	initialMsg := generator.WSMessage{
		Type:    "update",
		Status:  "connected",
		Message: "Connection established",
		Output:  "🎲 Initializing adventure generation...",
	}

	if err := conn.WriteJSON(initialMsg); err != nil {
		log.Printf("Failed to send initial message: %v", err)
		return
	}

	ui.AddMessage(sessionID, initialMsg)

	go func() {
		for msg := range updates {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("Failed to send message: %v", err)
				return
			}
			ui.AddMessage(sessionID, msg)
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})
	defer close(done)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	for {
		messageType, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		if messageType == websocket.CloseMessage {
			break
		}
	}
}

func (ui *GeneratorUI) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	log.Printf("[DEBUG] Starting handleGetMessages for session: %s", sessionID)

	// Set headers early
	w.Header().Set("Content-Type", "application/json")

	// Use buffered channel for message passing
	messageChan := make(chan []generator.WSMessage, 1)
	//	errChan := make(chan error, 1)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	go func() {
		ui.sessionsM.RLock()
		history, exists := ui.msgHistory[sessionID]
		ui.sessionsM.RUnlock()

		if !exists {
			messageChan <- []generator.WSMessage{}
			return
		}

		messages := history.GetMessages()
		messageChan <- messages
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		log.Printf("[ERROR] Timeout getting messages for session: %s", sessionID)
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		return

	case messages := <-messageChan:
		if err := json.NewEncoder(w).Encode(messages); err != nil {
			log.Printf("[ERROR] Failed to encode messages for session %s: %v", sessionID, err)
			http.Error(w, "Failed to encode messages", http.StatusInternalServerError)
			return
		}
		log.Printf("[DEBUG] Successfully sent messages for session: %s", sessionID)
	}
}

// Add this to handlers.go
func (ui *GeneratorUI) handleCheckSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		// No existing session, render empty status
		components.GenerationStatus("").Render(r.Context(), w)
		return
	}

	sessionID := cookie.Value
	if !isValidSession(sessionID) {
		// Invalid session, clear cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_id",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		components.GenerationStatus("").Render(r.Context(), w)
		return
	}

	// Valid session exists, render with session ID
	components.GenerationStatus(sessionID).Render(r.Context(), w)
}

// Add this helper function
func isValidSession(sessionID string) bool {
	if sessionID == "" {
		return false
	}

	// Validate UUID format
	_, err := uuid.Parse(sessionID)
	return err == nil
}

func formatMessages(messages []generator.WSMessage) string {
	var html strings.Builder
	for _, msg := range messages {
		html.WriteString(fmt.Sprintf(`
            <div class="message %s">
                <div class="message-header">
                    <span>%s</span>
                    <span>%s</span>
                </div>
                %s
                %s
            </div>
        `,
			msg.Status,
			msg.Status,
			msg.Timestamp.Format("15:04:05"),
			formatContent(msg.Message),
			formatOutput(msg.Output),
		))
	}
	return html.String()
}
