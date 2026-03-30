package converter

// StyledSpan is a text fragment with full styling information extracted from a PDF.
type StyledSpan struct {
	Text     string
	X, Y     float64
	FontName string
	FontSize float64
	Bold     bool
	Italic   bool
	Color    [3]float64 // RGB 0-1
	Width    float64    // computed advance width
}

// ExtractedImage is an image found in the PDF.
type ExtractedImage struct {
	Data     []byte
	MimeType string // "image/png" or "image/jpeg"
	X, Y     float64
	Width    float64
	Height   float64
	PageNum  int
}

// ExtractedLink is a hyperlink annotation from the PDF.
type ExtractedLink struct {
	URL     string
	Rect    [4]float64 // x1, y1, x2, y2 in PDF coordinates
	PageNum int
}

// TableCell represents a detected table cell.
type TableCell struct {
	Row, Col           int
	RowSpan, ColSpan   int
	Spans              []StyledSpan
	Text               string
}

// DetectedTable is a table found via position heuristics.
type DetectedTable struct {
	Rows  int
	Cols  int
	Cells [][]TableCell
	Rect  [4]float64 // bounding box
}

// Line is a reconstructed line of text from a PDF page.
type Line struct {
	Spans     []StyledSpan
	Y         float64
	IsHeading bool
	Level     int // 1-6 for headings
}

// Paragraph groups consecutive lines into a logical paragraph.
type Paragraph struct {
	Lines     []Line
	IsHeading bool
	Level     int
}

// PageResult holds everything extracted from one PDF page.
type PageResult struct {
	PageNum   int
	Width     float64
	Height    float64
	Lines     []Line
	Paragraphs []Paragraph
	Images    []ExtractedImage
	Links     []ExtractedLink
	Tables    []DetectedTable
}

// ConvertOptions controls conversion behavior.
type ConvertOptions struct {
	Pages         []int  // nil = all pages
	Mode          string // "positioned" or "reflowed" (default: "reflowed")
	ExtractImages bool
	DetectTables  bool
	Password      string // PDF password if encrypted
}

// ConvertResult is the final output of a PDF-to-HTML conversion.
type ConvertResult struct {
	HTML     string
	Pages    []PageResult
	Metadata map[string]string
}

// ProgressFunc is called during conversion to report progress.
type ProgressFunc func(currentPage, totalPages int)
