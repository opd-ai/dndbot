package dndbot

import (
	"strings"
)

func (c Adventure) getSettingDetails() string {
	val := `
	This adventure takes place in an established campaign setting.
	For details about the campaign setting, refer to the following details.
	Focus on writing the story of the adventure(above) in the provided setting(below)
	`
	val += "```\n"
	val += "BEGIN SETTING DETAILS\n"
	/*bytes, err := os.ReadFile(c.Setting)
	if err != nil {
		return ""
	}*/
	val += c.Setting
	val += "\nEND SETTING DETAILS\n"
	val += "```\n"
	return val
}

func (c Adventure) getWritingStyleDetails() string {
	val := "\n```\n"
	val += "BEGIN WRITING STYLE DETAILS\n"
	/*bytes, err := os.ReadFile(c.Style)
	if err != nil {
		return ""
	}*/

	val += c.Style
	val += "\nEND WRITING STYLE DETAILS\n"
	val += "```\n"
	return val
}

func GetOnePageDungeonPrompt(settingDetails string) string {
	return `Convert this episode summary into a one-page dungeon format(about 80 lines) following these guidelines:
    1. Start with a clear location description
    2. List key NPCs and their motivations
    3. Include a random encounter table (1d6)
    4. Add a treasure table (1d6)
    5. Describe key locations within the dungeon
    6. Include any relevant traps or puzzles
    7. Provide monster statistics in abbreviated format
	8. Game-system agnostic
    Format the response in beautifully structured markdown with symbols and emoji.
	The author is anonymous.
	No disclaimers or credits are necessary.
	Everything is Creative Commons Zero with no attribution.

	Do this without asking for confirmation or direction.
	Do not ask for confirmation in any way, just output the complete adventure.
	This is essential.
	
	Episode should also include a unique side-plot.
	Prefer a relatable sense of realism.
	Fantasy is acceptable, but avoiding material circumstances is not.
	Avoid overt flights of fancy.
	Maintain verisimiliture throughout the story.
	` + settingDetails
}

func GetExpandedAdventurePrompt(settingDetails string) string {
	return `Expand this one-page dungeon into a detailed 8 page adventure(about 600 lines) including:
    1. Detailed background and hook
    2. Complete location descriptions
    3. Full NPC descriptive backgrounds and personalities
    4. Detailed encounter descriptions
    5. Complete monster statistics with physical and tactical description
    6. Multiple possible paths through the adventure
    7. Alternative endings
    8. Scaling options for different party levels
	9. Game-system agnostic
    Format the response in beautifully structured markdown with symbols and emoji, with clear sections.
	Longer sections should use completete sentences and paragraphs.
	The author is anonymous.
	No disclaimers or credits are necessary.
	Everything is Creative Commons Zero with no attribution.

	Episode should also include a unique side-plot.
	Prefer a relatable sense of realism.
	Fantasy is acceptable, but avoiding material circumstances is not.
	Avoid overt flights of fancy.
	Maintain verisimiliture throughout the story.

	Do this without asking for confirmation or direction.
	Do not ask for confirmation in any way, just output the complete adventure.
	This is essential.

	If it is necessary due to response length, break the result into one-page(about 80 lines) sections.
	Do this until you reach the full 8 pages minimum.
	At the top of each page, add [Page Number] of [Total pages].
	At the bottom of each page except the final page, add [continued on next page].
	On the last page, add [final page].
	Never refer to yourself.
	` + settingDetails
}

func GetIllustrationPrompt() string {
	prompt := `Generate 2-6 Stable Diffusion prompts for this adventure. Include:
    1. At least one map or location layout
    2. Key scenes or dramatic moments
    3. Important characters or monsters
    Avoid text elements in the images.
    For each prompt, specify:
    - Detailed visual description
    - Art style (e.g., dark fantasy, heroic fantasy, etc.)
    - Lighting and mood
    - Composition details

	Follow this example format exactly for each consecutive episode:`
	prompt += "```\n"
	prompt += `## Illustration: Number - Episode Title - Illustration Title
Description: 3-8 sentence description of the scene optimized for Stable Diffusion XL
Style: Stylistic description of the art
Type: Map OR Portait OR Scene etc...

`
	prompt += "```\n"
	return prompt
}

func GetCopyrightRemovalPrompt() string {
	return `Review and revise this adventure to remove or replace any copyrighted material and output a complete edited version:
    1. Replace specific D&D monsters with generic alternatives
    2. Remove trademarked spells and items
    3. Generalize any specific setting references
    4. Maintain the adventure's theme and feeling while using original content
    5. Ensure mechanical elements are system-agnostic
	6. Remove unacceptable tropes such as racism and sexism.
	6. Output the complete adventure with revisions, do not provide suggestions.

	Even if no revisions need to be made, output the complete original adventure.
	Do this without asking for confirmation or direction.
	Do not ask for confirmation in any way, just output the complete adventure.
	This is essential.

	If no revision is made, do not report any additional information.
	Just output the original adventure.

	Provide a complete revised and edited version of the entire adventure.
	Preserve the existing response in beautifully formatted markdown with symbols and emoji, with clear sections.
	The author is anonymous.
	No disclaimers or credits are necessary.
	Everything is Creative Commons Zero with no attribution.

	If it is necessary due to response length, break the result into one-page(about 80 lines) sections.
	Do this until you reach the full 8 pages minimum.
	At the top of each page, add [Page Number] of [Total pages].
	At the bottom of each page except the final page, add [continued on next page].
	On the last page, add [final page].
	Never refer to yourself.
	`
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
