package font

// Face is the interface that all font implementations must satisfy.
type Face interface {
	PostScriptName() string
	UnitsPerEm() int
	GlyphIndex(r rune) uint16
	GlyphAdvance(id uint16) int
	Ascent() int
	Descent() int
	BBox() [4]int
	CapHeight() int
	StemV() int
	ItalicAngle() float64
	Kern(left, right uint16) int
	Flags() uint32
	RawData() []byte
	NumGlyphs() int
}

// ShapedGlyph is a positioned glyph produced by an OpenType shaper.
// Position and advance values are in font units.
type ShapedGlyph struct {
	GlyphID  uint16
	Cluster  string
	XAdvance int
	YAdvance int
	XOffset  int
	YOffset  int
}

// Shaper is implemented by font faces that can perform OpenType shaping.
type Shaper interface {
	ShapeText(text string) ([]ShapedGlyph, bool)
}

// Style represents a font style variant.
type Style int

const (
	StyleRegular Style = iota
	StyleBold
	StyleItalic
	StyleBoldItalic
)
