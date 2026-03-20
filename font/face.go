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

// Style represents a font style variant.
type Style int

const (
	StyleRegular   Style = iota
	StyleBold
	StyleItalic
	StyleBoldItalic
)
