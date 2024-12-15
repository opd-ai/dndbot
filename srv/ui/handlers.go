// Package ui provides the web user interface handlers for the DND bot generator
package ui

import (
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

//go:embed templates/index.html
var index []byte

// handleHome handles requests to the root endpoint, rendering the main application layout
// and generator form.
//
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response
//   - r: *http.Request containing the incoming request details
func (ui *GeneratorUI) handleHome(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("html").Parse(string(index))
	if err != nil {
		panic(err)
	}
	t.Execute(w, "")
}

// handleGetMessages retrieves and formats message history for a given session.
//
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response
//   - r: *http.Request containing the incoming request details
//
// The function extracts the sessionID from URL parameters, looks up the message history,
// and returns formatted messages as HTML. Returns empty string if session not found.
//
// Related: formatMessages()
func (ui *GeneratorUI) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionID")

	ui.sessionsM.RLock()
	history, exists := ui.msgHistory[sessionID]
	ui.sessionsM.RUnlock()

	if !exists {
		w.Write([]byte(""))
		return
	}
	log.Println("Got message history", history)

	messages := history.GetMessages()
	log.Println("Message history content", messages)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(formatMessages(messages)))
}

// handleCheckSession validates and checks the existence of a session.
//
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response
//   - r: *http.Request containing the incoming request details
//
// The function attempts to get sessionID from X-Session-Id header or session_id cookie.
// Validates the session and checks if it exists in memory or cache.
// Renders appropriate generation status based on session validity.
//
// Related: isValidSession()
func (ui *GeneratorUI) handleCheckSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-Id")
	if sessionID == "" {
		// Try cookie as fallback
		if cookie, err := r.Cookie("session_id"); err == nil {
			sessionID = cookie.Value
		}
	}

	if !isValidSession(sessionID) {
		//components.GenerationStatus("").Render(r.Context(), w)
		return
	}

	// Check if session exists in memory or cache
	ui.sessionsM.RLock()
	_, exists := ui.sessions[sessionID]
	ui.sessionsM.RUnlock()

	if !exists {
		if _, found := ui.cache.Get(sessionID); !found {
			//components.GenerationStatus("").Render(r.Context(), w)
			return
		}
	}

	//components.GenerationStatus(sessionID).Render(r.Context(), w)
}

// handleFavicon serves the favicon.ico file from the static directory.
//
// Parameters:
//   - w: http.ResponseWriter to write the HTTP response
//   - r: *http.Request containing the incoming request details
//
// Returns the favicon.ico file contents or a placeholder if file cannot be read.
// Logs any errors encountered when reading the favicon file.
func handleFavicon(w http.ResponseWriter, r *http.Request) {
	faviconBytes, err := os.ReadFile("static/favicon.ico")
	if err != nil {
		log.Printf("favicon error %s", err)
		w.Write([]byte("XXXXXXXXXXXXXXXXXXXXXXX"))
		return
	}
	w.Write(faviconBytes)
}
