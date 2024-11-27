// file_handler.go
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func saveToFiles(adventure *Adventure, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	tocPath := filepath.Join(outputDir, "ToC.md")
	if err := ioutil.WriteFile(tocPath, []byte(adventure.TableOfContents), 0o644); err != nil {
		return fmt.Errorf("saving episode: %w", err)
	}
	for i, episode := range adventure.Episodes {
		// Create episode directory
		episodeDir := filepath.Join(outputDir, fmt.Sprintf("%02d_Episode", i+1))
		if err := os.MkdirAll(episodeDir, 0o755); err != nil {
			return fmt.Errorf("creating episode directory: %w", err)
		}

		// Save episode content
		episodePath := filepath.Join(episodeDir, "Episode.md")
		if err := ioutil.WriteFile(episodePath, []byte(episode.FullAdventure), 0o644); err != nil {
			return fmt.Errorf("saving episode: %w", err)
		}

		// Save one page dungeon content
		onePagePath := filepath.Join(episodeDir, "OnePage.md")
		if err := ioutil.WriteFile(onePagePath, []byte(episode.OnePageDungeon), 0o644); err != nil {
			return fmt.Errorf("saving episode: %w", err)
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
