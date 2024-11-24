// processor.go
package main

import "fmt"

func generateTableOfContents(client *ClaudeClient, prompt string) (Adventure, error) {
	systemPrompt := `Create a D&D adventure series table of contents based on the following prompt.
    For each episode include:
    - Title
    - Summary (including plot and location)
    - Tagline
    - Main characters
    Format as structured markdown.`

	response, err := client.SendMessage(systemPrompt, prompt)
	if err != nil {
		return Adventure{}, fmt.Errorf("generating ToC: %w", err)
	}

	// Parse the response into Adventure struct
	adventure := Adventure{
		OriginalPrompt:  prompt,
		TableOfContents: response,
	}

	// Parse episodes from the response
	adventure.Episodes = parseEpisodes(response)

	return adventure, nil
}

func generateOnePageDungeons(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Convert this episode into a one-page dungeon format:\n%s",
			adventure.Episodes[i].Summary)

		response, err := client.SendMessage(getOnePageDungeonPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("generating one-page dungeon for episode %d: %w", i, err)
		}

		adventure.Episodes[i].OnePageDungeon = response
	}
	return nil
}

func expandAdventures(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Expand this one-page dungeon into a detailed 10-page adventure:\n%s",
			adventure.Episodes[i].OnePageDungeon)

		response, err := client.SendMessage(getExpandedAdventurePrompt(), prompt)
		if err != nil {
			return fmt.Errorf("expanding episode %d: %w", i, err)
		}

		adventure.Episodes[i].FullAdventure = response
	}
	return nil
}

func generateIllustrationPrompts(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Generate illustration prompts for this adventure:\n%s",
			adventure.Episodes[i].FullAdventure)

		response, err := client.SendMessage(getIllustrationPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("generating illustration prompts for episode %d: %w", i, err)
		}

		adventure.Episodes[i].Illustrations = parseIllustrationPrompts(response)
	}
	return nil
}

func removeCopyrightedMaterial(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Remove any copyrighted material from this adventure:\n%s",
			adventure.Episodes[i].FullAdventure)

		response, err := client.SendMessage(getCopyrightRemovalPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("removing copyrighted material from episode %d: %w", i, err)
		}

		adventure.Episodes[i].FullAdventure = response
	}
	return nil
}
