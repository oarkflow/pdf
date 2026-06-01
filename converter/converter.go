package converter

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/oarkflow/pdf/reader"
)

// Converter converts PDF documents to HTML.
type Converter struct {
	reader     *reader.Reader
	options    ConvertOptions
	onProgress ProgressFunc
}

// New creates a Converter from PDF data.
func New(data []byte, opts ConvertOptions) (*Converter, error) {
	var r *reader.Reader
	var err error
	if opts.Password != "" {
		r, err = reader.OpenWithPassword(data, opts.Password)
	} else {
		r, err = reader.Open(data)
	}
	if err != nil {
		return nil, fmt.Errorf("converter: %w", err)
	}
	if opts.Mode == "" {
		opts.Mode = "reflowed"
	}
	return &Converter{reader: r, options: opts}, nil
}

// NewFromFile creates a Converter from a PDF file.
func NewFromFile(path string, opts ConvertOptions) (*Converter, error) {
	var r *reader.Reader
	var err error
	if opts.Password != "" {
		r, err = reader.OpenFileWithPassword(path, opts.Password)
	} else {
		r, err = reader.OpenFile(path)
	}
	if err != nil {
		return nil, fmt.Errorf("converter: %w", err)
	}
	if opts.Mode == "" {
		opts.Mode = "reflowed"
	}
	return &Converter{reader: r, options: opts}, nil
}

// SetProgressCallback sets a function to be called during conversion.
func (c *Converter) SetProgressCallback(fn ProgressFunc) {
	c.onProgress = fn
}

// NumPages returns the total number of pages in the PDF.
func (c *Converter) NumPages() int {
	return c.reader.NumPages()
}

// Metadata returns the PDF document metadata.
func (c *Converter) Metadata() map[string]string {
	return c.reader.Metadata()
}

// Convert performs the full PDF-to-HTML conversion.
func (c *Converter) Convert() (*ConvertResult, error) {
	pages := c.pageNumbers()
	if err := c.validatePageNumbers(pages); err != nil {
		return nil, err
	}
	totalPages := len(pages)
	result := &ConvertResult{
		Pages:    make([]PageResult, 0, totalPages),
		Metadata: c.reader.Metadata(),
	}

	for _, pageNum := range pages {
		if c.onProgress != nil {
			c.onProgress(pageNum, totalPages)
		}

		pageResult, err := c.ConvertPage(pageNum)
		if err != nil {
			// Non-fatal: skip pages that fail (e.g. FlateDecode errors) and continue.
			result.Pages = append(result.Pages, PageResult{
				PageNum: pageNum,
				Width:   612, // default US Letter
				Height:  792,
			})
			continue
		}
		result.Pages = append(result.Pages, *pageResult)
	}

	// Generate HTML from all pages.
	builder := newHTMLBuilder(c.options.Mode)
	result.HTML = builder.Build(result.Pages, result.Metadata)
	result.Text = BuildText(result.Pages)

	return result, nil
}

// ConvertText extracts plain text from the selected pages.
func (c *Converter) ConvertText() (string, error) {
	pages := c.pageNumbers()
	if err := c.validatePageNumbers(pages); err != nil {
		return "", err
	}
	results := make([]PageResult, 0, len(pages))
	for _, pageNum := range pages {
		if c.onProgress != nil {
			c.onProgress(pageNum, len(pages))
		}
		pageResult, err := c.ConvertPage(pageNum)
		if err != nil {
			results = append(results, PageResult{PageNum: pageNum})
			continue
		}
		results = append(results, *pageResult)
	}
	return BuildText(results), nil
}

func (c *Converter) pageNumbers() []int {
	if c.options.Pages != nil {
		return c.options.Pages
	}
	pages := make([]int, c.reader.NumPages())
	for i := range pages {
		pages[i] = i
	}
	return pages
}

func (c *Converter) validatePageNumbers(pages []int) error {
	for _, pageNum := range pages {
		if pageNum < 0 || pageNum >= c.reader.NumPages() {
			return fmt.Errorf("page %d out of range [1, %d]", pageNum+1, c.reader.NumPages())
		}
	}
	return nil
}

