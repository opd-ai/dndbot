// cmd/server/main.go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/opd-ai/dndbot/srv/ui"
)

func main() {
	// Ensure environment variables are set
	if os.Getenv("CLAUDE_API_KEY") == "" {
		log.Fatal("CLAUDE_API_KEY environment variable is required")
	}

	// Create and configure the generator UI
	generator := ui.NewGeneratorUI()

	// Start the server
	log.Println("Server starting on :3000")
	if err := http.ListenAndServe(":3000", generator); err != nil {
		log.Fatal(err)
	}
}
