package converter

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// ExportDocument is a compact structured representation of converted PDF content.
type ExportDocument struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Pages    []ExportPage      `json:"pages"`
}

// ExportPage is a structured representation of one PDF page.
type ExportPage struct {
	Number int           `json:"number"`
	Width  float64       `json:"width"`
	Height float64       `json:"height"`
	Text   string        `json:"text,omitempty"`
	Lines  []ExportLine  `json:"lines,omitempty"`
	Images []ExportImage `json:"images,omitempty"`
	Links  []ExportLink  `json:"links,omitempty"`
	Tables []ExportTable `json:"tables,omitempty"`
}

// ExportLine is a text line with its source position and spans.
type ExportLine struct {
	Text      string       `json:"text"`
	X         float64      `json:"x,omitempty"`
	Y         float64      `json:"y,omitempty"`
	IsHeading bool         `json:"isHeading,omitempty"`
	Level     int          `json:"level,omitempty"`
	Spans     []ExportSpan `json:"spans,omitempty"`
}

// ExportSpan is a styled text fragment.
type ExportSpan struct {
	Text     string     `json:"text"`
	X        float64    `json:"x,omitempty"`
	Y        float64    `json:"y,omitempty"`
	FontName string     `json:"fontName,omitempty"`
	FontSize float64    `json:"fontSize,omitempty"`
	Bold     bool       `json:"bold,omitempty"`
	Italic   bool       `json:"italic,omitempty"`
	Color    [3]float64 `json:"color,omitempty"`
}

// ExportImage is image metadata without embedding image bytes.
type ExportImage struct {
	MimeType string  `json:"mimeType"`
	X        float64 `json:"x,omitempty"`
	Y        float64 `json:"y,omitempty"`
	Width    float64 `json:"width,omitempty"`
	Height   float64 `json:"height,omitempty"`
}

// ExportLink is a hyperlink annotation.
type ExportLink struct {
	URL  string     `json:"url"`
	Rect [4]float64 `json:"rect"`
}

// ExportTable is a detected table.
type ExportTable struct {
	Rows  int                 `json:"rows"`
	Cols  int                 `json:"cols"`
	Rect  [4]float64          `json:"rect"`
	Cells [][]ExportTableCell `json:"cells,omitempty"`
}

// ExportTableCell is a detected table cell.
type ExportTableCell struct {
	Row     int    `json:"row"`
	Col     int    `json:"col"`
	RowSpan int    `json:"rowSpan,omitempty"`
	ColSpan int    `json:"colSpan,omitempty"`
	Text    string `json:"text"`
}

// ConvertJSON converts the selected PDF pages to structured JSON.
func (c *Converter) ConvertJSON() ([]byte, error) {
	result, err := c.Convert()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(BuildExportDocument(result), "", "  ")
}

// ConvertMarkdown converts the selected PDF pages to Markdown.
func (c *Converter) ConvertMarkdown() (string, error) {
	result, err := c.Convert()
	if err != nil {
		return "", err
	}
	return BuildMarkdown(result.Pages), nil
}

// ExtractImages extracts images from selected pages.
func (c *Converter) ExtractImages() ([]ExtractedImage, error) {
	old := c.options.ExtractImages
	c.options.ExtractImages = true
	defer func() { c.options.ExtractImages = old }()

	pages := c.pageNumbers()
	if err := c.validatePageNumbers(pages); err != nil {
		return nil, err
	}
	var images []ExtractedImage
	for _, pageNum := range pages {
		page, err := c.ConvertPage(pageNum)
		if err != nil {
			return nil, err
		}
		images = append(images, page.Images...)
	}
	return images, nil
}

// BuildExportDocument creates structured content from a conversion result.
func BuildExportDocument(result *ConvertResult) ExportDocument {
	if result == nil {
		return ExportDocument{}
	}
	doc := ExportDocument{Metadata: result.Metadata}
	for _, page := range result.Pages {
		doc.Pages = append(doc.Pages, buildExportPage(page))
	}
	return doc
}

