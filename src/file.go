package dndbot

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func SaveToFiles(adventure *Adventure, outputDir string) error {
	contentPath := filepath.Join(outputDir, "00_Contents")
	if err := os.MkdirAll(contentPath, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	tocPath := filepath.Join(contentPath, "Contents.md")
	if err := ioutil.WriteFile(tocPath, []byte(adventure.TableOfContents), 0o644); err != nil {
		return fmt.Errorf("saving episode: %w", err)
	}
	for i, cover := range adventure.Covers {
		illusPath := filepath.Join(contentPath, fmt.Sprintf("Caption_%02d.md", i+1))
		content := fmt.Sprintf("Description: %s\nStyle: %s\nIs Map: %v",
			cover.Description, cover.Style, cover.IsMap)
		if err := ioutil.WriteFile(illusPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("saving illustration prompt: %w", err)
		}
	}
	for i, episode := range adventure.Episodes {
		// Create episode directory
		episodeDir := filepath.Join(outputDir, fmt.Sprintf("%02d_Episode", i+1))
		if err := os.MkdirAll(episodeDir, 0o755); err != nil {
			return fmt.Errorf("creating episode directory: %w", err)
		}

		// Save episode content
		episodePath := filepath.Join(cleanupBytes(episodeDir), "Episode.md")
		if len(episode.FullAdventure) > 0 {
			if err := ioutil.WriteFile(episodePath, []byte(episode.FullAdventure), 0o644); err != nil {
				return fmt.Errorf("saving episode: %w", err)
			}
		}

		// Save one page dungeon content
		onePagePath := filepath.Join(episodeDir, "OnePage.md")
		if len(episode.OnePageDungeon) > 0 {
			if err := ioutil.WriteFile(onePagePath, []byte(episode.OnePageDungeon), 0o644); err != nil {
				return fmt.Errorf("saving episode: %w", err)
			}
		}

		// Save illustration prompts
		for j, illus := range episode.Illustrations {
			illusPath := filepath.Join(episodeDir, fmt.Sprintf("Caption_%02d.md", j+1))
			content := fmt.Sprintf("Description: %s\nStyle: %s\nIs Map: %v",
				illus.Description, illus.Style, illus.IsMap)
			if err := ioutil.WriteFile(illusPath, []byte(content), 0o644); err != nil {
				return fmt.Errorf("saving illustration prompt: %w", err)
			}
		}
	}
	return nil
}

func cleanupBytes(input string) string {
	const (
		searchText  = "[continued on next page]"
		replaceText = "[continued on next page]\n\\newpage\n"
	)

	return strings.Replace(input, searchText, replaceText, -1)
}
