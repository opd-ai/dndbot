// cmd/server/main.go
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/opd-ai/dndbot/srv/ui"
	wileedot "github.com/opd-ai/wileedot"
)

var (
	paywall = flag.Bool("paywall", false, "paywall output")
	tls     = flag.Bool("tls", false, "auto-generate TLS certificate")
)

func main() {
	flag.Parse()
	// Ensure environment variables are set
	if os.Getenv("CLAUDE_API_KEY") == "" {
		log.Fatal("CLAUDE_API_KEY environment variable is required")
	}

	// Create and configure the generator UI
	generator := ui.NewGeneratorUI(*paywall)

	cfg := wileedot.Config{
		Domain:         "localhost",
		AllowedDomains: []string{"localhost"},
		CertDir:        "./",
		Email:          "example@example.com",
	}

	var listener net.Listener

	if *tls {
		var err error
		listener, err = wileedot.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		var err error
		listener, err = net.Listen("tcp", "localhost:")
		if err != nil {
			log.Fatal(err)
		}
	}

	// Start the server
	log.Println("Server starting on", listener.Addr())
	if err := http.Serve(listener, generator); err != nil {
		log.Fatal(err)
	}
}
