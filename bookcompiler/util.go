package bookcompiler

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/russross/blackfriday/v2"
	"golang.org/x/net/html"
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

// Helper functions
func findParent(n *html.Node, tag string) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == tag {
			return p
		}
	}
	return nil
}

func countPreviousSiblings(n *html.Node) int {
	count := 0
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.Type == html.ElementNode {
			count++
		}
	}
	return count
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	var text strings.Builder
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(n)
	return text.String()
}
