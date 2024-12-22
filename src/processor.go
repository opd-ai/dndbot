package dndbot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/opd-ai/horde"
)

func GenerateTableOfContents(client *ClaudeClient, prompt string, p progressor, setting, style string) (Adventure, error) {
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

	adventure := Adventure{
		OriginalPrompt: prompt,
		Setting:        setting,
		Style:          style,
	}

	systemPrompt += adventure.getSettingDetails()

	response, err := client.SendMessage(systemPrompt, "This is the story prompt, it is very important that you follow this prompt:"+prompt)
	if err != nil {
		return Adventure{}, fmt.Errorf("generating ToC: %w", err)
	}
	log.Println(response)
	pr.UpdateOutput(response)

	// Parse the response into Adventure struct
	adventure.TableOfContents = response

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

		response, err := client.SendMessage(GetOnePageDungeonPrompt(adventure.getSettingDetails()), prompt)
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
			response, err := client.SendMessage(GetExpandedAdventurePrompt(adventure.getWritingStyleDetails()), currentPrompt)
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

func amap(s bool) string {
	if s {
		return "Area map"
	}
	return "Illustration"
}

func GenerateIllustrationsFromPrompts(client ImageClient, adventure *Adventure, path string, progress progressor) error {
	var pr progressor
	if progress != nil {
		pr = progress
	} else {
		pr = &nullProgressor{}
	}
	for index, episode := range adventure.Episodes {
		indexString := fmt.Sprintf("%02d", index+1)
		dir := filepath.Join(path, indexString+"_Episode")
		for index2, illustration := range episode.Illustrations {
			prompt := fmt.Sprintf("%s\n%s\n%s", amap(illustration.IsMap), illustration.Description, illustration.Style)
			pr.UpdateOutput("Generating illustration image by prompting SDXL(This will take a while): " + prompt)
			data, err := client.ImageGenerate(prompt, 30, 0, 0, "Dreamshaper XL", progress)
			if err != nil {
				return err
			}
			os.MkdirAll(dir, 0o755)
			outPath := filepath.Join(dir, filenamer(illustration.Description))
			pngPath := strings.TrimSuffix(outPath, filepath.Ext(outPath))
			if err := os.WriteFile(outPath, data, 0o644); err != nil {
				return err
			} else {
				if os.Getenv("SD_WEBUI_URL") == "" {
					if err := horde.Webp2PNG(outPath); err != nil {
						return err
					} else {
						if err := os.Remove(outPath); err != nil {
							return err
						}
					}
				}
			}
			caption := fmt.Sprintf("%s:%s:%s", amap(illustration.IsMap), illustration.Description, illustration.Style)
			fields := fmt.Sprintf("\n  * Category: %s\n  * Description: %s\n  * Style: %s\n", amap(illustration.IsMap), illustration.Description, illustration.Style)
			captionFile := fmt.Sprintf(" - [%s](%s) `%s`\n", caption, pngPath, fields)
			indexString2 := fmt.Sprintf("%02d", index2)
			if err := os.WriteFile(filepath.Join(dir, indexString2+"_Illustration.md"), []byte(captionFile), 0o644); err != nil {
				return err
			}
			pr.UpdateOutput("Generated illustration image. Proceeding...\n")
		}
	}
	return nil
}

func GenerateCoversFromPrompts(client ImageClient, adventure *Adventure, path string, progress progressor) error {
	var pr progressor
	if progress != nil {
		pr = progress
	} else {
		pr = &nullProgressor{}
	}
	for index2, illustration := range adventure.Covers {
		prompt := fmt.Sprintf("%s\n%s\n%s", amap(illustration.IsMap), illustration.Description, illustration.Style)
		pr.UpdateOutput("Generating cover image by prompting SDXL(This will take a while): " + prompt)
		data, err := client.ImageGenerate(prompt, 30, 0, 0, "Dreamshaper XL", progress)
		if err != nil {
			return err
		}
		os.MkdirAll(path, 0o755)
		outPath := filepath.Join(path, filenamer(illustration.Description))
		pngPath := strings.TrimSuffix(outPath, filepath.Ext(outPath))
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return err
		} else {
			if os.Getenv("SD_WEBUI_URL") == "" {
				if err := horde.Webp2PNG(outPath); err != nil {
					return err
				} else {
					if err := os.Remove(outPath); err != nil {
						return err
					}
				}
			}
		}
		caption := fmt.Sprintf("%s:%s:%s", amap(illustration.IsMap), illustration.Description, illustration.Style)
		fields := fmt.Sprintf("\n  * Category: %s\n  * Description: %s\n  * Style: %s\n", amap(illustration.IsMap), illustration.Description, illustration.Style)
		captionFile := fmt.Sprintf(" - [%s](%s) `%s`\n", caption, pngPath, fields)
		indexString2 := fmt.Sprintf("%02d", index2)
		if err := os.WriteFile(filepath.Join(path, indexString2+"_CoverIllustration.md"), []byte(captionFile), 0o644); err != nil {
			return err
		}
		pr.UpdateOutput("Generated cover image. Proceeding...\n")
	}
	return nil
}

func filenamer(desc string) string {
	split := strings.Split(desc, " ")
	result := ""
	for _, str := range split {
		result += string(str[0])
	}
	return result + ".webp"
}
