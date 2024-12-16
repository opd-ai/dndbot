package dndbot

import (
	"fmt"
	"log"
	"strings"
)

func GenerateTableOfContents(client *ClaudeClient, prompt string, p progressor) (Adventure, error) {
	var pr progressor
	if p != nil {
		pr = p
	} else {
		pr = &nullProgressor{}
	}
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
Summary: 8 sentence summary of the adventure, setting, plot, and mood. (All one line)
Tagline: Catchy one-sentence quote about the adventure (All one line)
Location: Location name, 2-3 sentence location description (All one line)
Characters: Character One, Character Two, Characther Three... (All one line)

`
	systemPrompt += "```\n"
	systemPrompt += getSettingDetails()

	response, err := client.SendMessage(systemPrompt, "This is the story prompt, it is very important that you follow this prompt:"+prompt)
	if err != nil {
		return Adventure{}, fmt.Errorf("generating ToC: %w", err)
	}
	log.Println(response)
	pr.UpdateOutput(response)

	// Parse the response into Adventure struct
	adventure := Adventure{
		OriginalPrompt:  prompt,
		TableOfContents: response,
	}

	// Parse episodes from the response
	adventure.Episodes = parseEpisodes(response)

	return adventure, nil
}

func GenerateOnePageDungeons(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Expand this episode description into a one-page dungeon format:\n%s\n",
			adventure.Episodes[i].Text())
		if (i - 1) > 0 {
			prompt += fmt.Sprintf("There was a previous adventure in this series. Here is summary of the previous adventure:\n%s\n",
				adventure.Episodes[i-1].Text())
		}
		prompt += fmt.Sprintf("The original prompt provided by a human for this story arc was: \n%s\n", adventure.OriginalPrompt)

		response, err := client.SendMessage(GetOnePageDungeonPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("generating one-page dungeon for episode %d: %w", i, err)
		}
		log.Println(response)
		if err := SaveToFiles(adventure, "tmp"); err != nil {
			return fmt.Errorf("writing Episode %d %w", i, err)
		}

		adventure.Episodes[i].OnePageDungeon = response
	}
	return nil
}

type progressor interface {
	UpdateOutput(message string)
}

type nullProgressor struct{}

func (n nullProgressor) UpdateOutput(message string) {
	return
}

func ExpandAdventures(client *ClaudeClient, adventure *Adventure, p progressor) error {
	var pr progressor
	if p != nil {
		pr = p
	} else {
		pr = &nullProgressor{}
	}
	for i := range adventure.Episodes {
		// Build initial prompt
		prompt := fmt.Sprintf("Expand this one-page dungeon into a detailed 8 page(about 600 lines) adventure:\n%s\n",
			adventure.Episodes[i].OnePageDungeon)
		if (i - 1) > 0 {
			prompt += fmt.Sprintf("There was a previous adventure in this series. Here is summary of the previous adventure:\n%s\n",
				adventure.Episodes[i-1].Text())
		}
		msgUpd := fmt.Sprintf("Working on: %s ", adventure.Episodes[i].Title)
		pr.UpdateOutput(msgUpd)

		// Start with empty adventure text
		adventure.Episodes[i].FullAdventure = ""

		currentPrompt := prompt
		index := 0
		for {
			msgUpd := fmt.Sprintf("Working on: %s section %d", adventure.Episodes[i].Title, index)
			pr.UpdateOutput(msgUpd)
			index++
			response, err := client.SendMessage(GetExpandedAdventurePrompt()+getWritingStyleDetails(), currentPrompt)
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

			if err := SaveToFiles(adventure, "tmp"); err != nil {
				return fmt.Errorf("writing Episode %d %w", i, err)
			}
			// Update prompt for continuation
			currentPrompt = "Please continue from where you left off:\n" + response
		}
	}

	return nil
}

func GenerateIllustrationPrompts(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Generate illustration prompts for this adventure:\n%s\n",
			adventure.Episodes[i].FullAdventure)

		response, err := client.SendMessage(GetIllustrationPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("generating illustration prompts for episode %d: %w", i, err)
		}
		log.Println(response)
		adventure.Episodes[i].Illustrations = parseIllustrationPrompts(response)
		if err := SaveToFiles(adventure, "tmp"); err != nil {
			return fmt.Errorf("writing Episode %d %w", i, err)
		}
	}
	return nil
}

func GenerateCoverPrompts(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		prompt := fmt.Sprintf("Generate cover illustration prompts for this adventure:\n%s\n",
			adventure.TableOfContents)

		response, err := client.SendMessage(GetIllustrationPrompt(), prompt)
		if err != nil {
			return fmt.Errorf("generating illustration prompts for cover %d: %w", i, err)
		}

		adventure.Covers = parseIllustrationPrompts(response)
		if err := SaveToFiles(adventure, "tmp"); err != nil {
			return fmt.Errorf("writing Episode %d %w", i, err)
		}
	}
	return nil
}

func RemoveCopyrightedMaterial(client *ClaudeClient, adventure *Adventure) error {
	for i := range adventure.Episodes {
		// Build initial prompt
		prompt := fmt.Sprintf("Remove any copyrighted material from this adventure:\n%s",
			adventure.Episodes[i].FullAdventure)

		currentPrompt := prompt
		for {
			response, err := client.SendMessage(GetCopyrightRemovalPrompt(), currentPrompt)
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
				if err := SaveToFiles(adventure, "tmp"); err != nil {
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
