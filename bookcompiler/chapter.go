package bookcompiler

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

func (bc *BookCompiler) getChapters() ([]Chapter, error) {
	var chapters []Chapter

	entries, err := ioutil.ReadDir(bc.RootDir)
	if err != nil {
		return nil, fmt.Errorf("error reading root directory: %w", err)
	}

	// Debug the directory contents
	log.Printf("Directory contents of %s:", bc.RootDir)
	for _, entry := range entries {
		log.Printf("Found entry: %s (isDir: %v)", entry.Name(), entry.IsDir())
	}

	// Filter and sort episode directories
	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), "Episode") {
			chapterPath := filepath.Join(bc.RootDir, entry.Name())

			// Get all markdown files in the chapter
			files, err := bc.getMarkdownFiles(chapterPath)
			if err != nil {
				log.Printf("Warning: error reading chapter %s: %v", entry.Name(), err)
				continue // Skip this chapter but continue processing others
			}

			log.Printf("Adding chapter: %s with %d files", entry.Name(), len(files))
			chapters = append(chapters, Chapter{
				Path:  chapterPath,
				Files: files,
			})
		}
	}

	// Sort chapters by name to maintain order
	sort.Slice(chapters, func(i, j int) bool {
		// Extract episode numbers for proper sorting
		numI := extractEpisodeNumber(chapters[i].Path)
		numJ := extractEpisodeNumber(chapters[j].Path)
		return numI < numJ
	})

	return chapters, nil
}

func (bc *BookCompiler) getMarkdownFiles(path string) ([]string, error) {
	var files []string

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	log.Printf("Scanning directory %s for markdown files", path)

	// First, look for markdown files
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			filePath := filepath.Join(path, entry.Name())
			log.Printf("Found markdown file: %s", entry.Name())
			files = append(files, filePath)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no markdown files found in %s", path)
	}

	// Sort files to ensure consistent order
	sort.Strings(files)
	return files, nil
}
