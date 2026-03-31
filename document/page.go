package document

import (
	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/layout"
)

// PageSize defines the dimensions of a page in points (1 point = 1/72 inch).
type PageSize struct {
	Width, Height float64
}

var (
	A4     = PageSize{595.28, 841.89}
	A3     = PageSize{841.89, 1190.55}
	A5     = PageSize{419.53, 595.28}
	Letter = PageSize{612, 792}
	Legal  = PageSize{612, 1008}
)

// Margins defines page margins in points.
type Margins struct {
	Top, Right, Bottom, Left float64
}

// DefaultMargins returns 1-inch margins on all sides.
func DefaultMargins() Margins {
	return Margins{72, 72, 72, 72}
}

// Page represents a single PDF page.
type Page struct {
	Size        PageSize
	Resources   *core.PdfDictionary
	Contents    []byte
	Fonts       map[string]int // font name -> object number
	FontEntries map[string]layout.FontEntry
	Images      map[string]layout.ImageEntry
	Annotations []layout.LinkAnnotation
}

// NewPage creates a new page with the given size.
func NewPage(size PageSize) *Page {
	return &Page{
		Size:        size,
		Resources:   core.NewDictionary(),
		Fonts:       make(map[string]int),
		FontEntries: make(map[string]layout.FontEntry),
		Images:      make(map[string]layout.ImageEntry),
	}
}
