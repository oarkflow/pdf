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

	if author, ok := metadata["Author"]; ok && author != "" {
		sb.WriteString(fmt.Sprintf("<meta name=\"author\" content=\"%s\">\n", html.EscapeString(author)))
	}

	b.writeCSS(&sb)
	sb.WriteString("</head>\n<body>\n")

	for i, page := range pages {
		if i > 0 {
			sb.WriteString("<div class=\"page-break\"></div>\n")
		}
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
.pdf-page a { color: #1a0dab; text-decoration: underline; }
`)
	} else {
		sb.WriteString(`
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  background: #f5f5f5;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  line-height: 1.4;
  color: #222;
}
.pdf-page {
  max-width: 900px;
  background: white;
  margin: 24px auto;
  padding: 48px 56px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.1);
  border-radius: 4px;
}
.pdf-page .line { margin: 0; padding: 0; }
.pdf-page table {
  border-collapse: collapse;
  width: 100%;
  margin: 0.6em 0;
}
.pdf-page th, .pdf-page td {
  border: 1px solid #ddd;
  padding: 6px 10px;
  text-align: left;
  vertical-align: top;
}
.pdf-page th { background: #f8f8f8; font-weight: 600; }
.pdf-page tr:nth-child(even) { background: #fafafa; }
.pdf-page img { max-width: 100%; height: auto; margin: 0.8em 0; display: block; }
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

// buildReflowedPage generates HTML that preserves visual structure line-by-line.
// Instead of guessing paragraph boundaries, it renders each line as a block
// with margin-top derived from the actual Y gap between consecutive lines.
func (b *htmlBuilder) buildReflowedPage(sb *strings.Builder, page PageResult) {
	sb.WriteString(fmt.Sprintf("<div class=\"pdf-page\" data-page=\"%d\">\n", page.PageNum+1))

	// Build a link lookup by position.
	linkMap := buildLinkMap(page.Links, page.Lines)

	// Track which lines are part of tables.
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

	tableIdx := 0
	prevY := -1.0
	prevFontSize := 12.0

	for _, line := range page.Lines {
		if len(line.Spans) == 0 {
			continue
		}

		lineY := math.Round(line.Y*10) / 10

		// If this line belongs to a table, render the table instead.
		if tableLineSet[lineY] {
			if tableIdx < len(page.Tables) {
				b.renderTable(sb, page.Tables[tableIdx], linkMap)
				tableIdx++
			}
			prevY = line.Y
			prevFontSize = avgFontSize(line.Spans)
			continue
		}

		fontSize := avgFontSize(line.Spans)
		if fontSize == 0 {
			fontSize = 12
		}

		// Compute visual spacing from Y gap.
		marginTop := 0.0
		if prevY > 0 {
			yGap := math.Abs(prevY - line.Y)
			lineHeight := math.Max(fontSize, prevFontSize) * 1.2
			if yGap > lineHeight*2.0 {
				// Large gap: section break.
				marginTop = yGap - lineHeight
				if marginTop > 40 {
					marginTop = 40
				}
			} else if yGap > lineHeight*1.3 {
				// Medium gap: paragraph break.
				marginTop = fontSize * 0.6
			}
			// Normal line spacing: no extra margin needed.
		}

		// Determine element type.
		tag := "div"
		if line.IsHeading {
			level := line.Level
			if level < 1 {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			tag = fmt.Sprintf("h%d", level)
		}

		// Build the line's style.
		var lineStyles []string
		if marginTop > 1 {
			lineStyles = append(lineStyles, fmt.Sprintf("margin-top:%.0fpx", marginTop))
		}

		styleAttr := ""
		if tag != "div" {
			lineStyles = append([]string{"margin:0"}, lineStyles...)
		}
		if len(lineStyles) > 0 {
			styleAttr = fmt.Sprintf(" style=\"%s\"", strings.Join(lineStyles, ";"))
		}

		if tag == "div" {
			sb.WriteString(fmt.Sprintf("<div class=\"line\"%s>", styleAttr))
		} else {
			sb.WriteString(fmt.Sprintf("<%s%s>", tag, styleAttr))
		}

		b.renderSpans(sb, line.Spans, linkMap)

		if tag == "div" {
			sb.WriteString("</div>\n")
		} else {
			sb.WriteString(fmt.Sprintf("</%s>\n", tag))
		}

		prevY = line.Y
		prevFontSize = fontSize
	}

	// Render remaining tables.
	for tableIdx < len(page.Tables) {
		b.renderTable(sb, page.Tables[tableIdx], linkMap)
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
			top := page.Height - span.Y
			style := b.buildSpanStyle(span)
			sb.WriteString(fmt.Sprintf(
				"<span class=\"text-span\" style=\"left:%.1fpx;top:%.1fpx;%s\">%s</span>\n",
				span.X, top, style, html.EscapeString(span.Text)))
		}
	}

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
	for i, span := range spans {
		text := html.EscapeString(span.Text)

		// Check if this span is within a link.
		url := findLinkForSpan(linkMap, span)

		if url != "" {
			sb.WriteString(fmt.Sprintf("<a href=\"%s\">", html.EscapeString(url)))
		}

		style := b.buildSpanStyle(span)

		// Add left margin for horizontal gaps between spans on the same line.
		if i > 0 {
			prevSpan := spans[i-1]
			xGap := span.X - (prevSpan.X + prevSpan.Width)
			avgCharW := span.FontSize * 0.5
			if prevSpan.Width == 0 {
				// Estimate from text length.
				xGap = span.X - prevSpan.X - float64(len(prevSpan.Text))*avgCharW
			}
			if xGap > avgCharW*3 {
				// Significant horizontal gap — add spacing.
				gap := xGap
				if gap > 200 {
					gap = 200
				}
				if style != "" {
					style += fmt.Sprintf(";margin-left:%.0fpx", gap)
				} else {
					style = fmt.Sprintf("margin-left:%.0fpx", gap)
				}
			}
		}

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

	if span.FontSize > 0 {
		parts = append(parts, fmt.Sprintf("font-size:%.1fpx", span.FontSize))
	}

	if span.FontName != "" {
		cssFont := pdfFontToCSS(span.FontName)
		if cssFont != "" {
			parts = append(parts, fmt.Sprintf("font-family:%s", cssFont))
		}
	}

	if span.Bold {
		parts = append(parts, "font-weight:bold")
	}

	if span.Italic {
		parts = append(parts, "font-style:italic")
	}

	// Color — always preserve non-black colors.
	if span.Color[0] > 0.01 || span.Color[1] > 0.01 || span.Color[2] > 0.01 {
		parts = append(parts, fmt.Sprintf("color:%s", rgbToCSS(span.Color)))
	}

	return strings.Join(parts, ";")
}

// pdfFontToCSS maps a PDF font name to a CSS font-family value.
func pdfFontToCSS(pdfFont string) string {
	lower := strings.ToLower(pdfFont)
	lower = strings.TrimPrefix(lower, "/")

	switch {
	case strings.Contains(lower, "helvetica") || strings.Contains(lower, "arial"):
		return "Helvetica, Arial, sans-serif"
	case strings.Contains(lower, "times"):
		return "'Times New Roman', Times, serif"
	case strings.Contains(lower, "courier") || strings.Contains(lower, "mono"):
		return "'Courier New', Courier, monospace"
	case strings.Contains(lower, "symbol"):
		return "Symbol, serif"
	case strings.Contains(lower, "zapf") || strings.Contains(lower, "dingbat"):
		return "ZapfDingbats, serif"
	default:
		name := strings.TrimPrefix(pdfFont, "/")
		if idx := strings.Index(name, "+"); idx >= 0 && idx <= 6 {
			name = name[idx+1:]
		}
		for _, suffix := range []string{"-Bold", "-Italic", "-BoldItalic", "-Oblique", "-BoldOblique", ",Bold", ",Italic", ",BoldItalic"} {
			name = strings.TrimSuffix(name, suffix)
		}
		return fmt.Sprintf("'%s', sans-serif", name)
	}
}

func (b *htmlBuilder) renderTable(sb *strings.Builder, table DetectedTable, linkMap map[spanKey]string) {
	sb.WriteString("<table>\n")
	for r, row := range table.Cells {
		sb.WriteString("<tr>")
		tag := "td"
		if r == 0 {
			tag = "th"
		}
		for _, cell := range row {
			sb.WriteString(fmt.Sprintf("<%s>", tag))
			if len(cell.Spans) > 0 {
				b.renderSpans(sb, cell.Spans, linkMap)
			} else {
				sb.WriteString(html.EscapeString(cell.Text))
			}
			sb.WriteString(fmt.Sprintf("</%s>", tag))
		}
		sb.WriteString("</tr>\n")
	}
	sb.WriteString("</table>\n")
}

type spanKey struct {
	x, y float64
}

func buildLinkMap(links []ExtractedLink, lines []Line) map[spanKey]string {
	m := make(map[spanKey]string)
	for _, link := range links {
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

func spanInRect(span StyledSpan, rect [4]float64) bool {
	x1 := math.Min(rect[0], rect[2])
	y1 := math.Min(rect[1], rect[3])
	x2 := math.Max(rect[0], rect[2])
	y2 := math.Max(rect[1], rect[3])
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

func avgFontSize(spans []StyledSpan) float64 {
	if len(spans) == 0 {
		return 0
	}
	var total float64
	for _, s := range spans {
		total += s.FontSize
	}
	return total / float64(len(spans))
}
