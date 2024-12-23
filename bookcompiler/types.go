package bookcompiler

import "github.com/jung-kurt/gofpdf"

// BookCompiler handles the compilation of markdown files into a PDF
type BookCompiler struct {
	RootDir     string
	OutputPath  string
	pdf         *gofpdf.Fpdf
	imageCache  map[string]bool
	chapterFont string
	textFont    string
	toc         []ToCEntry
	pageNumbers bool
	tocTitle    string
	pageWidth   float64
	pageHeight  float64
	margin      float64
	tocLevels   map[int]TextStyle // Different styles for different ToC levels
}

// ToCEntry represents a table of contents entry
type ToCEntry struct {
	Title   string
	Level   int
	PageNum int
	Link    int // Internal PDF link identifier
}

// Chapter represents a directory containing markdown files
type Chapter struct {
	Path  string
	Files []string
}

// TextStyle holds current text formatting state
type TextStyle struct {
	FontFamily string
	Style      string
	Size       float64
	Alignment  string
}