func buildExportPage(page PageResult) ExportPage {
	out := ExportPage{
		Number: page.PageNum + 1,
		Width:  page.Width,
		Height: page.Height,
		Text:   BuildPageText(page),
	}
	for _, line := range page.Lines {
		exportLine := ExportLine{
			Text:      lineText(line.Spans),
			Y:         line.Y,
			IsHeading: line.IsHeading,
			Level:     line.Level,
		}
		if len(line.Spans) > 0 {
			exportLine.X = line.Spans[0].X
		}
		for _, span := range line.Spans {
			exportLine.Spans = append(exportLine.Spans, ExportSpan{
				Text:     span.Text,
				X:        span.X,
				Y:        span.Y,
				FontName: span.FontName,
				FontSize: span.FontSize,
				Bold:     span.Bold,
				Italic:   span.Italic,
				Color:    span.Color,
			})
		}
		out.Lines = append(out.Lines, exportLine)
	}
	for _, img := range page.Images {
		out.Images = append(out.Images, ExportImage{
			MimeType: img.MimeType,
			X:        img.X,
			Y:        img.Y,
			Width:    img.Width,
			Height:   img.Height,
		})
	}
	for _, link := range page.Links {
		out.Links = append(out.Links, ExportLink{URL: link.URL, Rect: link.Rect})
	}
	for _, table := range page.Tables {
		exportTable := ExportTable{Rows: table.Rows, Cols: table.Cols, Rect: table.Rect}
		for _, row := range table.Cells {
			var cells []ExportTableCell
			for _, cell := range row {
				cells = append(cells, ExportTableCell{
					Row:     cell.Row,
					Col:     cell.Col,
					RowSpan: cell.RowSpan,
					ColSpan: cell.ColSpan,
					Text:    cell.Text,
				})
			}
			exportTable.Cells = append(exportTable.Cells, cells)
		}
		out.Tables = append(out.Tables, exportTable)
	}
	return out
}

// BuildMarkdown converts page results to Markdown.
func BuildMarkdown(pages []PageResult) string {
	var out strings.Builder
	for i, page := range pages {
		if i > 0 {
			out.WriteString("\n\n---\n\n")
		}
		for _, line := range page.Lines {
			text := strings.TrimSpace(lineText(line.Spans))
			if text == "" {
				continue
			}
			if line.IsHeading {
				level := line.Level
				if level < 1 || level > 6 {
					level = 2
				}
				out.WriteString(strings.Repeat("#", level))
				out.WriteByte(' ')
				out.WriteString(markdownEscape(text))
				out.WriteString("\n\n")
				continue
			}
			out.WriteString(markdownEscape(text))
			out.WriteByte('\n')
		}
		for _, table := range page.Tables {
			writeMarkdownTable(&out, table)
		}
	}
	return strings.TrimSpace(out.String()) + "\n"
}

func lineText(spans []StyledSpan) string {
	var out strings.Builder
	writeLineText(&out, spans)
	return out.String()
}

func writeMarkdownTable(out *strings.Builder, table DetectedTable) {
	if len(table.Cells) == 0 {
		return
	}
	out.WriteByte('\n')
	for r, row := range table.Cells {
		out.WriteByte('|')
		for _, cell := range row {
			out.WriteByte(' ')
			out.WriteString(markdownEscape(strings.TrimSpace(cell.Text)))
			out.WriteString(" |")
		}
		out.WriteByte('\n')
		if r == 0 {
			out.WriteByte('|')
			for range row {
				out.WriteString(" --- |")
			}
			out.WriteByte('\n')
		}
	}
	out.WriteByte('\n')
}

func markdownEscape(s string) string {
	s = html.UnescapeString(s)
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"#", "\\#",
		"|", "\\|",
	)
	return replacer.Replace(s)
}

func imageExtension(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ".bin"
	}
}

// ImageFilename returns a stable filename for an extracted image.
func ImageFilename(img ExtractedImage, index int) string {
	return fmt.Sprintf("page_%03d_image_%03d%s", img.PageNum+1, index+1, imageExtension(img.MimeType))
}
