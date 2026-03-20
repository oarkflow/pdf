package forms

import (
	"fmt"

	"github.com/oarkflow/pdf/core"
)

// BuildAppearance generates the /AP normal appearance stream for a field.
func BuildAppearance(field *Field) *core.PdfStream {
	var content string

	w := field.Rect[2] - field.Rect[0]
	h := field.Rect[3] - field.Rect[1]
	bbox := fmt.Sprintf("0 0 %.2f %.2f", w, h)

	fontSize := field.FontSize
	if fontSize <= 0 {
		fontSize = 12
	}
	fontName := field.FontName
	if fontName == "" {
		fontName = "Helv"
	}

	switch field.Type {
	case FieldText:
		// Simple text appearance: render the current value.
		escaped := escapeStringForStream(field.Value)
		content = fmt.Sprintf("BT /%s %.1f Tf 2 %.1f Td (%s) Tj ET",
			fontName, fontSize, (h-fontSize)/2, escaped)

	case FieldCheckbox:
		if field.Value == "Yes" {
			// Draw a checkmark.
			content = fmt.Sprintf("q 0 0 %.2f %.2f re W n "+
				"0.2 0.2 %.2f %.2f re f "+
				"BT /ZaDb 0 Tf %.1f 0 0 %.1f 2 2 Tm (4) Tj ET Q",
				w, h, w-0.4, h-0.4, fontSize, fontSize)
		} else {
			// Empty box.
			content = fmt.Sprintf("q 0.5 G 1 1 %.2f %.2f re s Q", w-2, h-2)
		}

	case FieldRadio:
		cx := w / 2
		cy := h / 2
		r := cx
		if cy < r {
			r = cy
		}
		r -= 1
		if field.Value != "" && field.Value != "Off" {
			// Filled circle.
			content = fmt.Sprintf("q %.2f %.2f %.2f 0 360 arc f Q", cx, cy, r)
		} else {
			// Empty circle.
			content = fmt.Sprintf("q 0.5 G %.2f %.2f %.2f 0 360 arc s Q", cx, cy, r)
		}

	case FieldDropdown, FieldComboBox:
		// Draw text value and a small dropdown arrow.
		escaped := escapeStringForStream(field.Value)
		arrowX := w - 10
		content = fmt.Sprintf("BT /%s %.1f Tf 2 %.1f Td (%s) Tj ET "+
			"q 0.5 G %.2f %.2f m %.2f %.2f l %.2f %.2f l f Q",
			fontName, fontSize, (h-fontSize)/2, escaped,
			arrowX, h*0.6, arrowX+8, h*0.6, arrowX+4, h*0.3)

	case FieldSignature:
		// Signature placeholder: line and label.
		content = fmt.Sprintf("q 0.5 G 2 4 m %.2f 4 l S Q "+
			"BT /Helv 8 Tf 2 6 Td (Signature) Tj ET", w-2)
	}

	stream := core.NewStream([]byte(content))
	stream.Dictionary.Set("Type", core.PdfName("XObject"))
	stream.Dictionary.Set("Subtype", core.PdfName("Form"))
	stream.Dictionary.Set("BBox", core.PdfArray{
		core.PdfNumber(0), core.PdfNumber(0),
		core.PdfNumber(w), core.PdfNumber(h),
	})
	_ = bbox // bbox values already set via the array above

	return stream
}

// escapeStringForStream escapes parentheses and backslashes for use inside
// a PDF content stream string literal.
func escapeStringForStream(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			out = append(out, '\\', '(')
		case ')':
			out = append(out, '\\', ')')
		case '\\':
			out = append(out, '\\', '\\')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}
