package converter

import (
	"fmt"
	"math"
	"sort"

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
	pages := c.options.Pages
	if pages == nil {
		pages = make([]int, c.reader.NumPages())
		for i := range pages {
			pages[i] = i
		}
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

	return result, nil
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
