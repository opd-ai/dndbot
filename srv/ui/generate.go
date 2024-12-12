package ui

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/opd-ai/dndbot/srv/components"
	"github.com/opd-ai/dndbot/srv/generator"
)

// srv/ui/handlers.go - handleGenerate
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

	// Create new session
	sessionID := uuid.New().String()
	w.Header().Set("X-Session-Id", sessionID)

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Create progress object
	progress := &generator.GenerationProgress{
		SessionID: sessionID,
		Done:      make(chan bool),
		StartTime: time.Now(),
		State:     generator.StateInitialized,
		IsActive:  true,
	}

	ui.sessionsM.Lock()
	ui.sessions[sessionID] = progress
	if _, exists := ui.msgHistory[sessionID]; !exists {
		ui.msgHistory[sessionID] = &MessageHistory{
			Messages: make([]generator.WSMessage, 0),
		}
	}
	ui.sessionsM.Unlock()

	// Start generation immediately, don't wait for WebSocket
	go func() {
		log.Printf("[Session %s] Starting generation", sessionID)
		if err := generator.GenerateAdventure(progress, prompt); err != nil {
			log.Printf("[Session %s] Generation error: %v", sessionID, err)
			progress.UpdateState(generator.StateError)
			progress.SendUpdate(fmt.Sprintf("Error: %v", err))
		}
	}()

	components.GenerationStatus(sessionID).Render(r.Context(), w)
}
