// main.go
package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	. "github.com/opd-ai/dndbot/src"
	dndbot "github.com/opd-ai/dndbot/src"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, implement proper origin checking
	},
}

type GenerationProgress struct {
	SessionID string
	WSConn    *websocket.Conn
	Done      chan bool
}

var activeGenerations = make(map[string]*GenerationProgress)

func main() {
	r := mux.NewRouter()

	// Routes
	r.HandleFunc("/", handleIndex)
	r.HandleFunc("/generate", handleGenerate).Methods("POST")
	r.HandleFunc("/ws/{sessionID}", handleWebSocket)

	// Serve static files
	fileServer := http.FileServer(http.FS(staticFS))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))

	log.Printf("Server starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	sessionID := uuid.New().String()
	prompt := r.FormValue("prompt")

	// Create progress tracker
	progress := &GenerationProgress{
		SessionID: sessionID,
		Done:      make(chan bool),
	}
	activeGenerations[sessionID] = progress

	// Start generation in background
	go generateAdventure(sessionID, prompt)

	// Return session ID to client
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"sessionId": "` + sessionID + `"}`))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	progress, exists := activeGenerations[sessionID]
	if !exists {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	progress.WSConn = conn

	// Wait for generation to complete
	<-progress.Done
	delete(activeGenerations, sessionID)
}

func generateAdventure(sessionID string, prompt string) {
	progress := activeGenerations[sessionID]
	if progress == nil || progress.WSConn == nil {
		return
	}

	sendProgress := func(msg string) {
		progress.WSConn.WriteMessage(websocket.TextMessage, []byte(msg))
	}

	// Initialize Claude client
	client := NewClaudeClient(os.Getenv("CLAUDE_API_KEY"))

	sendProgress("🎲 Generating table of contents...")
	adventure, err := dndbot.GenerateTableOfContents(client, prompt)
	if err != nil {
		sendProgress("❌ Error generating table of contents: " + err.Error())
		progress.Done <- true
		return
	}

	sendProgress("📝 Generating cover art...")
	if err := dndbot.GenerateCoverPrompts(client, &adventure); err != nil {
		sendProgress("❌ Error generating cover pages: " + err.Error())
		progress.Done <- true
		return
	}

	sendProgress("📝 Generating one-page dungeons...")
	if err := dndbot.GenerateOnePageDungeons(client, &adventure); err != nil {
		sendProgress("❌ Error generating one-page dungeons: " + err.Error())
		progress.Done <- true
		return
	}

	sendProgress("📚 Generating expanded adventures...")
	if err := dndbot.ExpandAdventures(client, &adventure); err != nil {
		sendProgress("❌ Error expanding adventures: " + err.Error())
		progress.Done <- true
		return
	}

	sendProgress("📝 Generating cover art...")
	if err := dndbot.GenerateIllustrationPrompts(client, &adventure); err != nil {
		sendProgress("❌ Error generating illustration prompts: " + err.Error())
		progress.Done <- true
		return
	}

	sendProgress("📝 Generating cover art...")
	if err := dndbot.RemoveCopyrightedMaterial(client, &adventure); err != nil {
		sendProgress("❌ Error removing copyrighted material: " + err.Error())
		progress.Done <- true
		return
	}

	sendProgress("💾 Saving files...")
	if err := SaveToFiles(&adventure, "output/"+sessionID); err != nil {
		sendProgress("❌ Error saving files: " + err.Error())
		progress.Done <- true
		return
	}

	sendProgress("✅ Adventure generation complete!")
	progress.Done <- true
}
