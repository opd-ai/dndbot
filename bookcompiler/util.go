package bookcompiler

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/russross/blackfriday/v2"
)

// Helper function to extract text from markdown node
func getString(node *blackfriday.Node) string {
	var result strings.Builder
	node.Walk(func(n *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		if entering && n.Type == blackfriday.Text {
			result.Write(n.Literal)
		}
		return blackfriday.GoToNext
	})
	return result.String()
}

// Helper function to extract episode number
func extractEpisodeNumber(path string) int {
	// Try to find a number in the path
	re := regexp.MustCompile(`Episode(\d+)`)
	matches := re.FindStringSubmatch(filepath.Base(path))
	if len(matches) > 1 {
		if num, err := strconv.Atoi(matches[1]); err == nil {
			return num
		}
	}
	return 0
}
