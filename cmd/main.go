package main

import (
	"flag"
	"fmt"
	"os"

	dndbot "github.com/opd-ai/dndbot/src"
)

var (
	directory = flag.String("dirname", "01-Adventure", "Name of the output directory for the adventure")
	setting   = flag.String("setting", "SETTING.md", "a file containing the details of the campaign setting")
)

// main.go
func main() {
	flag.Parse()
	config := dndbot.Config{
		APIKey:     os.Getenv("CLAUDE_API_KEY"),
		OutputDir:  *directory,
		MaxRetries: 3,
	}

	if config.APIKey == "" {
		fmt.Println("Please set CLAUDE_API_KEY environment variable")
		os.Exit(1)
	}

	client := dndbot.NewClaudeClient(config.APIKey)
	var prompt string
	if len(os.Args) <= 1 {
		var err error
		promptb, err := os.ReadFile("PROMPT.md")
		if err != nil {
			fmt.Println("Please provide a narrative prompt or PROMPT.md")
			os.Exit(1)
		}
		prompt = string(promptb)
	} else {
		prompt = os.Args[1]
		if prompt == "" {
			fmt.Println("Please provide a narrative prompt")
			os.Exit(1)
		}
	}
	// Process the adventure
	adventure, err := dndbot.GenerateTableOfContents(client, prompt)
	if err != nil {
		fmt.Printf("Error generating table of contents: %v\n", err)
		os.Exit(1)
	}

	if err := dndbot.GenerateCoverPrompts(client, &adventure); err != nil {
		fmt.Printf("Error generating cover pages %v\n", err)
		os.Exit(1)
	}

	if err := dndbot.GenerateOnePageDungeons(client, &adventure); err != nil {
		fmt.Printf("Error generating one-page dungeons: %v\n", err)
		os.Exit(1)
	}

	if err := dndbot.ExpandAdventures(client, &adventure); err != nil {
		fmt.Printf("Error expanding adventures: %v\n", err)
		os.Exit(1)
	}

	if err := dndbot.GenerateIllustrationPrompts(client, &adventure); err != nil {
		fmt.Printf("Error generating illustration prompts: %v\n", err)
		os.Exit(1)
	}

	if err := dndbot.RemoveCopyrightedMaterial(client, &adventure); err != nil {
		fmt.Printf("Error removing copyrighted material: %v\n", err)
		os.Exit(1)
	}

	if err := dndbot.SaveToFiles(&adventure, config.OutputDir); err != nil {
		fmt.Printf("Error saving files: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Adventure generation complete!")
}
