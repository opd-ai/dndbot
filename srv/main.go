// cmd/server/main.go
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/opd-ai/dndbot/srv/ui"
)

var paywall = flag.Bool("paywall", false, "paywall output")

func main() {
	flag.Parse()
	// Ensure environment variables are set
	if os.Getenv("CLAUDE_API_KEY") == "" {
		log.Fatal("CLAUDE_API_KEY environment variable is required")
	}

	// Create and configure the generator UI
	generator := ui.NewGeneratorUI(*paywall)

	// Start the server
	log.Println("Server starting on :3000")
	if err := http.ListenAndServe(":3000", generator); err != nil {
		log.Fatal(err)
	}
}
