// render.go
package bookcompiler

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// renderNode handles rendering of a single HTML node
func (bc *BookCompiler) renderNode(n *html.Node) error {
	return bc.renderHTML(n)
}

// renderChildren processes all child nodes
func (bc *BookCompiler) renderChildren(n *html.Node) error {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := bc.renderNode(c); err != nil {
			return err
		}
	}
	return nil
}

func (bc *BookCompiler) renderHTML(n *html.Node) error {
	// Save current text state
	currentStyle := TextStyle{
		FontFamily: bc.textFont,
		Style:      "",
		Size:       12,
		Alignment:  "L",
	}

	switch n.Type {
	case html.TextNode:
		text := bc.cleanText(n.Data)
		if strings.TrimSpace(text) != "" {
			bc.pdf.Write(5, text)
		}

	case html.ElementNode:
		switch n.Data {
		// Headings
		case "h1":
			bc.pdf.AddPage()
			bc.pdf.SetFont(bc.chapterFont, "B", 24)
			bc.renderChildren(n)
			bc.pdf.Ln(20)

		case "h2":
			bc.pdf.Ln(15)
			bc.pdf.SetFont(bc.chapterFont, "B", 20)
			bc.renderChildren(n)
			bc.pdf.Ln(10)

		case "h3":
			bc.pdf.Ln(10)
			bc.pdf.SetFont(bc.chapterFont, "B", 16)
			bc.renderChildren(n)
			bc.pdf.Ln(8)

		case "h4", "h5", "h6":
			bc.pdf.Ln(8)
			bc.pdf.SetFont(bc.chapterFont, "B", 14)
			bc.renderChildren(n)
			bc.pdf.Ln(6)

		// Block elements
		case "p":
			bc.pdf.SetFont(bc.textFont, "", 12)
			bc.renderChildren(n)
			bc.pdf.Ln(8)

		case "blockquote":
			bc.pdf.SetX(bc.pdf.GetX() + 20)
			bc.pdf.SetFont(bc.textFont, "I", 12)
			bc.renderChildren(n)
			bc.pdf.SetX(bc.pdf.GetX() - 20)
			bc.pdf.Ln(8)

		case "pre", "code":
			bc.pdf.SetFont("Courier", "", 10)
			bc.renderChildren(n)
			bc.pdf.SetFont(bc.textFont, "", 12)
			bc.pdf.Ln(8)

		// Lists
		case "ul", "ol":
			bc.pdf.Ln(5)
			bc.renderChildren(n)
			bc.pdf.Ln(5)

		case "li":
			indent := 10.0
			if parent := findParent(n, "li"); parent != nil {
				indent += 10.0 // Nested list indentation
			}

			bc.pdf.SetX(bc.pdf.GetX() + indent)
			if parent := findParent(n, "ol"); parent != nil {
				number := countPreviousSiblings(n) + 1
				bc.pdf.Write(5, fmt.Sprintf("%d. ", number))
			} else {
				bc.pdf.Write(5, "• ")
			}
			bc.renderChildren(n)
			bc.pdf.Ln(5)
			bc.pdf.SetX(bc.pdf.GetX() - indent)

		// Text formatting
		case "em", "i":
			bc.pdf.SetFont(bc.textFont, "I", 12)
			bc.renderChildren(n)
			bc.pdf.SetFont(bc.textFont, "", 12)

		case "strong", "b":
			bc.pdf.SetFont(bc.textFont, "B", 12)
			bc.renderChildren(n)
			bc.pdf.SetFont(bc.textFont, "", 12)

		case "u":
			currentX := bc.pdf.GetX()
			currentY := bc.pdf.GetY()
			bc.renderChildren(n)
			width := bc.pdf.GetStringWidth(getTextContent(n))
			bc.pdf.Line(currentX, currentY+3, currentX+width, currentY+3)

		// Tables
		case "table":
			bc.renderTable(n)
			bc.pdf.Ln(8)

		// Links
		case "a":
			href := getAttr(n, "href")
			if href != "" {
				bc.pdf.SetTextColor(0, 0, 255) // Blue color for links
				bc.renderChildren(n)
				bc.pdf.SetTextColor(0, 0, 0) // Reset to black
			} else {
				bc.renderChildren(n)
			}

		// Images
		case "img":
			if src := getAttr(n, "src"); src != "" {
				bc.handleImage(src, getAttr(n, "alt"))
			}

		// Horizontal rule
		case "hr":
			bc.pdf.Line(
				bc.pdf.GetX(),
				bc.pdf.GetY(),
				bc.pdf.GetX()+190,
				bc.pdf.GetY(),
			)
			bc.pdf.Ln(8)
		}
	}

	// Restore previous text state
	bc.pdf.SetFont(currentStyle.FontFamily, currentStyle.Style, currentStyle.Size)

	// Process siblings
	for c := n.NextSibling; c != nil; c = c.NextSibling {
		if err := bc.renderHTML(c); err != nil {
			return err
		}
	}

	return nil
}

func (bc *BookCompiler) handleImage(src, alt string) error {
	// Basic image handling
	if strings.HasSuffix(strings.ToLower(src), ".jpg") ||
		strings.HasSuffix(strings.ToLower(src), ".jpeg") {
		bc.pdf.Image(src, bc.pdf.GetX(), bc.pdf.GetY(), 100, 0, false, "", 0, "")
		bc.pdf.Ln(8)
		if alt != "" {
			bc.pdf.SetFont(bc.textFont, "I", 10)
			bc.pdf.Write(5, alt)
			bc.pdf.Ln(8)
		}
	}
	return nil
}
