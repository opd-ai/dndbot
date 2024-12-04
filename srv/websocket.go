package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// handlers.go
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	progress, valid := GlobalSessionManager.GetSession(sessionID)
	if !valid {
		ErrorLogger.Printf("Invalid session ID requested: %s", sessionID)
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Configure appropriately for production
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ErrorLogger.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	progress.mu.Lock()
	progress.WSConn = conn
	progress.State = StateConnected
	progress.mu.Unlock()

	// Keep connection alive with ping/pong
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-progress.Done:
				return
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Handle connection closure
	defer func() {
		conn.Close()
		progress.SetActive(false)
	}()

	// Main message loop
	for {
		messageType, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				ErrorLogger.Printf("WebSocket error: %v", err)
			}
			break
		}
		if messageType == websocket.CloseMessage {
			break
		}
	}
}
