package font

import (
	"fmt"
	"strings"

	"github.com/oarkflow/pdf/core"
)

// EmbeddedFont tracks glyph usage and produces PDF font objects.
type EmbeddedFont struct {
	Face        Face
	PDFName     string
	usedGlyphs map[uint16]rune
	usedRunes   map[rune]uint16
}

// NewEmbeddedFont creates a new EmbeddedFont wrapper.
func NewEmbeddedFont(face Face, pdfName string) *EmbeddedFont {
	return &EmbeddedFont{
		Face:        face,
		PDFName:     pdfName,
		usedGlyphs:  make(map[uint16]rune),
		usedRunes:   make(map[rune]uint16),
	}
}

// AddRune records a rune as used and returns its glyph ID.
func (ef *EmbeddedFont) AddRune(r rune) uint16 {
	if gid, ok := ef.usedRunes[r]; ok {
		return gid
	}
	gid := ef.Face.GlyphIndex(r)
	// If glyph is missing (0), try common substitutions.
	if gid == 0 {
		if sub, ok := glyphSubstitutions[r]; ok {
			if subGid := ef.Face.GlyphIndex(sub); subGid != 0 {
				gid = subGid
				r = sub
			}
		}
	}
	ef.usedGlyphs[gid] = r
	ef.usedRunes[r] = gid
	return gid
}

// glyphSubstitutions maps characters to fallback alternatives when the
// primary glyph is missing from a font.
var glyphSubstitutions = map[rune]rune{
	'\u00A0': ' ',  // NBSP → regular space
	'\u2002': ' ',  // EN SPACE → regular space
	'\u2003': ' ',  // EM SPACE → regular space
	'\u2009': ' ',  // THIN SPACE → regular space
	'\u200A': ' ',  // HAIR SPACE → regular space
	'\u202F': ' ',  // NARROW NO-BREAK SPACE → regular space
}

// AddString records all runes in s as used.
func (ef *EmbeddedFont) AddString(s string) {
	for _, r := range s {
		ef.AddRune(r)
	}
}

// UsedGlyphs returns the glyph-to-rune mapping.
func (ef *EmbeddedFont) UsedGlyphs() map[uint16]rune {
	return ef.usedGlyphs
}

// IsStandard returns true if the underlying face is a standard PDF font.
func (ef *EmbeddedFont) IsStandard() bool {
	_, ok := ef.Face.(*StandardFont)
	return ok
}

// EncodedString returns the encoded byte string for the PDF Tj operator.
func (ef *EmbeddedFont) EncodedString(s string) string {
	if ef.IsStandard() {
		var buf []byte
		for _, r := range s {
			gid := ef.Face.GlyphIndex(r)
			buf = append(buf, byte(gid))
		}
		return string(buf)
	}
	var buf []byte
	for _, r := range s {
		gid := ef.AddRune(r)
		buf = append(buf, byte(gid>>8), byte(gid))
	}
	return string(buf)
}

// BuildObjects returns the PDF indirect objects needed for this font.
func (ef *EmbeddedFont) BuildObjects(nextObjNum func() int) []core.PdfIndirectObject {
	if ef.IsStandard() {
		return ef.buildStandardObjects(nextObjNum)
	}
	return ef.buildCIDFontObjects(nextObjNum)
}

func (ef *EmbeddedFont) buildStandardObjects(nextObjNum func() int) []core.PdfIndirectObject {
	num := nextObjNum()
	d := core.NewDictionary()
	d.Set("Type", core.PdfName("Font"))
	d.Set("Subtype", core.PdfName("Type1"))
	d.Set("BaseFont", core.PdfName(ef.Face.PostScriptName()))
	d.Set("Encoding", core.PdfName("WinAnsiEncoding"))

	return []core.PdfIndirectObject{{
		Reference: core.PdfIndirectReference{ObjectNumber: num},
		Object:    d,
	}}
}

