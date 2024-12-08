package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	. "github.com/opd-ai/dndbot/srv/handlers"
	"github.com/opd-ai/dndbot/srv/util"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In development, allow all origins
		return true
	},
}

func main() {
	r := mux.NewRouter()

	// Static file serving
	fileServer := http.FileServer(http.FS(StaticFS))
	r.PathPrefix("/static/").Handler(fileServer)

	// API routes
	r.HandleFunc("/", HandleIndex)
	r.HandleFunc("/generate", HandleGenerate).Methods("POST")
	r.HandleFunc("/ws/{sessionID}", HandleWebSocket)
	r.HandleFunc("/download/{sessionID}", HandleDownload).Methods("GET")

	// Enable CORS
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Apply middleware
	handler := corsMiddleware(r)
	r.Use(util.LoggingMiddleware)
	r.Use(util.RecoveryMiddleware)

	// Start cleanup goroutine
	go GlobalSessionManager.CleanupOldSessions()

	// Start server
	port := ":8081"
	util.InfoLogger.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		util.ErrorLogger.Fatal(err)
	}
}
