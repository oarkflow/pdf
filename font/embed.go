package font

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/oarkflow/pdf/core"
)

// EmbeddedFont tracks glyph usage and produces PDF font objects.
type EmbeddedFont struct {
	Face       Face
	PDFName    string
	usedGlyphs map[uint16]glyphUse
	usedRunes  map[rune]uint16
	usedShaped map[shapedGlyphKey]uint16
	nextCID    uint16
}

type glyphUse struct {
	GID     uint16
	Unicode string
}

type shapedGlyphKey struct {
	GID     uint16
	Unicode string
}

// NewEmbeddedFont creates a new EmbeddedFont wrapper.
func NewEmbeddedFont(face Face, pdfName string) *EmbeddedFont {
	return &EmbeddedFont{
		Face:       face,
		PDFName:    pdfName,
		usedGlyphs: make(map[uint16]glyphUse),
		usedRunes:  make(map[rune]uint16),
		usedShaped: make(map[shapedGlyphKey]uint16),
		nextCID:    1,
	}
}

// AddRune records a rune as used and returns its glyph ID.
func (ef *EmbeddedFont) AddRune(r rune) uint16 {
	if cid, ok := ef.usedRunes[r]; ok {
		return cid
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
	cid := ef.addGlyph(gid, string(r))
	ef.usedRunes[r] = cid
	return cid
}

// AddGlyph records a shaped glyph ID and maps it to the supplied Unicode
// cluster for ToUnicode extraction.
func (ef *EmbeddedFont) AddGlyph(gid uint16, unicode string) uint16 {
	key := shapedGlyphKey{GID: gid, Unicode: unicode}
	if cid, ok := ef.usedShaped[key]; ok {
		return cid
	}
	cid := ef.addGlyph(gid, unicode)
	ef.usedShaped[key] = cid
	return cid
}

func (ef *EmbeddedFont) addGlyph(gid uint16, unicode string) uint16 {
	cid := ef.nextCID
	ef.nextCID++
	ef.usedGlyphs[cid] = glyphUse{GID: gid, Unicode: unicode}
	return cid
}

// glyphSubstitutions maps characters to fallback alternatives when the
// primary glyph is missing from a font.
var glyphSubstitutions = map[rune]rune{
	'\u00A0': ' ', // NBSP → regular space
	'\u2002': ' ', // EN SPACE → regular space
	'\u2003': ' ', // EM SPACE → regular space
	'\u2009': ' ', // THIN SPACE → regular space
	'\u200A': ' ', // HAIR SPACE → regular space
	'\u202F': ' ', // NARROW NO-BREAK SPACE → regular space
}

// AddString records all runes in s as used.
func (ef *EmbeddedFont) AddString(s string) {
	for _, r := range s {
		ef.AddRune(r)
	}
}

// UsedGlyphs returns the glyph-to-rune mapping.
func (ef *EmbeddedFont) UsedGlyphs() map[uint16]glyphUse {
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
		cid := ef.AddRune(r)
		buf = append(buf, byte(cid>>8), byte(cid))
	}
	return string(buf)
}

// EncodedGlyph returns the encoded CID bytes for a shaped glyph.
func (ef *EmbeddedFont) EncodedGlyph(gid uint16, unicode string) string {
	cid := ef.AddGlyph(gid, unicode)
	return string([]byte{byte(cid >> 8), byte(cid)})
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

	// CID-to-GID map stream.
	cidMapNum := nextObjNum()
	cidMapStream := core.NewStream(ef.CIDToGIDMap())
	cidMapObj := core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: cidMapNum},
		Object:    cidMapStream,
	}
	objects = append(objects, cidMapObj)

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
	cidDict.Set("CIDToGIDMap", core.PdfIndirectReference{ObjectNumber: cidMapNum})

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
		adv := ef.Face.GlyphAdvance(ef.usedGlyphs[gid].GID)
		w := adv
		if upem != 1000 && upem != 0 {
			w = adv * 1000 / upem
		}
		arr = append(arr, core.PdfInteger(gid), core.PdfArray{core.PdfInteger(w)})
	}
	return arr
}

// CIDToGIDMap builds the binary map used by CIDFontType2 fonts.
func (ef *EmbeddedFont) CIDToGIDMap() []byte {
	maxCID := 0
	for cid := range ef.usedGlyphs {
		if int(cid) > maxCID {
			maxCID = int(cid)
		}
	}
	data := make([]byte, (maxCID+1)*2)
	for cid, use := range ef.usedGlyphs {
		binary.BigEndian.PutUint16(data[int(cid)*2:], use.GID)
	}
	return data
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
		for cid, use := range ef.usedGlyphs {
			b.WriteString(fmt.Sprintf("<%04X> <%s>\n", cid, utf16Hex(use.Unicode)))
		}
		b.WriteString("endbfchar\n")
	}

	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
	return []byte(b.String())
}

func utf16Hex(s string) string {
	var b strings.Builder
	for _, u := range utf16.Encode([]rune(s)) {
		b.WriteString(fmt.Sprintf("%04X", u))
	}
	return b.String()
}
