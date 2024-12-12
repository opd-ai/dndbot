package ui

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/opd-ai/dndbot/srv/components"
)

func (ui *GeneratorUI) handleHome(w http.ResponseWriter, r *http.Request) {
	components.Layout().Render(r.Context(), w)
	components.GeneratorForm().Render(r.Context(), w)
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
