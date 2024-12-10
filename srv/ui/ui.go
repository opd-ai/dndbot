// srv/ui/ui.go
package ui

import (
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
		http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	progress.Lock()
	progress.WSConn = conn
	progress.State = generator.StateConnected
	progress.Unlock()

	// Keep connection alive until generation is complete or client disconnects
	select {
	case <-progress.Done:
		return
	case <-r.Context().Done():
		return
	}
}
