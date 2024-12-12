package ui

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/opd-ai/dndbot/srv/components"
	"github.com/opd-ai/dndbot/srv/generator"
)

func (ui *GeneratorUI) handleHome(w http.ResponseWriter, r *http.Request) {
	components.Layout().Render(r.Context(), w)
	components.GeneratorForm().Render(r.Context(), w)
}

// srv/ui/handlers.go
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

	// Wait for WebSocket connection before starting generation
	go func() {
		// Wait up to 5 seconds for WebSocket connection
		timeout := time.After(5 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				log.Printf("Timeout waiting for WebSocket connection for session %s", sessionID)
				return
			case <-ticker.C:
				progress.Lock()
				if progress.WSConn != nil {
					progress.Unlock()
					// Start generation once we have a WebSocket connection
					if err := generator.GenerateAdventure(progress, prompt); err != nil {
						log.Printf("Generation error: %v", err)
						progress.UpdateState(generator.StateError)
						progress.SendUpdate(fmt.Sprintf("Error: %v", err))
					}
					return
				}
				progress.Unlock()
			}
		}
	}()

	components.GenerationStatus(sessionID).Render(r.Context(), w)
}

func (ui *GeneratorUI) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	ui.sessionsM.RLock()
	history, exists := ui.msgHistory[sessionID]
	ui.sessionsM.RUnlock()

	if !exists {
		w.Write([]byte(""))
		return
	}

	messages := history.GetMessages()
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(formatMessages(messages)))
}

func (ui *GeneratorUI) handleCheckSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-Id")
	if sessionID == "" {
		// Try cookie as fallback
		if cookie, err := r.Cookie("session_id"); err == nil {
			sessionID = cookie.Value
		}
	}

	if !isValidSession(sessionID) {
		components.GenerationStatus("").Render(r.Context(), w)
		return
	}

	// Check if session exists in memory or cache
	ui.sessionsM.RLock()
	_, exists := ui.sessions[sessionID]
	ui.sessionsM.RUnlock()

	if !exists {
		if _, found := ui.cache.Get(sessionID); !found {
			components.GenerationStatus("").Render(r.Context(), w)
			return
		}
	}

	components.GenerationStatus(sessionID).Render(r.Context(), w)
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	faviconBytes, err := os.ReadFile("static/favicon.ico")
	if err != nil {
		log.Printf("favicon error %s", err)
		w.Write([]byte("XXXXXXXXXXXXXXXXXXXXXXX"))
		return
	}
	w.Write(faviconBytes)
}
