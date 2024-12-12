package ui

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/opd-ai/dndbot/srv/generator"
)

// srv/ui/handlers.go
func (ui *GeneratorUI) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Printf("No session cookie found: %v", err)
		http.Error(w, "No session found", http.StatusBadRequest)
		return
	}

	sessionID := cookie.Value
	if !isValidSession(sessionID) {
		log.Printf("[Session %s] Invalid session ID", sessionID)
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	log.Printf("[Session %s] WebSocket connection attempt", sessionID)

	ui.sessionsM.RLock()
	progress, exists := ui.sessions[sessionID]
	ui.sessionsM.RUnlock()

	if !exists {
		log.Printf("[Session %s] Checking cache for session", sessionID)
		if cachedProgress, found := ui.cache.Get(sessionID); found {
			progress = cachedProgress.(*generator.GenerationProgress)
			log.Printf("[Session %s] Found cached session", sessionID)
		} else {
			log.Printf("[Session %s] Session not found in memory or cache", sessionID)
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
		log.Printf("[Session %s] WebSocket upgrade failed: %v", sessionID, err)
		http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
		return
	}
	log.Printf("[Session %s] WebSocket connection established", sessionID)

	// Setup connection parameters
	conn.SetReadLimit(4096)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	// Set the connection in progress object
	progress.Lock()
	if progress.WSConn != nil {
		log.Printf("[Session %s] Closing existing WebSocket connection", sessionID)
		if err := progress.WSConn.Close(); err != nil {
			log.Printf("[Session %s] Error closing existing connection: %v", sessionID, err)
		}
	}
	progress.WSConn = conn
	progress.Unlock()
	log.Printf("[Session %s] WebSocket connection registered with progress object", sessionID)

	// Send historical messages
	if history, exists := ui.msgHistory[sessionID]; exists {
		history.mu.RLock()
		messages := history.GetMessages()
		history.mu.RUnlock()

		log.Printf("[Session %s] Sending %d historical messages", sessionID, len(messages))
		for i, msg := range messages {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("[Session %s] Failed to send historical message %d: %v", sessionID, i, err)
				continue
			}
			log.Printf("[Session %s] Historical message %d sent: %s", sessionID, i, msg.Message)
		}
	} else {
		log.Printf("[Session %s] No message history found", sessionID)
	}

	// Send current state message
	currentState := progress.GetState()
	stateMsg := generator.NewWSMessage(
		"update",
		string(currentState),
		fmt.Sprintf("Generator state: %s", currentState),
		fmt.Sprintf("🎲 Current state: %s", currentState),
	)

	if err := conn.WriteJSON(stateMsg); err != nil {
		log.Printf("[Session %s] Failed to send state message: %v", sessionID, err)
	} else {
		log.Printf("[Session %s] Sent current state message: %s", sessionID, currentState)
		ui.AddMessage(sessionID, stateMsg)
	}

	// Setup ping/pong
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
					log.Printf("[Session %s] Ping failed: %v", sessionID, err)
					return
				}
				log.Printf("[Session %s] Ping sent successfully", sessionID)
			case <-done:
				log.Printf("[Session %s] Ping loop terminated", sessionID)
				return
			}
		}
	}()

	// Main message loop
	log.Printf("[Session %s] Starting main message loop", sessionID)
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Session %s] WebSocket error: %v", sessionID, err)
			} else {
				log.Printf("[Session %s] WebSocket closed normally: %v", sessionID, err)
			}
			break
		}

		log.Printf("[Session %s] Received message type: %d", sessionID, messageType)

		if messageType == websocket.CloseMessage {
			log.Printf("[Session %s] Received close message", sessionID)
			break
		} else if messageType == websocket.TextMessage {
			log.Printf("[Session %s] Received text message: %s", sessionID, string(message))
		}
	}

	// Cleanup
	log.Printf("[Session %s] Cleaning up WebSocket connection", sessionID)
	progress.Lock()
	if progress.WSConn == conn {
		progress.WSConn = nil
		log.Printf("[Session %s] Cleared WebSocket connection from progress", sessionID)
	}
	progress.Unlock()

	if err := conn.Close(); err != nil {
		log.Printf("[Session %s] Error closing WebSocket connection: %v", sessionID, err)
	} else {
		log.Printf("[Session %s] WebSocket connection closed successfully", sessionID)
	}
}
