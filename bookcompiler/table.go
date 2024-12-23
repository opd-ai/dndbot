package bookcompiler

import (
	"strings"

	"golang.org/x/net/html"
)

func (bc *BookCompiler) renderTable(n *html.Node) error {
	// Get table structure
	var headers []string
	var rows [][]string

	// Process table rows
	for tr := n.FirstChild; tr != nil; tr = tr.NextSibling {
		if tr.Type != html.ElementNode || tr.Data != "tr" {
			continue
		}

		var row []string
		isHeaderRow := false
		for td := tr.FirstChild; td != nil; td = td.NextSibling {
			if td.Type != html.ElementNode || (td.Data != "td" && td.Data != "th") {
				continue
			}

			// Get cell content
			cellText := getTextContent(td)

			// If this is a header row
			if td.Data == "th" {
				headers = append(headers, cellText)
				isHeaderRow = true
			} else {
				row = append(row, cellText)
			}
		}

		if !isHeaderRow && len(row) > 0 {
			rows = append(rows, row)
		}
	}

	// Calculate column widths
	colCount := len(headers)
	if colCount == 0 && len(rows) > 0 {
		colCount = len(rows[0])
	}

	// Available width (accounting for margins)
	availWidth := 170.0 // A4 width minus margins
	colWidth := availWidth / float64(colCount)

	// Set table styling
	lineHt := 6.0
	fontSize := 10.0
	bc.pdf.SetFont(bc.textFont, "B", fontSize)

	// Draw headers
	if len(headers) > 0 {
		bc.pdf.SetFillColor(240, 240, 240) // Light gray background
		for _, header := range headers {
			x := bc.pdf.GetX()
			y := bc.pdf.GetY()
			bc.pdf.Rect(x, y, colWidth, lineHt, "F")
			bc.pdf.Cell(colWidth, lineHt, header)
		}
		bc.pdf.Ln(lineHt)
	}

	// Draw rows
	bc.pdf.SetFont(bc.textFont, "", fontSize)
	for _, row := range rows {
		maxHt := lineHt
		// First pass: calculate max height for this row
		for _, cell := range row {
			lines := bc.SplitText(cell, colWidth)
			ht := float64(len(lines)) * lineHt
			if ht > maxHt {
				maxHt = ht
			}
		}

		// Second pass: draw cells
		y := bc.pdf.GetY()
		x := bc.pdf.GetX()
		for i, cell := range row {
			bc.pdf.Rect(x+float64(i)*colWidth, y, colWidth, maxHt, "D")
			bc.pdf.MultiCell(colWidth, lineHt, cell, "0", "L", false)
			bc.pdf.SetXY(x+float64(i+1)*colWidth, y)
		}
		bc.pdf.Ln(maxHt)
	}

	return nil
}

func (bc *BookCompiler) SplitText(text string, width float64) []string {
	var lines []string
	words := strings.Split(text, " ")
	currentLine := ""

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if bc.pdf.GetStringWidth(testLine) > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				lines = append(lines, word)
			}
		} else {
			currentLine = testLine
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
