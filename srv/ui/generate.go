// Package ui provides the web user interface handlers for the DND bot generator
package ui

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/opd-ai/dndbot/srv/generator"
)

// handleGenerate processes adventure generation requests and manages the generation session.
//
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response
//   - r: *http.Request containing the form data with the 'prompt' field
//
// The function:
//   - Creates a new session with UUID
//   - Sets up session cookies and headers
//   - Initializes generation progress tracking
//   - Starts asynchronous adventure generation
//
// Error cases:
//   - Returns 400 if form parsing fails
//   - Returns 400 if prompt is empty
//   - Logs and handles generation errors via progress updates
//
// Related types:
//   - generator.GenerationProgress
//   - MessageHistory
//
// The generation process runs asynchronously and updates are tracked through
// the GenerationProgress object. Client can monitor progress via WebSocket
// connection using the provided session ID.
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

	setting := r.FormValue("setting")
	style := r.FormValue("style")

	// Create new session
	sessionID := r.Header.Get("X-Session-Id")
	if sessionID == "" {
		log.Println("no client side sessionID:", sessionID)
		// Try cookie as fallback
		if cookie, err := r.Cookie("session_id"); err == nil && cookie.Value != "null" {
			log.Println("cookie found", cookie.Value, err)
			sessionID = cookie.Value
		} else {
			sessionID = uuid.New().String()
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    sessionID,
				Path:     "/",
				MaxAge:   864000,
				HttpOnly: false,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}
	w.Header().Set("X-Session-Id", sessionID)
	if ui.historyCheck(sessionID) {
		w.Write([]byte(""))
		return
	}

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
			Messages: make([]generator.Message, 0),
		}
	}
	ui.sessionsM.Unlock()

	// Start generation immediately, don't wait for WebSocket
	go func() {
		log.Printf("[Session %s] Starting generation", sessionID)
		if err := generator.GenerateAdventure(progress, prompt, setting, style); err != nil {
			log.Printf("[Session %s] Generation error: %v", sessionID, err)
			progress.UpdateState(generator.StateError)
			progress.SendUpdate(fmt.Sprintf("Error: %v", err))
		}
	}()

	// components.GenerationStatus(sessionID).Render(r.Context(), w)
}
