package ui

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/opd-ai/dndbot/srv/generator"
)

func (ui *GeneratorUI) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		log.Printf("No session cookie found: %v", err)
		http.Error(w, "No session found", http.StatusBadRequest)
		return
	}

	sessionID := cookie.Value
	if !isValidSession(sessionID) {
		log.Printf("Invalid session ID: %s", sessionID)
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	log.Printf("[Session %s] WebSocket connection request received", sessionID)

	ui.sessionsM.RLock()
	progress, exists := ui.sessions[sessionID]
	ui.sessionsM.RUnlock()

	if !exists {
		log.Printf("[Session %s] Active session not found, checking cache", sessionID)
		// Check cache for completed session
		if cachedProgress, found := ui.cache.Get(sessionID); found {
			progress = cachedProgress.(*generator.GenerationProgress)
			log.Printf("[Session %s] Found cached session", sessionID)
		} else {
			log.Printf("[Session %s] Session not found in cache", sessionID)
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

	// Create message buffer channel
	msgBuffer := make(chan generator.WSMessage, 100)
	log.Printf("[Session %s] Message buffer created", sessionID)

	cleanup := func() {
		log.Printf("[Session %s] Cleaning up WebSocket connection", sessionID)
		close(msgBuffer)
		if err := conn.Close(); err != nil {
			log.Printf("[Session %s] Error closing WebSocket connection: %v", sessionID, err)
		}
		progress.Lock()
		progress.WSConn = nil
		progress.Unlock()
		log.Printf("[Session %s] Cleanup completed", sessionID)
	}
	defer cleanup()

	conn.SetReadLimit(4096)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	// Send historical messages
	if history, exists := ui.msgHistory[sessionID]; exists {
		history.mu.RLock()
		messageCount := len(history.Messages)
		log.Printf("[Session %s] Sending %d historical messages", sessionID, messageCount)
		for i, msg := range history.Messages {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("[Session %s] Failed to send historical message %d/%d: %v", sessionID, i+1, messageCount, err)
				continue
			}
		}
		history.mu.RUnlock()
		log.Printf("[Session %s] Historical messages sent", sessionID)
	}

	// Set up message sender goroutine
	go func() {
		log.Printf("[Session %s] Starting message sender goroutine", sessionID)
		for msg := range msgBuffer {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("[Session %s] Failed to send message: %v", sessionID, err)
				return
			}
			ui.AddMessage(sessionID, msg)
			log.Printf("[Session %s] Message sent and added to history: %s", sessionID, msg.Message)
		}
	}()

	progress.Lock()
	if progress.WSConn != nil {
		log.Printf("[Session %s] Closing existing WebSocket connection", sessionID)
		if err := progress.WSConn.Close(); err != nil {
			log.Printf("[Session %s] Error closing existing WebSocket connection: %v", sessionID, err)
		}
	}
	progress.WSConn = conn
	progress.Unlock()
	log.Printf("[Session %s] New WebSocket connection set", sessionID)

	// Send initial connection message
	initialMsg := generator.NewWSMessage(
		"update",
		"connected",
		"Connection established",
		"🎲 Initializing adventure generation...",
	)

	log.Printf("[Session %s] Sending initial message", sessionID)
	msgBuffer <- initialMsg
	ui.AddMessage(sessionID, initialMsg)

	// Set up ping/pong
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := make(chan struct{})
	defer close(done)

	go func() {
		defer ticker.Stop()
		log.Printf("[Session %s] Starting ping/pong handler", sessionID)
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("[Session %s] Ping failed: %v", sessionID, err)
					return
				}
				log.Printf("[Session %s] Ping sent", sessionID)
			case <-done:
				log.Printf("[Session %s] Ping/pong handler stopped", sessionID)
				return
			}
		}
	}()

	// Main message loop
	log.Printf("[Session %s] Entering main message loop", sessionID)
	for {
		messageType, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[Session %s] WebSocket error: %v", sessionID, err)
			} else {
				log.Printf("[Session %s] WebSocket closed: %v", sessionID, err)
			}
			break
		}
		if messageType == websocket.CloseMessage {
			log.Printf("[Session %s] Received close message", sessionID)
			break
		}
	}
	log.Printf("[Session %s] WebSocket connection terminated", sessionID)
}
