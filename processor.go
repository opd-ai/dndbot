// processor.go
package main

import (
	"fmt"
	"log"
	"strings"
)

func generateTableOfContents(client *ClaudeClient, prompt string) (Adventure, error) {
	systemPrompt := `Create a Role-Playing Game adventure series table of contents based on the following prompt.
    For each episode include:
    - Title
    - Summary (including plot and location)
    - Tagline
    - Main non-player characters

	The main plot should extend from the first episode to the last episode.
	Each episode should also include a unique side-plot.
	Prefer a relatable sense of realismi.
	Fantasy is acceptable, but avoiding material circumstances is not.
	Avoid overt flights of fancy.
	Maintain verisimilitude throughout the story.

    Format as structured markdown.
	Avoid the direct use of copyrighted material and characters.
	Avoid the use of real places.

	Do this without asking for confirmation or direction.
	Do not ask for confirmation in any way, just output the complete adventure.
	This is essential.

	Follow this example format exactly for each consecutive episode:`
	systemPrompt += "```\n"
	systemPrompt += `## Episode: Number - Episode Title
Summary: 8 sentence summary of the adventure, setting, plot, and mood.
Tagline: Catchy one-sentence quote about the adventure
Location: Location name, 2-3 sentence location description
Characters: Character One, Character Two, Characther Three...

`
	systemPrompt += "```\n"

	response, err := client.SendMessage(systemPrompt, prompt)
	if err != nil {
		return Adventure{}, fmt.Errorf("generating ToC: %w", err)
	}
	log.Println(response)

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
		prompt := fmt.Sprintf("Expand this episode description into a one-page dungeon format:\n%s\n",
			adventure.Episodes[i].Text())
		if (i - 1) > 0 {
			prompt += fmt.Sprintf("There was a previous adventure in this series. Here is summary of the previous adventure:\n%s\n",
				adventure.Episodes[i-1].Text())
		}
		prompt += fmt.Sprintf("The original prompt provided by a human for this story arc was: \n%s\n", adventure.OriginalPrompt)

		response, err := client.SendMessage(getOnePageDungeonPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("generating one-page dungeon for episode %d: %w", i, err)
		}
		log.Println(response)
		if err := saveToFiles(adventure, "tmp"); err != nil {
			return fmt.Errorf("writing Episode %d %w", i, err)
		}

		adventure.Episodes[i].OnePageDungeon = response
	}
	return nil
}

func expandAdventures(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		// Build initial prompt
		prompt := fmt.Sprintf("Expand this one-page dungeon into a detailed 8 page(about 600 lines) adventure:\n%s\n",
			adventure.Episodes[i].OnePageDungeon)
		if (i - 1) > 0 {
			prompt += fmt.Sprintf("There was a previous adventure in this series. Here is summary of the previous adventure:\n%s\n",
				adventure.Episodes[i-1].Text())
		}

		// Start with empty adventure text
		adventure.Episodes[i].FullAdventure = ""

		currentPrompt := prompt
		for {
			response, err := client.SendMessage(getExpandedAdventurePrompt(), currentPrompt)
			if err != nil {
				return fmt.Errorf("expanding episode %d: %w", i, err)
			}

			// Append new response
			if adventure.Episodes[i].FullAdventure != "" {
				adventure.Episodes[i].FullAdventure += "\n\n"
			}
			adventure.Episodes[i].FullAdventure += response
			log.Println(adventure.Episodes[i].FullAdventure)

			// Check if Claude indicates it's continuing
			if !strings.Contains(strings.ToLower(response), "continue") {
				break
			}

			if err := saveToFiles(adventure, "tmp"); err != nil {
				return fmt.Errorf("writing Episode %d %w", i, err)
			}

			// Update prompt for continuation
			currentPrompt = "Please continue from where you left off:\n" + response
		}
	}

	return nil
}

func generateIllustrationPrompts(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Generate illustration prompts for this adventure:\n%s\n",
			adventure.Episodes[i].FullAdventure)

		response, err := client.SendMessage(getIllustrationPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("generating illustration prompts for episode %d: %w", i, err)
		}
		log.Println(response)
		adventure.Episodes[i].Illustrations = parseIllustrationPrompts(response)
		if err := saveToFiles(adventure, "tmp"); err != nil {
			return fmt.Errorf("writing Episode %d %w", i, err)
		}
	}
	return nil
}

func generateCoverPrompts(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Generate cover illustration prompts for this adventure:\n%s\n",
			adventure.TableOfContents)

		response, err := client.SendMessage(getIllustrationPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("generating illustration prompts for cover %d: %w", i, err)
		}

		adventure.Covers = parseIllustrationPrompts(response)
		if err := saveToFiles(adventure, "tmp"); err != nil {
			return fmt.Errorf("writing Episode %d %w", i, err)
		}
	}
	return nil
}

func removeCopyrightedMaterial(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		// Build initial prompt
		prompt := fmt.Sprintf("Remove any copyrighted material from this adventure:\n%s",
			adventure.Episodes[i].FullAdventure)

		currentPrompt := prompt
		for {
			response, err := client.SendMessage(getCopyrightRemovalPrompt(), currentPrompt)
			if err != nil {
				return fmt.Errorf("editing episode %d: %w", i, err)
			}

			// Append new response
			if adventure.Episodes[i].FullAdventure != "" {
				adventure.Episodes[i].FullAdventure += "\n\n"
			}
			adventure.Episodes[i].FullAdventure += response

			// Check if Claude indicates it's continuing
			if !strings.Contains(strings.ToLower(response), "continue") {
				if err := saveToFiles(adventure, "tmp"); err != nil {
					return fmt.Errorf("writing Episode %d %w", i, err)
				}
				break
			}

			// Update prompt for continuation
			currentPrompt = "Please continue from where you left off:\n" + response
		}
	}

	return nil
}
