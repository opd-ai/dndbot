package util

import (
	"log"
	"net/http"
	"os"
	"time"
)

// Logger setup
var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	// Initialize loggers
	logFile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}

	InfoLogger = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll("output", 0o755); err != nil {
		log.Fatal("Failed to create output directory:", err)
	}
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		InfoLogger.Printf("Request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		InfoLogger.Printf("Request completed in %v", time.Since(start))
	})
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				ErrorLogger.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
