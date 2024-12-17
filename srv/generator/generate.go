package generator

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	dndbot "github.com/opd-ai/dndbot/src"
	"github.com/opd-ai/horde"
	// util "github.com/opd-ai/dndbot/srv/util"
)

func GenerateAdventure(progress *GenerationProgress, prompt string) error {
	client := dndbot.NewClaudeClient(os.Getenv("CLAUDE_API_KEY"))
	hordeClient := horde.NewClient(os.Getenv("HORDE_API_KEY"))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Minute)
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
				log.Println("Generating table of Contents")
				progress.UpdateOutput("🎲 Generating table of contents...")
				var err error
				adventure, err = dndbot.GenerateTableOfContents(client, prompt, progress)
				return err
			},
		},
		{
			name: "Creating cover pages",
			function: func() error {
				log.Println("Creating cover pages")
				progress.UpdateOutput("🎨 Creating cover pages...")
				return dndbot.GenerateCoverPrompts(client, &adventure)
			},
		},
		{
			name: "Incremental saving files",
			function: func() error {
				log.Println("Incremental save adventure files")
				progress.UpdateOutput("💾 Incrementally Saving adventure files...")
				return dndbot.SaveToFiles(&adventure, filepath.Join("outputs", progress.SessionID))
			},
		},
		{
			name: "",
			function: func() error {
				log.Println("Generating actual covers")
				progress.UpdateOutput("Generating actual covers...")
				return dndbot.GenerateCoversFromPrompts(hordeClient, &adventure, filepath.Join("outputs", progress.SessionID, "00_Contents"))
			},
		},
		{
			name: "Designing dungeons",
			function: func() error {
				log.Println("Designing dungeon layouts")
				progress.UpdateOutput("🗺️ Designing dungeon layouts...")
				return dndbot.GenerateOnePageDungeons(client, &adventure)
			},
		},
		{
			name: "Incremental saving files",
			function: func() error {
				log.Println("Incremental save adventure files")
				progress.UpdateOutput("💾 Incrementally Saving adventure files...")
				return dndbot.SaveToFiles(&adventure, filepath.Join("outputs", progress.SessionID))
			},
		},
		{
			name: "Expanding adventure content",
			function: func() error {
				log.Println("Expanding adventure content")
				progress.UpdateOutput("📚 Expanding adventure content...")
				return dndbot.ExpandAdventures(client, &adventure, progress)
			},
		},
		{
			name: "Incremental saving files",
			function: func() error {
				log.Println("Incremental save adventure files")
				progress.UpdateOutput("💾 Incrementally Saving adventure files...")
				return dndbot.SaveToFiles(&adventure, filepath.Join("outputs", progress.SessionID))
			},
		},
		{
			name: "Creating illustrations",
			function: func() error {
				log.Println("Creating illustration prompts")
				progress.UpdateOutput("🖼️ Creating illustration prompts...")
				return dndbot.GenerateIllustrationPrompts(client, &adventure)
			},
		},
		{
			name: "",
			function: func() error {
				log.Println("Generating actual illustrations")
				progress.UpdateOutput("Generating actual illustrations...")
				return dndbot.GenerateIllustrationsFromPrompts(hordeClient, &adventure, filepath.Join("outputs", progress.SessionID))
			},
		},
		{
			name: "Reviewing content",
			function: func() error {
				log.Println("Review and adjust content")
				progress.UpdateOutput("⚖️ Reviewing and adjusting content...")
				return dndbot.RemoveCopyrightedMaterial(client, &adventure)
			},
		},
		{
			name: "Saving files",
			function: func() error {
				log.Println("Save adventure files")
				progress.UpdateOutput("💾 Saving adventure files...")
				return dndbot.SaveToFiles(&adventure, filepath.Join("outputs", progress.SessionID))
			},
		},
		{
			name: "Generating zip",
			function: func() error {
				log.Println("Generating zip file")
				progress.UpdateOutput("💾 Generating zip file...")
				zipPath, err := ZipOutputDirectory(filepath.Join("outputs", progress.SessionID))
				if err != nil {
					return err
				}
				zipHref := fmt.Sprintf("<font size=\"5\">  <a href=\"%s\">Download your archived adventure</a>  </font>", zipPath)
				zipMessage := fmt.Sprintf("💾 Adventure generatation complete! %s", zipHref)
				progress.UpdateOutput(zipMessage)
				return nil
			},
		},
	}

	// Execute each step with error handling and progress updates
	for x, step := range steps {
		select {
		case <-ctx.Done():
			log.Printf("Generation timeout during step: %d", x)
			return fmt.Errorf("generation timed out during %s", step.name)
		default:
			if err := step.function(); err != nil {
				errMsg := fmt.Sprintf("❌ Error during %s: %v", step.name, err)
				log.Println(errMsg)
				progress.UpdateOutput(errMsg)
				return fmt.Errorf("failed during %s: %w", step.name, err)
			}
		}
	}

	// Send completion message
	progress.UpdateOutput("✨ Adventure generation completed successfully!")
	return nil
}

func ZipOutputDirectory(outDir string) (zipPath string, err error) {
	zipPath = outDir + ".zip"
	file, err := os.Create(zipPath)
	if err != nil {
		return
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if filepath.IsAbs(path) {
			return fmt.Errorf("absolute path error: %s", path)
		}
		fmt.Printf("Crawling: %#v\n", path)
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("error crawling file: %s err: %s", path, err)
		}
		defer file.Close()

		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	err = filepath.Walk(outDir, walker)
	if err != nil {
		return
	}
	return
}
