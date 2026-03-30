package converter

import (
	"encoding/base64"
	"fmt"
	"html"
	"math"
	"strings"
)

// htmlBuilder generates HTML from extracted PDF content.
type htmlBuilder struct {
	mode string // "reflowed" or "positioned"
}

func newHTMLBuilder(mode string) *htmlBuilder {
	if mode == "" {
		mode = "reflowed"
	}
	return &htmlBuilder{mode: mode}
}

// Build generates a complete HTML document from all pages.
func (b *htmlBuilder) Build(pages []PageResult, metadata map[string]string) string {
	var sb strings.Builder

	title := metadata["Title"]
	if title == "" {
		title = "Converted PDF"
	}

	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(title)))

	// Metadata.
	if author, ok := metadata["Author"]; ok && author != "" {
		sb.WriteString(fmt.Sprintf("<meta name=\"author\" content=\"%s\">\n", html.EscapeString(author)))
	}

	b.writeCSS(&sb)
	sb.WriteString("</head>\n<body>\n")

	for _, page := range pages {
		b.buildPage(&sb, page)
	}

	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

func (b *htmlBuilder) writeCSS(sb *strings.Builder) {
	sb.WriteString("<style>\n")

	if b.mode == "positioned" {
		sb.WriteString(`
* { margin: 0; padding: 0; box-sizing: border-box; }
body { background: #f0f0f0; font-family: sans-serif; }
.pdf-page {
  position: relative;
  background: white;
  margin: 20px auto;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  overflow: hidden;
}
.pdf-page .text-span {
  position: absolute;
  white-space: nowrap;
}
.pdf-page .pdf-image {
  position: absolute;
}
.pdf-page a {
  color: #1a0dab;
  text-decoration: underline;
}
`)
	} else {
		sb.WriteString(`
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  background: #f5f5f5;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  line-height: 1.5;
  color: #333;
}
.pdf-page {
  max-width: 900px;
  background: white;
  margin: 24px auto;
  padding: 48px 56px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.1);
  border-radius: 4px;
}
.pdf-page p { margin: 0.3em 0; }
.pdf-page table {
  border-collapse: collapse;
  width: 100%;
  margin: 1em 0;
}
.pdf-page th, .pdf-page td {
  border: 1px solid #ddd;
  padding: 8px 12px;
  text-align: left;
  vertical-align: top;
}
.pdf-page th { background: #f8f8f8; font-weight: 600; }
.pdf-page tr:nth-child(even) { background: #fafafa; }
.pdf-page img { max-width: 100%; height: auto; margin: 1em 0; display: block; }
.pdf-page a { color: #1a0dab; }
.page-break { border-top: 1px dashed #ccc; margin: 32px 0; }
`)
	}

	sb.WriteString("</style>\n")
}

func (b *htmlBuilder) buildPage(sb *strings.Builder, page PageResult) {
	if b.mode == "positioned" {
		b.buildPositionedPage(sb, page)
	} else {
		b.buildReflowedPage(sb, page)
	}
}

