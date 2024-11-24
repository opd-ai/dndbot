// helpers.go
package main

import (
	"strings"
)

func getOnePageDungeonPrompt() string {
	return `Convert this episode summary into a one-page dungeon format following these guidelines:
    1. Start with a clear location description
    2. List key NPCs and their motivations
    3. Include a random encounter table (1d6)
    4. Add a treasure table (1d6)
    5. Describe key locations within the dungeon
    6. Include any relevant traps or puzzles
    7. Provide monster statistics in abbreviated format
    Format the response in markdown.`
}

func getExpandedAdventurePrompt() string {
	return `Expand this one-page dungeon into a detailed 10-page adventure including:
    1. Detailed background and hook
    2. Complete location descriptions
    3. Full NPC backgrounds and personalities
    4. Detailed encounter descriptions
    5. Complete monster statistics
    6. Multiple possible paths through the adventure
    7. Alternative endings
    8. Scaling options for different party levels
    Format the response in markdown with clear sections.`
}

func getIllustrationPrompt() string {
	return `Generate 2-6 Stable Diffusion prompts for this adventure. Include:
    1. At least one map or location layout
    2. Key scenes or dramatic moments
    3. Important characters or monsters
    Avoid text elements in the images.
    For each prompt, specify:
    - Detailed visual description
    - Art style (e.g., dark fantasy, heroic fantasy, etc.)
    - Lighting and mood
    - Composition details`
}

func getCopyrightRemovalPrompt() string {
	return `Review and revise this adventure to remove or replace any copyrighted material:
    1. Replace specific D&D monsters with generic alternatives
    2. Remove trademarked spells and items
    3. Generalize any specific setting references
    4. Maintain the adventure's theme and feeling while using original content
    5. Ensure mechanical elements are system-agnostic`
}

func parseEpisodes(content string) []Episode {
	var episodes []Episode
	lines := strings.Split(content, "\n")
	var currentEpisode Episode

	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "## Episode"):
			if currentEpisode.Title != "" {
				episodes = append(episodes, currentEpisode)
			}
			currentEpisode = Episode{
				Title: strings.TrimPrefix(line, "## "),
			}
		case strings.HasPrefix(line, "Summary:"):
			currentEpisode.Summary = strings.TrimPrefix(line, "Summary: ")
		case strings.HasPrefix(line, "Tagline:"):
			currentEpisode.Tagline = strings.TrimPrefix(line, "Tagline: ")
		case strings.HasPrefix(line, "Location:"):
			currentEpisode.Location = strings.TrimPrefix(line, "Location: ")
		case strings.HasPrefix(line, "Characters:"):
			chars := strings.TrimPrefix(line, "Characters: ")
			currentEpisode.Characters = strings.Split(chars, ", ")
		}
	}

	if currentEpisode.Title != "" {
		episodes = append(episodes, currentEpisode)
	}

	return episodes
}

func parseIllustrationPrompts(content string) []IllustrationPrompt {
	var prompts []IllustrationPrompt
	lines := strings.Split(content, "\n")
	var currentPrompt IllustrationPrompt

	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "## Illustration"):
			if currentPrompt.Description != "" {
				prompts = append(prompts, currentPrompt)
			}
			currentPrompt = IllustrationPrompt{}
		case strings.HasPrefix(line, "Description:"):
			currentPrompt.Description = strings.TrimPrefix(line, "Description: ")
		case strings.HasPrefix(line, "Style:"):
			currentPrompt.Style = strings.TrimPrefix(line, "Style: ")
		case strings.HasPrefix(line, "Type:"):
			currentPrompt.IsMap = strings.Contains(strings.ToLower(line), "map")
		}
	}

	if currentPrompt.Description != "" {
		prompts = append(prompts, currentPrompt)
	}

	return prompts
}
