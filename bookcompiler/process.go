package bookcompiler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/russross/blackfriday/v2"
	"golang.org/x/net/html"
)

func (bc *BookCompiler) processMarkdownFile(file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", file, err)
	}

	// Debug logging
	log.Printf("Processing file: %s", file)
	log.Printf("Content length: %d bytes", len(content))

	// Convert markdown to HTML
	htmlContent := blackfriday.Run(content)

	// Parse HTML into DOM
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return fmt.Errorf("error parsing HTML: %w", err)
	}

	// Process the HTML DOM tree
	return bc.renderHTML(doc)
}
