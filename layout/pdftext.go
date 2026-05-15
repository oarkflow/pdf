package layout

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	pdffont "github.com/oarkflow/pdf/font"
)

func isEmbeddedFontFace(face pdffont.Face) bool {
	if face == nil {
		return false
	}
	if _, ok := face.(*pdffont.StandardFont); ok {
		return false
	}
	return !pdffont.IsStandardFont(face.PostScriptName())
}

func fontResourceID(name string, bold, italic bool, face pdffont.Face) string {
	if isEmbeddedFontFace(face) {
		if ps := strings.TrimSpace(face.PostScriptName()); ps != "" {
			return ps
		}
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			return trimmed
		}
		return fmt.Sprintf("custom-font-%p", face)
	}

	suffix := ""
	if bold {
		suffix += "-Bold"
	}
	if italic {
		suffix += "-Oblique"
	}
	return name + suffix
}

func resolveFontEntry(ctx *DrawContext, name string, bold, italic bool, face pdffont.Face) *FontEntry {
	resourceID := fontResourceID(name, bold, italic, face)
	if entry, ok := ctx.Fonts[resourceID]; ok {
		return &entry
	}

	pdfName := fmt.Sprintf("F%d", len(ctx.Fonts)+1)
	entry := FontEntry{
		PDFName: pdfName,
		Name:    name,
		Face:    face,
	}
	if isEmbeddedFontFace(face) {
		entry.Embedded = pdffont.NewEmbeddedFont(face, pdfName)
	}
	ctx.Fonts[resourceID] = entry
	return &entry
}

func storeFontEntry(ctx *DrawContext, name string, bold, italic bool, face pdffont.Face, entry *FontEntry) {
	if entry == nil {
		return
	}
	ctx.Fonts[fontResourceID(name, bold, italic, face)] = *entry
}

// EnsureFontResource registers a font resource for the page and returns the
// PDF resource name that should be used in the content stream.
func EnsureFontResource(ctx *DrawContext, name string, bold, italic bool, face pdffont.Face) string {
	return resolveFontEntry(ctx, name, bold, italic, face).PDFName
}

// PrepareTextOperand registers the font and returns the PDF font resource name
// along with a ready-to-write text operand for the Tj operator.
func PrepareTextOperand(ctx *DrawContext, name string, bold, italic bool, face pdffont.Face, text string) (string, string) {
	entry := resolveFontEntry(ctx, name, bold, italic, face)
	if entry.Embedded != nil {
		encoded := entry.Embedded.EncodedString(text)
		storeFontEntry(ctx, name, bold, italic, face, entry)
		return entry.PDFName, "<" + strings.ToUpper(hex.EncodeToString([]byte(encoded))) + ">"
	}
	return entry.PDFName, "(" + escapePDFText(text) + ")"
}

// PositionedGlyph is a PDF-ready shaped glyph with point-based positioning.
type PositionedGlyph struct {
	Operand  string
	XAdvance float64
	YAdvance float64
	XOffset  float64
	YOffset  float64
}

// PrepareShapedText registers shaped glyphs for an embedded font and returns
// PDF-ready glyph operands. It returns ok=false when shaping is unavailable.
func PrepareShapedText(ctx *DrawContext, name string, bold, italic bool, face pdffont.Face, text string, fontSize float64) (pdfName string, glyphs []PositionedGlyph, ok bool) {
	shaper, canShape := face.(pdffont.Shaper)
	if !canShape || face == nil || text == "" || utf8.RuneCountInString(text) == len(text) {
		return "", nil, false
	}
	shaped, shapedOK := shaper.ShapeText(text)
	if !shapedOK || len(shaped) == 0 {
		return "", nil, false
	}
	entry := resolveFontEntry(ctx, name, bold, italic, face)
	if entry.Embedded == nil {
		return "", nil, false
	}
	upem := face.UnitsPerEm()
	if upem <= 0 {
		upem = 1000
	}
	scale := fontSize / float64(upem)
	glyphs = make([]PositionedGlyph, 0, len(shaped))
	for _, g := range shaped {
		encoded := entry.Embedded.EncodedGlyph(g.GlyphID, g.Cluster)
		glyphs = append(glyphs, PositionedGlyph{
			Operand:  "<" + strings.ToUpper(hex.EncodeToString([]byte(encoded))) + ">",
			XAdvance: float64(g.XAdvance) * scale,
			YAdvance: float64(g.YAdvance) * scale,
			XOffset:  float64(g.XOffset) * scale,
			YOffset:  float64(g.YOffset) * scale,
		})
	}
	storeFontEntry(ctx, name, bold, italic, face, entry)
	return entry.PDFName, glyphs, true
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
