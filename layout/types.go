package layout

import (
	"fmt"

	pdffont "github.com/oarkflow/pdf/font"
	pdfimage "github.com/oarkflow/pdf/image"
)

// LayoutArea represents available space for layout.
type LayoutArea struct {
	Width, Height float64
}

// LayoutStatus indicates how much of an element was placed.
type LayoutStatus int

const (
	LayoutFull    LayoutStatus = iota // element fully placed
	LayoutPartial                     // partially placed, overflow remains
	LayoutNothing                     // nothing could be placed (e.g., not enough space)
)

// LayoutPlan is the result of planning layout for an element.
type LayoutPlan struct {
	Status   LayoutStatus
	Consumed float64 // vertical space consumed
	Blocks   []PlacedBlock
	Overflow Element // remainder for next page (nil if Full)
}

// PlacedBlock is a positioned rectangle with a draw function.
type PlacedBlock struct {
	X, Y, Width, Height float64
	Draw                func(ctx *DrawContext, x, topY float64)
	Tag                 string // PDF structure tag (P, H1, Table, etc.)
	AltText             string
	Children            []PlacedBlock
}

// LinkAnnotation represents a clickable link region on a page.
type LinkAnnotation struct {
	X1, Y1, X2, Y2 float64 // rectangle in PDF coordinates (bottom-left origin)
	URI            string
}

// DrawContext provides drawing capabilities during rendering.
type DrawContext struct {
	ContentStream []byte
	Fonts         map[string]FontEntry
	Images        map[string]ImageEntry
	Links         []LinkAnnotation
	ExtGStates    map[string]ExtGState
	PageWidth     float64
	PageHeight    float64
}

// NewDrawContext creates a new draw context for a page.
func NewDrawContext(pageWidth, pageHeight float64) *DrawContext {
	return &DrawContext{
		Fonts:      make(map[string]FontEntry),
		Images:     make(map[string]ImageEntry),
		ExtGStates: make(map[string]ExtGState),
		PageWidth:  pageWidth,
		PageHeight: pageHeight,
	}
}

// Write appends raw content stream operators.
func (ctx *DrawContext) Write(data []byte) { ctx.ContentStream = append(ctx.ContentStream, data...) }

// WriteString appends a string as raw content stream operators.
func (ctx *DrawContext) WriteString(s string) { ctx.ContentStream = append(ctx.ContentStream, s...) }

// BeginMarkedContent writes a BDC operator with the given tag and MCID.
func (ctx *DrawContext) BeginMarkedContent(tag string, mcid int) {
	ctx.WriteString(fmt.Sprintf("/%s <</MCID %d>> BDC\n", tag, mcid))
}

// EndMarkedContent writes an EMC operator.
func (ctx *DrawContext) EndMarkedContent() {
	ctx.WriteString("EMC\n")
}

// AddLink records a clickable link annotation for the current page.
func (ctx *DrawContext) AddLink(x1, y1, x2, y2 float64, uri string) {
	ctx.Links = append(ctx.Links, LinkAnnotation{X1: x1, Y1: y1, X2: x2, Y2: y2, URI: uri})
}

// EnsureExtGState returns a reusable graphics state resource name for the
// requested fill/stroke alpha values.
func (ctx *DrawContext) EnsureExtGState(fillAlpha, strokeAlpha float64) string {
	for name, gs := range ctx.ExtGStates {
		if gs.FillAlpha == fillAlpha && gs.StrokeAlpha == strokeAlpha {
			return name
		}
	}
	name := fmt.Sprintf("GS%d", len(ctx.ExtGStates)+1)
	ctx.ExtGStates[name] = ExtGState{FillAlpha: fillAlpha, StrokeAlpha: strokeAlpha}
	return name
}

// FontEntry tracks a font used on a page.
type FontEntry struct {
	PDFName   string // /F1, /F2, etc.
	Name      string
	ObjectNum int
	Face      pdffont.Face
	Embedded  *pdffont.EmbeddedFont
}

// ImageEntry tracks an image used on a page.
type ImageEntry struct {
	PDFName   string // /Im1, /Im2, etc.
	ObjectNum int
	Width     int
	Height    int
	Image     *pdfimage.Image
}

// ExtGState tracks a graphics state resource used on a page.
type ExtGState struct {
	FillAlpha   float64
	StrokeAlpha float64
}

// TextRun represents a styled run of text.
type TextRun struct {
	Text      string
	FontName  string
	FontSize  float64
	FontFace  pdffont.Face
	Bold      bool
	Italic    bool
	Color     [3]float64 // RGB 0-1
	Underline bool
	Strike    bool
	Link      string // URL for hyperlinks
}

// Word is a measured word from text shaping.
type Word struct {
	Runs  []TextRun
	Width float64
	Space float64 // trailing space width
}

// Line is a laid-out line of words.
type Line struct {
	Words  []Word
	Width  float64
	Height float64 // line height (max ascent + max descent)
	Ascent float64
}

// Alignment for text and blocks.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
	AlignJustify
)

// BoxModel holds margin, padding, border for div-like elements.
type BoxModel struct {
	MarginTop, MarginRight, MarginBottom, MarginLeft                     float64
	PaddingTop, PaddingRight, PaddingBottom, PaddingLeft                 float64
	BorderTopWidth, BorderRightWidth, BorderBottomWidth, BorderLeftWidth float64
	BorderColor                                                          [3]float64
	BorderTopColor, BorderRightColor, BorderBottomColor, BorderLeftColor [3]float64
	Background                                                           *[3]float64 // nil means transparent
	BackgroundImage                                                      string
	BackgroundPosition                                                   string
	BackgroundSize                                                       string
	BackgroundRepeat                                                     string
	BoxShadow                                                            string
	BorderRadius                                                         float64
	BorderTopLeftRadius, BorderTopRightRadius                            float64
	BorderBottomRightRadius, BorderBottomLeftRadius                      float64
}

// TotalHorizontal returns total horizontal space consumed by the box model.
func (b BoxModel) TotalHorizontal() float64 {
	return b.MarginLeft + b.BorderLeftWidth + b.PaddingLeft + b.PaddingRight + b.BorderRightWidth + b.MarginRight
}

// TotalVertical returns total vertical space consumed by the box model.
func (b BoxModel) TotalVertical() float64 {
	return b.MarginTop + b.BorderTopWidth + b.PaddingTop + b.PaddingBottom + b.BorderBottomWidth + b.MarginBottom
}

// InnerHorizontal returns horizontal space consumed by border and padding (no margin).
func (b BoxModel) InnerHorizontal() float64 {
	return b.BorderLeftWidth + b.PaddingLeft + b.PaddingRight + b.BorderRightWidth
}

// ContentLeft returns the left offset to the content area.
func (b BoxModel) ContentLeft() float64 {
	return b.MarginLeft + b.BorderLeftWidth + b.PaddingLeft
}

// ContentTop returns the top offset to the content area.
func (b BoxModel) ContentTop() float64 {
	return b.MarginTop + b.BorderTopWidth + b.PaddingTop
}
