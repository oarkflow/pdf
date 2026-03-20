package font

import (
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// TrueTypeFont implements Face for TrueType/OpenType fonts.
type TrueTypeFont struct {
	sfntFont *sfnt.Font
	name     string
	data     []byte
	upem     int
}

// LoadTrueType parses a TrueType or OpenType font from raw bytes.
func LoadTrueType(data []byte) (*TrueTypeFont, error) {
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}
	var buf sfnt.Buffer
	name, err := f.Name(&buf, sfnt.NameIDPostScript)
	if err != nil {
		name = "Unknown"
	}
	upem := f.UnitsPerEm()
	return &TrueTypeFont{
		sfntFont: f,
		name:     name,
		data:     data,
		upem:     int(upem),
	}, nil
}

// LoadTrueTypeFile loads a TrueType font from a file path.
func LoadTrueTypeFile(path string) (*TrueTypeFont, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadTrueType(data)
}

func (f *TrueTypeFont) ppem() fixed.Int26_6 {
	return fixed.I(f.upem)
}

func (f *TrueTypeFont) PostScriptName() string { return f.name }
func (f *TrueTypeFont) UnitsPerEm() int        { return f.upem }

func (f *TrueTypeFont) GlyphIndex(r rune) uint16 {
	var buf sfnt.Buffer
	idx, err := f.sfntFont.GlyphIndex(&buf, r)
	if err != nil {
		return 0
	}
	return uint16(idx)
}

func (f *TrueTypeFont) GlyphAdvance(id uint16) int {
	var buf sfnt.Buffer
	adv, err := f.sfntFont.GlyphAdvance(&buf, sfnt.GlyphIndex(id), f.ppem(), font.HintingNone)
	if err != nil {
		return 0
	}
	return fix266ToInt(adv)
}

func (f *TrueTypeFont) Ascent() int {
	m, err := f.sfntFont.Metrics(nil, f.ppem(), font.HintingNone)
	if err != nil {
		return 0
	}
	return fix266ToInt(m.Ascent)
}

func (f *TrueTypeFont) Descent() int {
	m, err := f.sfntFont.Metrics(nil, f.ppem(), font.HintingNone)
	if err != nil {
		return 0
	}
	return -fix266ToInt(m.Descent)
}

func (f *TrueTypeFont) BBox() [4]int {
	b, err := f.sfntFont.Bounds(nil, f.ppem(), font.HintingNone)
	if err != nil {
		return [4]int{}
	}
	return [4]int{
		b.Min.X.Floor(),
		b.Min.Y.Floor(),
		b.Max.X.Ceil(),
		b.Max.Y.Ceil(),
	}
}

func (f *TrueTypeFont) CapHeight() int {
	m, err := f.sfntFont.Metrics(nil, f.ppem(), font.HintingNone)
	if err != nil {
		return 0
	}
	return fix266ToInt(m.CapHeight)
}

func (f *TrueTypeFont) StemV() int    { return 80 }
func (f *TrueTypeFont) ItalicAngle() float64 { return 0 }

func (f *TrueTypeFont) Kern(left, right uint16) int {
	var buf sfnt.Buffer
	k, err := f.sfntFont.Kern(&buf, sfnt.GlyphIndex(left), sfnt.GlyphIndex(right), f.ppem(), font.HintingNone)
	if err != nil {
		return 0
	}
	return fix266ToInt(k)
}

func (f *TrueTypeFont) Flags() uint32   { return 32 }
func (f *TrueTypeFont) RawData() []byte { return f.data }
func (f *TrueTypeFont) NumGlyphs() int  { return f.sfntFont.NumGlyphs() }

func fix266ToInt(v fixed.Int26_6) int {
	return int((v + 32) >> 6)
}