// buildReflowedPage generates semantic HTML with full styling preserved.
func (b *htmlBuilder) buildReflowedPage(sb *strings.Builder, page PageResult) {
	sb.WriteString(fmt.Sprintf("<div class=\"pdf-page\" data-page=\"%d\">\n", page.PageNum+1))

	// Track which lines are part of tables so we skip them in paragraph output.
	tableLineSet := make(map[float64]bool)
	for _, table := range page.Tables {
		for _, row := range table.Cells {
			for _, cell := range row {
				for _, span := range cell.Spans {
					tableLineSet[math.Round(span.Y*10)/10] = true
				}
			}
		}
	}

	// Build a link lookup by position.
	linkMap := buildLinkMap(page.Links, page.Lines)

	// Render paragraphs.
	tableIdx := 0
	for _, para := range page.Paragraphs {
		// Check if this paragraph's first line is in a table.
		if len(para.Lines) > 0 {
			firstY := math.Round(para.Lines[0].Y*10) / 10
			if tableLineSet[firstY] {
				// Render the table instead.
				if tableIdx < len(page.Tables) {
					b.renderTable(sb, page.Tables[tableIdx])
					tableIdx++
				}
				continue
			}
		}

		if para.IsHeading {
			level := para.Level
			if level < 1 {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			sb.WriteString(fmt.Sprintf("<h%d style=\"margin:0.5em 0 0.2em;\">", level))
			for _, line := range para.Lines {
				b.renderSpans(sb, line.Spans, linkMap)
			}
			sb.WriteString(fmt.Sprintf("</h%d>\n", level))
		} else {
			sb.WriteString("<p>")
			for li, line := range para.Lines {
				if li > 0 {
					sb.WriteString(" ")
				}
				b.renderSpans(sb, line.Spans, linkMap)
			}
			sb.WriteString("</p>\n")
		}
	}

	// Render remaining tables.
	for tableIdx < len(page.Tables) {
		b.renderTable(sb, page.Tables[tableIdx])
		tableIdx++
	}

	// Render images.
	for _, img := range page.Images {
		b64 := base64.StdEncoding.EncodeToString(img.Data)
		sb.WriteString(fmt.Sprintf("<img src=\"data:%s;base64,%s\" alt=\"Extracted image\" style=\"max-width:%.0fpx\">\n",
			img.MimeType, b64, img.Width))
	}

	sb.WriteString("</div>\n")
}

// buildPositionedPage generates pixel-positioned HTML.
func (b *htmlBuilder) buildPositionedPage(sb *strings.Builder, page PageResult) {
	sb.WriteString(fmt.Sprintf("<div class=\"pdf-page\" style=\"width:%.0fpx;height:%.0fpx\" data-page=\"%d\">\n",
		page.Width, page.Height, page.PageNum+1))

	for _, line := range page.Lines {
		for _, span := range line.Spans {
			// Convert PDF Y (bottom-origin) to CSS top (top-origin).
			top := page.Height - span.Y
			style := b.buildSpanStyle(span)

			sb.WriteString(fmt.Sprintf(
				"<span class=\"text-span\" style=\"left:%.1fpx;top:%.1fpx;%s\">%s</span>\n",
				span.X, top, style, html.EscapeString(span.Text)))
		}
	}

	// Images.
	for _, img := range page.Images {
		top := page.Height - img.Y - img.Height
		b64 := base64.StdEncoding.EncodeToString(img.Data)
		sb.WriteString(fmt.Sprintf(
			"<img class=\"pdf-image\" src=\"data:%s;base64,%s\" style=\"left:%.1fpx;top:%.1fpx;width:%.1fpx;height:%.1fpx\" alt=\"\">\n",
			img.MimeType, b64, img.X, top, img.Width, img.Height))
	}

	sb.WriteString("</div>\n")
}

func (b *htmlBuilder) renderSpans(sb *strings.Builder, spans []StyledSpan, linkMap map[spanKey]string) {
	for _, span := range spans {
		text := html.EscapeString(span.Text)

		// Check if this span is within a link.
		url := findLinkForSpan(linkMap, span)

		if url != "" {
			sb.WriteString(fmt.Sprintf("<a href=\"%s\">", html.EscapeString(url)))
		}

		style := b.buildSpanStyle(span)
		if style != "" {
			sb.WriteString(fmt.Sprintf("<span style=\"%s\">", style))
			sb.WriteString(text)
			sb.WriteString("</span>")
		} else {
			sb.WriteString(text)
		}

		if url != "" {
			sb.WriteString("</a>")
		}
	}
}

// buildSpanStyle creates a CSS style string that preserves the PDF formatting.
func (b *htmlBuilder) buildSpanStyle(span StyledSpan) string {
	var parts []string

	// Font size — always preserve.
	if span.FontSize > 0 {
		parts = append(parts, fmt.Sprintf("font-size:%.1fpx", span.FontSize))
	}

	// Font family from PDF font name.
	if span.FontName != "" {
		cssFont := pdfFontToCSS(span.FontName)
		if cssFont != "" {
			parts = append(parts, fmt.Sprintf("font-family:%s", cssFont))
		}
	}

	// Bold.
	if span.Bold {
		parts = append(parts, "font-weight:bold")
	}

	// Italic.
	if span.Italic {
		parts = append(parts, "font-style:italic")
	}

	// Color — always preserve (even dark or white colors).
	if span.Color[0] > 0.01 || span.Color[1] > 0.01 || span.Color[2] > 0.01 {
		parts = append(parts, fmt.Sprintf("color:%s", rgbToCSS(span.Color)))
	}

	return strings.Join(parts, ";")
}

// pdfFontToCSS maps a PDF font name to a CSS font-family value.
func pdfFontToCSS(pdfFont string) string {
	lower := strings.ToLower(pdfFont)

	// Strip common prefixes and suffixes.
	lower = strings.TrimPrefix(lower, "/")

	switch {
	case strings.Contains(lower, "helvetica") || strings.Contains(lower, "arial"):
		return "Helvetica, Arial, sans-serif"
	case strings.Contains(lower, "times") || strings.Contains(lower, "serif"):
		return "'Times New Roman', Times, serif"
	case strings.Contains(lower, "courier") || strings.Contains(lower, "mono"):
		return "'Courier New', Courier, monospace"
	case strings.Contains(lower, "symbol"):
		return "Symbol, serif"
	case strings.Contains(lower, "zapf") || strings.Contains(lower, "dingbat"):
		return "ZapfDingbats, serif"
	default:
		// Use the original name as CSS, plus fallback.
		name := strings.TrimPrefix(pdfFont, "/")
		// Strip subset prefix (ABCDEF+).
		if idx := strings.Index(name, "+"); idx >= 0 && idx <= 6 {
			name = name[idx+1:]
		}
		// Strip -Bold, -Italic etc.
		for _, suffix := range []string{"-Bold", "-Italic", "-BoldItalic", "-Oblique", "-BoldOblique", ",Bold", ",Italic", ",BoldItalic"} {
			name = strings.TrimSuffix(name, suffix)
		}
		return fmt.Sprintf("'%s', sans-serif", name)
	}
}

func (b *htmlBuilder) renderTable(sb *strings.Builder, table DetectedTable) {
	sb.WriteString("<table>\n")
	for r, row := range table.Cells {
		sb.WriteString("<tr>")
		tag := "td"
		if r == 0 {
			tag = "th"
		}
		for _, cell := range row {
			// Build cell content with span styling.
			var cellContent strings.Builder
			if len(cell.Spans) > 0 {
				for _, span := range cell.Spans {
					text := html.EscapeString(span.Text)
					style := b.buildSpanStyle(span)
					if style != "" {
						cellContent.WriteString(fmt.Sprintf("<span style=\"%s\">%s</span>", style, text))
					} else {
						cellContent.WriteString(text)
					}
				}
			} else {
				cellContent.WriteString(html.EscapeString(cell.Text))
			}
			sb.WriteString(fmt.Sprintf("<%s>%s</%s>", tag, cellContent.String(), tag))
		}
		sb.WriteString("</tr>\n")
	}
	sb.WriteString("</table>\n")
}

type spanKey struct {
	x, y float64
}

// buildLinkMap creates a position-based lookup for links using bounding boxes.
func buildLinkMap(links []ExtractedLink, lines []Line) map[spanKey]string {
	m := make(map[spanKey]string)
	for _, link := range links {
		// Find spans that fall within this link's rectangle.
		for _, line := range lines {
			for _, span := range line.Spans {
				if spanInRect(span, link.Rect) {
					m[spanKey{x: span.X, y: span.Y}] = link.URL
				}
			}
		}
	}
	return m
}

// spanInRect checks if a span's position falls within a PDF rectangle.
func spanInRect(span StyledSpan, rect [4]float64) bool {
	// rect is [x1, y1, x2, y2], normalize.
	x1 := math.Min(rect[0], rect[2])
	y1 := math.Min(rect[1], rect[3])
	x2 := math.Max(rect[0], rect[2])
	y2 := math.Max(rect[1], rect[3])
	// Add tolerance for matching.
	tol := 2.0
	return span.X >= x1-tol && span.X <= x2+tol && span.Y >= y1-tol && span.Y <= y2+tol
}

func findLinkForSpan(linkMap map[spanKey]string, span StyledSpan) string {
	return linkMap[spanKey{x: span.X, y: span.Y}]
}

func rgbToCSS(c [3]float64) string {
	r := clamp(int(math.Round(c[0]*255)), 0, 255)
	g := clamp(int(math.Round(c[1]*255)), 0, 255)
	b := clamp(int(math.Round(c[2]*255)), 0, 255)
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