// ConvertPage converts a single PDF page.
func (c *Converter) ConvertPage(n int) (*PageResult, error) {
	page, err := c.reader.Page(n)
	if err != nil {
		return nil, err
	}

	resolver := c.reader.GetResolver()

	// Extract styled text.
	ext := newStyledExtractor(resolver)
	if err := ext.loadFonts(page.Resources); err != nil {
		return nil, fmt.Errorf("loading fonts: %w", err)
	}
	ext.parse(page.Contents)

	// Page dimensions.
	width := page.MediaBox[2] - page.MediaBox[0]
	height := page.MediaBox[3] - page.MediaBox[1]

	// Reconstruct lines from spans.
	lines := reconstructLines(ext.spans, height)

	// Detect headings.
	lines = detectHeadings(lines)

	// Group into paragraphs.
	paragraphs := groupParagraphs(lines)

	result := &PageResult{
		PageNum:    n,
		Width:      width,
		Height:     height,
		Lines:      lines,
		Paragraphs: paragraphs,
	}

	// Extract images if requested.
	if c.options.ExtractImages {
		result.Images = extractImages(resolver, page.Resources, ext.images, n)
	}

	// Extract links.
	pageDict, err := c.reader.PageDict(n)
	if err == nil {
		result.Links = extractLinks(resolver, pageDict, n)
	}

	// Detect tables if requested.
	if c.options.DetectTables {
		result.Tables = detectTables(lines)
	}

	return result, nil
}

// reconstructLines groups spans into lines based on Y position.
func reconstructLines(spans []StyledSpan, pageHeight float64) []Line {
	if len(spans) == 0 {
		return nil
	}

	// Sort by Y descending (PDF Y=0 is bottom), then X ascending.
	sorted := make([]StyledSpan, len(spans))
	copy(sorted, spans)
	sort.Slice(sorted, func(i, j int) bool {
		if math.Abs(sorted[i].Y-sorted[j].Y) > sorted[i].FontSize*0.3 {
			return sorted[i].Y > sorted[j].Y // higher Y first (top of page)
		}
		return sorted[i].X < sorted[j].X
	})

	// Group spans into lines by Y tolerance.
	var lines []Line
	var currentLine Line
	currentY := sorted[0].Y

	for _, span := range sorted {
		tolerance := span.FontSize * 0.3
		if tolerance < 1 {
			tolerance = 1
		}

		if math.Abs(span.Y-currentY) > tolerance && len(currentLine.Spans) > 0 {
			currentLine.Y = currentY
			lines = append(lines, currentLine)
			currentLine = Line{}
			currentY = span.Y
		}
		currentLine.Spans = append(currentLine.Spans, span)
	}
	if len(currentLine.Spans) > 0 {
		currentLine.Y = currentY
		lines = append(lines, currentLine)
	}

	return lines
}

// detectHeadings marks lines as headings based on font size analysis.
func detectHeadings(lines []Line) []Line {
	if len(lines) == 0 {
		return lines
	}

	// Find the most common font size (body text size).
	sizeCount := make(map[float64]int)
	for _, line := range lines {
		for _, span := range line.Spans {
			rounded := math.Round(span.FontSize*10) / 10
			sizeCount[rounded]++
		}
	}

	var bodySize float64
	maxCount := 0
	for size, count := range sizeCount {
		if count > maxCount {
			maxCount = count
			bodySize = size
		}
	}

	if bodySize == 0 {
		return lines
	}

	// Mark lines with significantly larger font as headings.
	for i := range lines {
		if len(lines[i].Spans) == 0 {
			continue
		}
		avgSize := avgFontSize(lines[i].Spans)
		ratio := avgSize / bodySize

		if ratio >= 2.0 {
			lines[i].IsHeading = true
			lines[i].Level = 1
		} else if ratio >= 1.7 {
			lines[i].IsHeading = true
			lines[i].Level = 2
		} else if ratio >= 1.4 {
			lines[i].IsHeading = true
			lines[i].Level = 3
		} else if ratio >= 1.2 {
			lines[i].IsHeading = true
			lines[i].Level = 4
		} else if isBoldLine(lines[i]) && ratio >= 1.05 {
			lines[i].IsHeading = true
			lines[i].Level = 5
		}
	}

	return lines
}

