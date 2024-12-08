package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	generator "github.com/opd-ai/dndbot/srv/generator"
	"github.com/opd-ai/dndbot/srv/util"
)

var (
	activeGenerations = make(map[string]*generator.GenerationProgress)
	generationsMutex  sync.RWMutex
)

func createNewGeneration(sessionID string) *generator.GenerationProgress {
	progress := &generator.GenerationProgress{
		SessionID: sessionID,
		Done:      make(chan bool),
		StartTime: time.Now(),
		State:     generator.StateInitialized,
	}

	generationsMutex.Lock()
	activeGenerations[sessionID] = progress
	generationsMutex.Unlock()

	return progress
}

// handlers.go

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	// 1. Method validation
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Parse form data
	if err := r.ParseForm(); err != nil {
		util.ErrorLogger.Printf("Failed to parse form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// 3. Get and validate prompt
	prompt := r.FormValue("prompt")
	if prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	// 4. Create session with UUID
	sessionID := uuid.New().String()
	progress := GlobalSessionManager.CreateSession(sessionID)

	// 5. Prepare JSON response
	response := struct {
		SessionID string `json:"sessionId"`
		Status    string `json:"status"`
	}{
		SessionID: sessionID,
		Status:    "initialized",
	}

	// 6. Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		util.ErrorLogger.Printf("Failed to encode response: %v", err)
		return
	}

	// 7. Start generation in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				util.ErrorLogger.Printf("Panic in generation goroutine: %v", r)
				progress.UpdateState(generator.StateError)
				progress.Error = fmt.Errorf("internal server error")
				close(progress.Done)
			}
		}()

		progress.UpdateState(generator.StateGenerating)

		// Call your generation function
		if err := generator.GenerateAdventure(progress, prompt); err != nil {
			util.ErrorLogger.Printf("Generation failed for session %s: %v", sessionID, err)
			progress.UpdateState(generator.StateError)
			progress.Error = err

			// Notify client of error if WebSocket is connected
			if progress.WSConn != nil {
				errMsg := fmt.Sprintf("Generation failed: %v", err)
				progress.WSConn.WriteMessage(websocket.TextMessage, []byte(errMsg))
			}
		} else {
			progress.UpdateState(generator.StateCompleted)
		}

		close(progress.Done)
	}()
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		util.ErrorLogger.Printf("Template parsing error: %v", err)
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		util.ErrorLogger.Printf("Template execution error: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	filePath := filepath.Join("output", sessionID, "adventure.zip")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		util.ErrorLogger.Printf("Download file not found: %s", filePath)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=adventure-%s.zip", sessionID))
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, filePath)
	util.InfoLogger.Printf("File downloaded: %s", filePath)
}
