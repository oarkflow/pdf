package font

import (
	"bytes"
	"os"

	gtfont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/harfbuzz"
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
	hbFace   *gtfont.Face
	hbFont   *harfbuzz.Font
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
	var hbFace *gtfont.Face
	var hbFont *harfbuzz.Font
	if parsedFace, parseErr := gtfont.ParseTTF(bytes.NewReader(data)); parseErr == nil {
		hbFace = parsedFace
		hbFont = harfbuzz.NewFont(parsedFace)
	}
	return &TrueTypeFont{
		sfntFont: f,
		name:     name,
		data:     data,
		upem:     int(upem),
		hbFace:   hbFace,
		hbFont:   hbFont,
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

func (f *TrueTypeFont) StemV() int           { return 80 }
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

// ShapeText converts Unicode text into positioned glyph IDs using OpenType
// shaping. It returns false when shaping is unavailable for this font.
func (f *TrueTypeFont) ShapeText(text string) ([]ShapedGlyph, bool) {
	if f == nil || f.hbFont == nil || text == "" {
		return nil, false
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return nil, false
	}

	buf := harfbuzz.NewBuffer()
	buf.AddRunes(runes, 0, len(runes))
	buf.GuessSegmentProperties()
	buf.Shape(f.hbFont, nil)
	if len(buf.Info) == 0 || len(buf.Info) != len(buf.Pos) {
		return nil, false
	}

	clusterEnds := make(map[int]int)
	for _, info := range buf.Info {
		start := info.Cluster
		if start < 0 || start >= len(runes) {
			continue
		}
		end := len(runes)
		for _, other := range buf.Info {
			if other.Cluster > start && other.Cluster < end {
				end = other.Cluster
			}
		}
		clusterEnds[start] = end
	}

	seenCluster := make(map[int]bool)
	glyphs := make([]ShapedGlyph, 0, len(buf.Info))
	for i, info := range buf.Info {
		pos := buf.Pos[i]
		cluster := ""
		if info.Cluster >= 0 && info.Cluster < len(runes) && !seenCluster[info.Cluster] {
			end := clusterEnds[info.Cluster]
			if end <= info.Cluster || end > len(runes) {
				end = info.Cluster + 1
			}
			cluster = string(runes[info.Cluster:end])
			seenCluster[info.Cluster] = true
		}
		glyphs = append(glyphs, ShapedGlyph{
			GlyphID:  uint16(info.Glyph),
			Cluster:  cluster,
			XAdvance: int(pos.XAdvance),
			YAdvance: int(pos.YAdvance),
			XOffset:  int(pos.XOffset),
			YOffset:  int(pos.YOffset),
		})
	}
	return glyphs, true
}

func fix266ToInt(v fixed.Int26_6) int {
	return int((v + 32) >> 6)
}
