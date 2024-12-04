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
)

var (
	activeGenerations = make(map[string]*GenerationProgress)
	generationsMutex  sync.RWMutex
)

func createNewGeneration(sessionID string) *GenerationProgress {
	progress := &GenerationProgress{
		SessionID: sessionID,
		Done:      make(chan bool),
		StartTime: time.Now(),
		State:     StateInitialized,
	}

	generationsMutex.Lock()
	activeGenerations[sessionID] = progress
	generationsMutex.Unlock()

	return progress
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
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
	progress := createNewGeneration(sessionID)

	// Start generation in background
	go func() {
		if err := generateAdventure(sessionID, prompt); err != nil {
			ErrorLogger.Printf("Generation failed for session %s: %v", sessionID, err)
			progress.UpdateState(StateError)
			progress.Error = err
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"sessionId": sessionID,
		"status":    "started",
	})
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		ErrorLogger.Printf("Template parsing error: %v", err)
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, nil); err != nil {
		ErrorLogger.Printf("Template execution error: %v", err)
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
	}
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	filePath := filepath.Join("output", sessionID, "adventure.zip")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ErrorLogger.Printf("Download file not found: %s", filePath)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=adventure-%s.zip", sessionID))
	w.Header().Set("Content-Type", "application/zip")
	http.ServeFile(w, r, filePath)
	InfoLogger.Printf("File downloaded: %s", filePath)
}