func (ef *EmbeddedFont) buildCIDFontObjects(nextObjNum func() int) []core.PdfIndirectObject {
	var objects []core.PdfIndirectObject
	face := ef.Face

	// Font descriptor.
	descNum := nextObjNum()
	bbox := face.BBox()
	descDict := core.NewDictionary()
	descDict.Set("Type", core.PdfName("FontDescriptor"))
	descDict.Set("FontName", core.PdfName(face.PostScriptName()))
	descDict.Set("Flags", core.PdfInteger(face.Flags()))
	descDict.Set("FontBBox", core.PdfArray{
		core.PdfInteger(bbox[0]), core.PdfInteger(bbox[1]),
		core.PdfInteger(bbox[2]), core.PdfInteger(bbox[3]),
	})
	descDict.Set("ItalicAngle", core.PdfNumber(face.ItalicAngle()))
	descDict.Set("Ascent", core.PdfInteger(face.Ascent()))
	descDict.Set("Descent", core.PdfInteger(face.Descent()))
	descDict.Set("CapHeight", core.PdfInteger(face.CapHeight()))
	descDict.Set("StemV", core.PdfInteger(face.StemV()))

	descObj := core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: descNum},
		Object:    descDict,
	}

	// Font file stream.
	rawData := face.RawData()
	if rawData != nil {
		ffNum := nextObjNum()
		stream := core.NewStream(rawData)
		stream.Dictionary.Set("Length1", core.PdfInteger(len(rawData)))
		ffObj := core.PdfIndirectObject{
			Reference: core.PdfIndirectReference{ObjectNumber: ffNum},
			Object:    stream,
		}
		descDict.Set("FontFile2", core.PdfIndirectReference{ObjectNumber: ffNum})
		objects = append(objects, ffObj)
	}
	objects = append(objects, descObj)

	// CIDFont dict.
	cidNum := nextObjNum()
	cidSysInfo := core.NewDictionary()
	cidSysInfo.Set("Registry", core.PdfString("Adobe"))
	cidSysInfo.Set("Ordering", core.PdfString("Identity"))
	cidSysInfo.Set("Supplement", core.PdfInteger(0))

	cidDict := core.NewDictionary()
	cidDict.Set("Type", core.PdfName("Font"))
	cidDict.Set("Subtype", core.PdfName("CIDFontType2"))
	cidDict.Set("BaseFont", core.PdfName(face.PostScriptName()))
	cidDict.Set("CIDSystemInfo", cidSysInfo)
	cidDict.Set("FontDescriptor", core.PdfIndirectReference{ObjectNumber: descNum})
	cidDict.Set("DW", core.PdfInteger(1000))
	cidDict.Set("W", ef.buildWidthArray())
	cidDict.Set("CIDToGIDMap", core.PdfName("Identity"))

	cidObj := core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: cidNum},
		Object:    cidDict,
	}
	objects = append(objects, cidObj)

	// ToUnicode CMap stream.
	tuNum := nextObjNum()
	tuStream := core.NewStream(ef.ToUnicodeCMap())
	tuObj := core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: tuNum},
		Object:    tuStream,
	}
	objects = append(objects, tuObj)

	// Type0 font dict.
	t0Num := nextObjNum()
	t0Dict := core.NewDictionary()
	t0Dict.Set("Type", core.PdfName("Font"))
	t0Dict.Set("Subtype", core.PdfName("Type0"))
	t0Dict.Set("BaseFont", core.PdfName(face.PostScriptName()))
	t0Dict.Set("Encoding", core.PdfName("Identity-H"))
	t0Dict.Set("DescendantFonts", core.PdfArray{
		core.PdfIndirectReference{ObjectNumber: cidNum},
	})
	t0Dict.Set("ToUnicode", core.PdfIndirectReference{ObjectNumber: tuNum})

	t0Obj := core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: t0Num},
		Object:    t0Dict,
	}
	objects = append(objects, t0Obj)

	return objects
}

func (ef *EmbeddedFont) buildWidthArray() core.PdfArray {
	upem := ef.Face.UnitsPerEm()
	var arr core.PdfArray
	for gid := range ef.usedGlyphs {
		adv := ef.Face.GlyphAdvance(gid)
		w := adv
		if upem != 1000 && upem != 0 {
			w = adv * 1000 / upem
		}
		arr = append(arr, core.PdfInteger(gid), core.PdfArray{core.PdfInteger(w)})
	}
	return arr
}

// ToUnicodeCMap generates a CMap for text extraction from the PDF.
func (ef *EmbeddedFont) ToUnicodeCMap() []byte {
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\n")
	b.WriteString("begincmap\n")
	b.WriteString("/CIDSystemInfo\n<< /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n")
	b.WriteString("/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")

	n := len(ef.usedGlyphs)
	if n > 0 {
		b.WriteString(fmt.Sprintf("%d beginbfchar\n", n))
		for gid, r := range ef.usedGlyphs {
			b.WriteString(fmt.Sprintf("<%04X> <%04X>\n", gid, r))
		}
		b.WriteString("endbfchar\n")
	}

	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
	return []byte(b.String())
}
