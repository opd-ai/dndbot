package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"log"

	"github.com/gorilla/websocket"
	dndbot "github.com/opd-ai/dndbot/src"
	//util "github.com/opd-ai/dndbot/srv/util"
)

func GenerateAdventure(progress *GenerationProgress, prompt string) error {
	client := dndbot.NewClaudeClient(os.Getenv("CLAUDE_API_KEY"))
	// Helper function to send WebSocket updates
	sendUpdate := func(message string) {
		if progress.WSConn != nil {
			progress.mu.Lock()
			defer progress.mu.Unlock()
			if err := progress.WSConn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Println("Failed to send WebSocket message: %v", err)
			}
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Initialize adventure structure
	var adventure dndbot.Adventure

	// Define generation steps
	steps := []struct {
		name     string
		function func() error
	}{
		{
			name: "Generating table of contents",
			function: func() error {
				sendUpdate("🎲 Generating table of contents...")
				var err error
				adventure, err = dndbot.GenerateTableOfContents(client, prompt)
				return err
			},
		},
		{
			name: "Creating cover pages",
			function: func() error {
				sendUpdate("🎨 Creating cover pages...")
				return dndbot.GenerateCoverPrompts(client, &adventure)
			},
		},
		{
			name: "Designing dungeons",
			function: func() error {
				sendUpdate("🗺️ Designing dungeon layouts...")
				return dndbot.GenerateOnePageDungeons(client, &adventure)
			},
		},
		{
			name: "Expanding adventure content",
			function: func() error {
				sendUpdate("📚 Expanding adventure content...")
				return dndbot.ExpandAdventures(client, &adventure)
			},
		},
		{
			name: "Creating illustrations",
			function: func() error {
				sendUpdate("🖼️ Creating illustration prompts...")
				return dndbot.GenerateIllustrationPrompts(client, &adventure)
			},
		},
		{
			name: "Reviewing content",
			function: func() error {
				sendUpdate("⚖️ Reviewing and adjusting content...")
				return dndbot.RemoveCopyrightedMaterial(client, &adventure)
			},
		},
		{
			name: "Saving files",
			function: func() error {
				sendUpdate("💾 Saving adventure files...")
				return dndbot.SaveToFiles(&adventure, filepath.Join("outputs", progress.SessionID))
			},
		},
	}

	// Execute each step with error handling and progress updates
	for _, step := range steps {
		select {
		case <-ctx.Done():
			return fmt.Errorf("generation timed out during %s", step.name)
		default:
			if err := step.function(); err != nil {
				errMsg := fmt.Sprintf("❌ Error during %s: %v", step.name, err)
				sendUpdate(errMsg)
				return fmt.Errorf("failed during %s: %w", step.name, err)
			}
		}
	}

	// Send completion message
	sendUpdate("✨ Adventure generation completed successfully!")
	return nil
}
