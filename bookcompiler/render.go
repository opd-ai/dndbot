package bookcompiler

import (
	"strings"

	"golang.org/x/net/html"
)

func (bc *BookCompiler) renderHTML(n *html.Node) error {
	switch n.Type {
	case html.TextNode:
		text := bc.cleanText(n.Data)
		if strings.TrimSpace(text) != "" {
			bc.pdf.Write(5, text)
		}
	case html.ElementNode:
		switch n.Data {
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
		case "h4":
			bc.pdf.Ln(8)
			bc.pdf.SetFont(bc.chapterFont, "B", 14)
			bc.renderChildren(n)
			bc.pdf.Ln(6)
		case "p":
			bc.pdf.SetFont(bc.textFont, "", 12)
			bc.renderChildren(n)
			bc.pdf.Ln(8)
		case "em":
			// For italics, just switch to italic style and back
			bc.pdf.SetFont(bc.textFont, "I", 12)
			bc.renderChildren(n)
			bc.pdf.SetFont(bc.textFont, "", 12)
		case "strong":
			// For bold, switch to bold style and back
			bc.pdf.SetFont(bc.textFont, "B", 12)
			bc.renderChildren(n)
			bc.pdf.SetFont(bc.textFont, "", 12)
		case "ul":
			bc.pdf.Ln(5)
			bc.renderChildren(n)
			bc.pdf.Ln(5)
		case "ol":
			bc.pdf.Ln(5)
			bc.renderChildren(n)
			bc.pdf.Ln(5)
		case "li":
			bc.pdf.SetX(bc.pdf.GetX() + 10)
			bc.pdf.Write(5, "• ")
			bc.renderChildren(n)
			bc.pdf.Ln(5)
			bc.pdf.SetX(bc.pdf.GetX() - 10)
		case "img":
			// Handle images if needed
			for _, a := range n.Attr {
				if a.Key == "src" {
					// Process image
					break
				}
			}
		}
	}

	// Process siblings
	for c := n.NextSibling; c != nil; c = c.NextSibling {
		if err := bc.renderHTML(c); err != nil {
			return err
		}
	}

	return nil
}

func (bc *BookCompiler) renderChildren(n *html.Node) error {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := bc.renderNode(c); err != nil {
			return err
		}
	}
	return nil
}
