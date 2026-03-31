package layout

import (
	"encoding/hex"
	"fmt"
	"strings"

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

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
