package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	generator "github.com/opd-ai/dndbot/srv/generator"
	"github.com/opd-ai/dndbot/srv/util"
)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	progress, valid := GlobalSessionManager.GetSession(sessionID)
	if !valid {
		// Add retry mechanism with backoff
		for i := 0; i < 5; i++ {
			time.Sleep(time.Duration(i*100) * time.Millisecond)
			progress, valid = GlobalSessionManager.GetSession(sessionID)
			if valid {
				break
			}
		}

		if !valid {
			util.ErrorLogger.Printf("Invalid session ID requested: %s", sessionID)
			http.Error(w, "Invalid session ID", http.StatusBadRequest)
			return
		}
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// In production, implement proper origin checking
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		util.ErrorLogger.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	progress.Lock()
	progress.WSConn = conn
	progress.UpdateState(generator.StateConnected)
	progress.Unlock()

	// Keep-alive Handler
	go HandleKeepAlive(progress)

	// Message Handler
	HandleWebSocketMessages(progress, conn)
}

func HandleKeepAlive(progress *generator.GenerationProgress) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-progress.Done:
			return
		case <-ticker.C:
			if progress.WSConn != nil {
				if err := progress.WSConn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}
}

func HandleWebSocketMessages(progress *generator.GenerationProgress, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		progress.SetActive(false)
	}()

	conn.SetReadLimit(512) // Set reasonable message size limit
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		messageType, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				util.ErrorLogger.Printf("WebSocket error: %v", err)
			}
			break
		}
		if messageType == websocket.CloseMessage {
			break
		}
	}
}
