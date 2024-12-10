// srv/ui/ui.go
package ui

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/opd-ai/dndbot/srv/components"
	"github.com/opd-ai/dndbot/srv/generator"
)

type GeneratorUI struct {
	router    chi.Router
	sessions  map[string]*generator.GenerationProgress
	sessionsM sync.RWMutex
}

// Better session cleanup handling
func (ui *GeneratorUI) cleanupSession(sessionID string, progress *generator.GenerationProgress) {
	progress.SetActive(false)
	progress.Lock()
	if progress.WSConn != nil {
		progress.WSConn.Close()
		progress.WSConn = nil
	}
	progress.Unlock()

	ui.sessionsM.Lock()
	delete(ui.sessions, sessionID)
	ui.sessionsM.Unlock()

	close(progress.Done)
}

func NewGeneratorUI() *GeneratorUI {
	ui := &GeneratorUI{
		router:   chi.NewRouter(),
		sessions: make(map[string]*generator.GenerationProgress),
	}

	ui.setupRoutes()
	return ui
}

func (ui *GeneratorUI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ui.router.ServeHTTP(w, r)
}

func (ui *GeneratorUI) setupRoutes() {
	// Serve static files
	fileServer := http.FileServer(http.Dir("static"))
	ui.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))
	outputServer := http.FileServer(http.Dir("outputs"))
	ui.router.Handle("/outputs/*", http.StripPrefix("/outputs/", outputServer))

	// API routes
	ui.router.Get("/", ui.handleHome)
	ui.router.Post("/generate", ui.handleGenerate)
	ui.router.Get("/ws/{sessionID}", ui.handleWebSocket)
}

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

	sessionID := uuid.New().String()

	progress := &generator.GenerationProgress{
		SessionID: sessionID,
		Done:      make(chan bool),
		StartTime: time.Now(),
		State:     generator.StateInitialized,
		IsActive:  true,
	}

	ui.sessionsM.Lock()
	ui.sessions[sessionID] = progress
	ui.sessionsM.Unlock()

	// Start generation in background
	go func() {
		defer func() {
			progress.SetActive(false)
			close(progress.Done)

			// Cleanup session after completion
			ui.sessionsM.Lock()
			delete(ui.sessions, sessionID)
			ui.sessionsM.Unlock()
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
	sessionID := chi.URLParam(r, "sessionID")

	ui.sessionsM.RLock()
	progress, exists := ui.sessions[sessionID]
	ui.sessionsM.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
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

	// Ensure proper cleanup
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection: %v", err)
		}
		progress.Lock()
		progress.WSConn = nil
		progress.Unlock()
	}()

	// Set connection parameters
	conn.SetReadLimit(4096)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	// Update connection safely
	progress.Lock()
	if progress.WSConn != nil {
		// Close existing connection if any
		if err := progress.WSConn.Close(); err != nil {
			log.Printf("Error closing existing WebSocket connection: %v", err)
		}
	}
	progress.WSConn = conn
	progress.Unlock()

	// Create buffered message channel for updates
	updates := make(chan generator.WSMessage, 10)
	defer close(updates)

	// Send initial connection message
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

	// Start message sender goroutine
	go func() {
		for msg := range updates {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("Failed to send message: %v", err)
				return
			}
		}
	}()

	// Start ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Create done channel for cleanup
	done := make(chan struct{})
	defer close(done)

	// Start health check goroutine
	go func() {
		healthTicker := time.NewTicker(5 * time.Second)
		defer healthTicker.Stop()

		for {
			select {
			case <-healthTicker.C:
				if !progress.IsStillActive() {
					updates <- generator.WSMessage{
						Type:    "error",
						Status:  "disconnected",
						Message: "Generation process stopped",
					}
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Main event loop
	for {
		select {
		case <-ticker.C:
			if err := conn.WriteControl(
				websocket.PingMessage,
				[]byte{},
				time.Now().Add(10*time.Second),
			); err != nil {
				log.Printf("Ping failed: %v", err)
				return
			}

		case <-progress.Done:
			// Send completion message
			finalMsg := generator.WSMessage{
				Type:    "complete",
				Status:  "completed",
				Message: "Generation completed",
				Output:  progress.Output,
			}
			if err := conn.WriteJSON(finalMsg); err != nil {
				log.Printf("Failed to send completion message: %v", err)
			}
			return

		case <-r.Context().Done():
			// Send disconnection message
			disconnectMsg := generator.WSMessage{
				Type:    "disconnect",
				Status:  "closed",
				Message: "Connection closed by client",
			}
			if err := conn.WriteJSON(disconnectMsg); err != nil {
				log.Printf("Failed to send disconnect message: %v", err)
			}
			return
		}

		// Check connection state
		if !progress.IsStillActive() {
			return
		}
	}
}
