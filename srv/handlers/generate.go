package handlers

import (
	"fmt"
	"net/http"

	generator "github.com/opd-ai/dndbot/srv/generator"
	util "github.com/opd-ai/dndbot/srv/util"
)

func HandleGenerate(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the form data
	if err := r.ParseForm(); err != nil {
		util.ErrorLogger.Printf("Form parsing error: %v", err)
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	// Get the prompt from the form
	prompt := r.FormValue("prompt")
	if prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// Generate a unique session ID
	sessionID := util.GenerateUUID() // Assuming this exists in util package

	// Create a new session
	progress := GlobalSessionManager.CreateSession(sessionID)

	// Start the generation process in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				util.ErrorLogger.Printf("Panic in generation goroutine: %v", r)
				progress.UpdateState(generator.StateError)
				progress.Error = fmt.Errorf("internal server error: %v", r)
			}
		}()

		if err := generator.GenerateAdventure(progress, prompt); err != nil {
			util.ErrorLogger.Printf("Generation error for session %s: %v", sessionID, err)
			progress.UpdateState(generator.StateError)
			progress.Error = err
			return
		}

		progress.UpdateState(generator.StateCompleted)
	}()

	// Return the session ID to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, `{"sessionId": "%s"}`, sessionID)
}
