package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/opd-ai/dndbot/src"
)

func (s *Server) generateAdventure(order *Order) {
	defer close(order.LogChan)

	order.LogChan <- "Starting adventure generation..."

	// Create output directory for this order
	outputDir := filepath.Join("adventures", order.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		order.Status = "error"
		order.LogChan <- "Error creating output directory: " + err.Error()
		return
	}

	// Generate the complete adventure structure
	var adventure Adventure
	var err error

	// Step 1: Generate Table of Contents
	order.LogChan <- "Generating table of contents..."
	adventure, err = GenerateTableOfContents(s.claude, order.PromptText)
	if err != nil {
		order.Status = "error"
		order.LogChan <- fmt.Sprintf("Error generating table of contents: %v", err)
		return
	}
	order.LogChan <- "Table of contents complete."

	// Step 2: Generate Cover Pages
	order.LogChan <- "Generating cover pages..."
	if err := GenerateCoverPrompts(s.claude, &adventure); err != nil {
		order.Status = "error"
		order.LogChan <- fmt.Sprintf("Error generating cover pages: %v", err)
		return
	}
	order.LogChan <- "Cover pages complete."

	// Step 3: Generate One-Page Dungeons
	order.LogChan <- "Generating dungeon layouts..."
	if err := GenerateOnePageDungeons(s.claude, &adventure); err != nil {
		order.Status = "error"
		order.LogChan <- fmt.Sprintf("Error generating dungeons: %v", err)
		return
	}
	order.LogChan <- "Dungeon layouts complete."

	// Step 4: Expand Adventures
	order.LogChan <- "Expanding adventure content..."
	if err := ExpandAdventures(s.claude, &adventure); err != nil {
		order.Status = "error"
		order.LogChan <- fmt.Sprintf("Error expanding adventures: %v", err)
		return
	}
	order.LogChan <- "Adventure content expansion complete."

	// Step 5: Generate Illustration Prompts
	order.LogChan <- "Generating illustration prompts..."
	if err := GenerateIllustrationPrompts(s.claude, &adventure); err != nil {
		order.Status = "error"
		order.LogChan <- fmt.Sprintf("Error generating illustration prompts: %v", err)
		return
	}
	order.LogChan <- "Illustration prompts complete."

	// Step 6: Remove Copyrighted Material
	order.LogChan <- "Reviewing and removing copyrighted material..."
	if err := RemoveCopyrightedMaterial(s.claude, &adventure); err != nil {
		order.Status = "error"
		order.LogChan <- fmt.Sprintf("Error removing copyrighted material: %v", err)
		return
	}
	order.LogChan <- "Copyright review complete."

	// Step 7: Save all files
	order.LogChan <- "Saving adventure files..."
	if err := SaveToFiles(&adventure, outputDir); err != nil {
		order.Status = "error"
		order.LogChan <- fmt.Sprintf("Error saving files: %v", err)
		return
	}

	// Create ZIP file of the adventure
	zipPath := filepath.Join(outputDir, "adventure.zip")
	if err := createZip(outputDir, zipPath); err != nil {
		order.Status = "error"
		order.LogChan <- fmt.Sprintf("Error creating ZIP file: %v", err)
		return
	}

	// Update order status
	order.OutputPath = zipPath
	order.Status = "complete"
	order.CompletedAt = time.Now()
	order.LogChan <- "Adventure generation complete! You can now download your adventure."
}
