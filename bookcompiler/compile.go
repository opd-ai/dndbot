package bookcompiler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/net/html"
)

// Compile processes all markdown files and creates the PDF
func (bc *BookCompiler) Compile() error {
	// Initialize PDF
	bc.pdf = gofpdf.New("P", "mm", "A4", "")
	bc.pdf.SetMargins(20, 20, 20)

	// Enable page numbers if requested
	if bc.pageNumbers {
		bc.pdf.SetFooterFunc(func() {
			bc.pdf.SetY(-15)
			bc.pdf.SetFont("Arial", "I", 8)
			bc.pdf.CellFormat(0, 10, fmt.Sprintf("Page %d", bc.pdf.PageNo()),
				"", 0, "C", false, 0, "")
		})
	}

	// First pass: collect ToC entries and page numbers
	if err := bc.collectToCEntries(); err != nil {
		return fmt.Errorf("error collecting ToC entries: %w", err)
	}

	// Reset PDF and start actual content generation
	bc.pdf = gofpdf.New("P", "mm", "A4", "")
	bc.pdf.SetMargins(20, 20, 20)

	// Generate ToC pages
	bc.generateToC()

	// Process all chapters
	chapters, err := bc.getChapters()
	if err != nil {
		return fmt.Errorf("error getting chapters: %w", err)
	}

	for _, chapter := range chapters {
		if err := bc.processChapter(chapter); err != nil {
			return fmt.Errorf("error processing chapter %s: %w", chapter.Path, err)
		}
	}

	return bc.pdf.OutputFileAndClose(bc.OutputPath)
}

func (bc *BookCompiler) processChapter(chapter Chapter) error {
	// Add chapter title
	chapterName := filepath.Base(chapter.Path)
	chapterName = strings.TrimPrefix(chapterName, "Episode")
	chapterName = fmt.Sprintf("Episode %s", strings.TrimSpace(chapterName))

	bc.pdf.SetFont(bc.chapterFont, "B", 24)
	bc.pdf.Cell(0, 10, chapterName)
	bc.pdf.Ln(20)

	// Process each markdown file
	for _, file := range chapter.Files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("error reading file %s: %w", file, err)
		}

		// Convert markdown to HTML
		htmlbytes := blackfriday.Run(content)

		// Parse HTML
		doc, err := html.Parse(bytes.NewReader(htmlbytes))
		if err != nil {
			return fmt.Errorf("error parsing HTML: %w", err)
		}
		log.Printf("generated HTML: %s\n", htmlbytes)

		// Render content
		if err := bc.renderHTML(doc); err != nil {
			return err
		}
	}

	return nil
}

func (bc *BookCompiler) updateToCPageNumbers(pageTracker map[string]int) {
	for i := range bc.toc {
		if page, ok := pageTracker[bc.toc[i].Title]; ok {
			bc.toc[i].PageNum = page
		}
	}
}