// avgFontSize is in html_builder.go

func isBoldLine(line Line) bool {
	for _, s := range line.Spans {
		if !s.Bold {
			return false
		}
	}
	return len(line.Spans) > 0
}

// groupParagraphs groups consecutive non-heading lines into paragraphs.
func groupParagraphs(lines []Line) []Paragraph {
	if len(lines) == 0 {
		return nil
	}

	var paragraphs []Paragraph
	var current Paragraph

	for i, line := range lines {
		if line.IsHeading {
			// Flush current paragraph.
			if len(current.Lines) > 0 {
				paragraphs = append(paragraphs, current)
				current = Paragraph{}
			}
			paragraphs = append(paragraphs, Paragraph{
				Lines:     []Line{line},
				IsHeading: true,
				Level:     line.Level,
			})
			continue
		}

		// Check for paragraph break: large gap between lines.
		if i > 0 && len(current.Lines) > 0 {
			prevY := current.Lines[len(current.Lines)-1].Y
			gap := math.Abs(prevY - line.Y)
			fontSize := avgFontSize(line.Spans)
			if fontSize == 0 {
				fontSize = 12
			}
			if gap > fontSize*1.8 {
				// Paragraph break.
				paragraphs = append(paragraphs, current)
				current = Paragraph{}
			}
		}

		current.Lines = append(current.Lines, line)
	}

	if len(current.Lines) > 0 {
		paragraphs = append(paragraphs, current)
	}

	return paragraphs
}

// BuildText converts extracted page results to plain text in reconstructed reading order.
func BuildText(pages []PageResult) string {
	var doc strings.Builder
	for i, page := range pages {
		if i > 0 {
			doc.WriteString("\n\n")
		}
		doc.WriteString(BuildPageText(page))
	}
	return strings.TrimRight(doc.String(), "\n")
}

// BuildPageText converts one extracted page to plain text.
func BuildPageText(page PageResult) string {
	var out strings.Builder
	prevY := 0.0
	prevFontSize := 12.0
	wroteLine := false

	for _, line := range page.Lines {
		if len(line.Spans) == 0 {
			continue
		}
		if wroteLine {
			fontSize := avgFontSize(line.Spans)
			if fontSize == 0 {
				fontSize = prevFontSize
			}
			yGap := math.Abs(prevY - line.Y)
			lineHeight := math.Max(fontSize, prevFontSize) * 1.2
			out.WriteByte('\n')
			if yGap > lineHeight*1.7 {
				out.WriteByte('\n')
			}
		}
		writeLineText(&out, line.Spans)
		prevY = line.Y
		prevFontSize = avgFontSize(line.Spans)
		if prevFontSize == 0 {
			prevFontSize = 12
		}
		wroteLine = true
	}

	return strings.TrimRight(out.String(), "\n")
}

func writeLineText(out *strings.Builder, spans []StyledSpan) {
	wrote := false
	var prev StyledSpan
	for _, span := range spans {
		text := strings.TrimSpace(span.Text)
		if text == "" {
			continue
		}
		if wrote && needsPlainTextGap(prev, span) {
			out.WriteByte(' ')
		}
		out.WriteString(text)
		prev = span
		wrote = true
	}
}

func needsPlainTextGap(prev, cur StyledSpan) bool {
	if strings.HasSuffix(prev.Text, " ") || strings.HasPrefix(cur.Text, " ") {
		return false
	}
	xGap := cur.X - (prev.X + prev.Width)
	avgCharW := cur.FontSize * 0.35
	if avgCharW <= 0 {
		avgCharW = 3
	}
	if prev.Width == 0 {
		xGap = cur.X - prev.X - float64(len([]rune(prev.Text)))*avgCharW
	}
	return xGap > avgCharW
}
