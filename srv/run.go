package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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
	fileServer := http.FileServer(http.FS(staticFS))
	r.PathPrefix("/static/").Handler(fileServer)

	// API routes
	r.HandleFunc("/", handleIndex)
	r.HandleFunc("/generate", handleGenerate).Methods("POST")
	r.HandleFunc("/ws/{sessionID}", handleWebSocket)
	r.HandleFunc("/download/{sessionID}", handleDownload).Methods("GET")
	r.HandleFunc("/health", handleHealthCheck).Methods("GET")

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
	r.Use(loggingMiddleware)
	r.Use(recoveryMiddleware)

	// Start cleanup goroutine
	go cleanupOldSessions()

	// Start server
	port := ":8081"
	InfoLogger.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		ErrorLogger.Fatal(err)
	}
	//if err := ListenAndServeTLS(port, "cert.pem", "key.pem", handler); err != nil {
	//	ErrorLogger.Fatal(err)
	//}
}
