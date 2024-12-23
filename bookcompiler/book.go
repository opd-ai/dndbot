package bookcompiler

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday/v2"
)

// NewBookCompiler creates a new instance of BookCompiler
func NewBookCompiler(rootDir, outputPath string) *BookCompiler {
	bc := &BookCompiler{
		RootDir:     rootDir,
		OutputPath:  outputPath,
		imageCache:  make(map[string]bool),
		chapterFont: "Arial",
		textFont:    "Times",
		pageNumbers: true,
		tocTitle:    "Contents",
		pageWidth:   210, // A4 width in mm
		pageHeight:  297, // A4 height in mm
		margin:      20,
		tocLevels:   make(map[int]TextStyle),
	}

	// Configure ToC styles
	bc.tocLevels[1] = TextStyle{FontFamily: "Arial", Style: "B", Size: 14} // Chapter titles
	bc.tocLevels[2] = TextStyle{FontFamily: "Arial", Style: "", Size: 12}  // Major sections
	bc.tocLevels[3] = TextStyle{FontFamily: "Arial", Style: "", Size: 10}  // Subsections

	return bc
}

func (bc *BookCompiler) collectToCEntries() error {
	chapters, err := bc.getChapters()
	if err != nil {
		return err
	}

	// Add ToC page(s)
	bc.pdf.AddPage()

	for _, chapter := range chapters {
		bc.pdf.AddPage()
		chapterName := filepath.Base(chapter.Path)

		// Add chapter to ToC
		bc.toc = append(bc.toc, ToCEntry{
			Title:   chapterName,
			Level:   1,
			PageNum: bc.pdf.PageNo(),
		})

		// Collect subheadings from markdown files
		for _, file := range chapter.Files {
			if err := bc.collectMarkdownHeadings(file); err != nil {
				return err
			}
		}
	}

	return nil
}

func (bc *BookCompiler) collectMarkdownHeadings(file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// Parse markdown and extract headings
	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{})
	parser := blackfriday.New(blackfriday.WithRenderer(renderer))
	ast := parser.Parse(content)

	ast.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		if entering && node.Type == blackfriday.Heading && node.Level > 1 {
			title := getString(node)
			bc.toc = append(bc.toc, ToCEntry{
				Title:   title,
				Level:   node.Level,
				PageNum: bc.pdf.PageNo(),
			})
		}
		return blackfriday.GoToNext
	})

	return nil
}

func (bc *BookCompiler) generateToC() {
	bc.pdf.AddPage()

	// Add ToC title
	bc.pdf.SetFont(bc.chapterFont, "B", 24)
	bc.pdf.Cell(0, 10, bc.tocTitle)
	bc.pdf.Ln(20)

	// Calculate width for different columns
	contentWidth := bc.pageWidth - 2*bc.margin
	titleWidth := contentWidth * 0.85
	pageNumWidth := contentWidth * 0.15

	// Add ToC entries
	for _, entry := range bc.toc {
		// Get style for current level
		style := bc.tocLevels[entry.Level]
		bc.pdf.SetFont(style.FontFamily, style.Style, style.Size)

		// Calculate indentation
		indent := float64(entry.Level-1) * 10
		bc.pdf.SetX(bc.margin + indent)

		// Add entry text with dots
		title := entry.Title
		dots := "..."
		bc.pdf.CellFormat(
			titleWidth-indent,
			8,
			title,
			"", 0, "L", false, 0, "",
		)

		// Add page number right-aligned
		bc.pdf.CellFormat(
			pageNumWidth,
			8,
			fmt.Sprintf("%s %d", dots, entry.PageNum),
			"", 1, "R", false, 0, "",
		)
	}
}

// Add configuration methods
func (bc *BookCompiler) SetPageNumbers(enable bool) {
	bc.pageNumbers = enable
}

func (bc *BookCompiler) SetToCTitle(title string) {
	bc.tocTitle = title
}

// book.go
func (bc *BookCompiler) cleanText(text string) string {
	// More robust text cleaning
	text = strings.ReplaceAll(text, "\n", " ") // Replace newlines with spaces
	text = strings.ReplaceAll(text, "\t", " ") // Replace tabs with spaces
	text = strings.TrimSpace(text)             // Remove extra whitespace

	// Remove or replace problematic characters
	text = strings.ReplaceAll(text, "ðŸ", "")  // Remove emoji placeholders
	text = strings.ReplaceAll(text, `"`, "\"") // Replace smart quotes
	text = strings.ReplaceAll(text, `"`, "\"")
	text = strings.ReplaceAll(text, "'", "'")
	text = strings.ReplaceAll(text, "'", "'")
	text = strings.ReplaceAll(text, "…", "...")
	text = strings.ReplaceAll(text, "–", "-") // Replace en-dash
	text = strings.ReplaceAll(text, "—", "-") // Replace em-dash

	// Remove any other non-printable characters
	clean := strings.Map(func(r rune) rune {
		if r < 32 || r >= 127 {
			return -1
		}
		return r
	}, text)

	return clean
}
