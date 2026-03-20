package font

import "fmt"

// StandardFont implements Face for the 14 standard PDF fonts.
type StandardFont struct {
	name        string
	widths      [256]int
	ascent      int
	descent     int
	bbox        [4]int
	capHeight   int
	stemV       int
	italicAngle float64
	flags       uint32
}

// standardFontNames lists all 14 standard PDF font names.
var standardFontNames = []string{
	"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
	"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
	"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
	"Symbol", "ZapfDingbats",
}

// IsStandardFont returns true if name is one of the 14 standard PDF fonts.
func IsStandardFont(name string) bool {
	for _, n := range standardFontNames {
		if n == name {
			return true
		}
	}
	return false
}

// NewStandardFont returns a StandardFont for one of the 14 standard PDF fonts.
func NewStandardFont(name string) (*StandardFont, error) {
	m, ok := standardMetrics[name]
	if !ok {
		return nil, fmt.Errorf("font: %q is not a standard PDF font", name)
	}
	return &m, nil
}

func (f *StandardFont) PostScriptName() string  { return f.name }
func (f *StandardFont) UnitsPerEm() int          { return 1000 }
func (f *StandardFont) Ascent() int               { return f.ascent }
func (f *StandardFont) Descent() int              { return f.descent }
func (f *StandardFont) BBox() [4]int              { return f.bbox }
func (f *StandardFont) CapHeight() int            { return f.capHeight }
func (f *StandardFont) StemV() int                { return f.stemV }
func (f *StandardFont) ItalicAngle() float64      { return f.italicAngle }
func (f *StandardFont) Flags() uint32             { return f.flags }
func (f *StandardFont) RawData() []byte           { return nil }
func (f *StandardFont) NumGlyphs() int            { return 256 }
func (f *StandardFont) Kern(_, _ uint16) int      { return 0 }

// GlyphIndex maps a rune to its WinAnsiEncoding byte index.
func (f *StandardFont) GlyphIndex(r rune) uint16 {
	if idx, ok := winAnsiEncoding[r]; ok {
		return uint16(idx)
	}
	// Direct mapping for runes 0-127 and 160-255 that match Latin-1.
	if r >= 0 && r <= 255 {
		// Exclude the 128-159 range which is special in WinAnsi.
		if r < 128 || r > 159 {
			return uint16(r)
		}
	}
	return 0
}

// GlyphAdvance returns the advance width for the given glyph index.
func (f *StandardFont) GlyphAdvance(id uint16) int {
	if int(id) >= 256 {
		return 0
	}
	return f.widths[id]
}

// winAnsiEncoding maps runes in the 128-159 range to their WinAnsiEncoding positions.
var winAnsiEncoding = map[rune]byte{
	0x20AC: 128, // Euro sign
	0x201A: 130, // Single low-9 quotation mark
	0x0192: 131, // Latin small letter f with hook
	0x201E: 132, // Double low-9 quotation mark
	0x2026: 133, // Horizontal ellipsis
	0x2020: 134, // Dagger
	0x2021: 135, // Double dagger
	0x02C6: 136, // Modifier letter circumflex accent
	0x2030: 137, // Per mille sign
	0x0160: 138, // Latin capital letter S with caron
	0x2039: 139, // Single left-pointing angle quotation mark
	0x0152: 140, // Latin capital ligature OE
	0x017D: 142, // Latin capital letter Z with caron
	0x2018: 145, // Left single quotation mark
	0x2019: 146, // Right single quotation mark
	0x201C: 147, // Left double quotation mark
	0x201D: 148, // Right double quotation mark
	0x2022: 149, // Bullet
	0x2013: 150, // En dash
	0x2014: 151, // Em dash
	0x02DC: 152, // Small tilde
	0x2122: 153, // Trade mark sign
	0x0161: 154, // Latin small letter s with caron
	0x203A: 155, // Single right-pointing angle quotation mark
	0x0153: 156, // Latin small ligature oe
	0x017E: 158, // Latin small letter z with caron
	0x0178: 159, // Latin capital letter Y with diaeresis
}
