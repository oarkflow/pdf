package forms

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/reader"
)

// FormData maps field names to their string values for form filling.
type FormData map[string]string

// FillForm fills form fields in an existing PDF document and returns a
// flattened PDF with the supplied values stamped into widget rectangles.
func FillForm(pdfData []byte, data FormData) ([]byte, error) {
	if len(pdfData) == 0 {
		return nil, errors.New("forms: PDF data is empty")
	}
	if len(data) == 0 {
		return nil, errors.New("forms: form data is empty")
	}
	r, err := reader.Open(pdfData)
	if err != nil {
		return nil, err
	}
	fields := r.FormFields()
	if len(fields) == 0 {
		return nil, errors.New("forms: no AcroForm widget fields found")
	}
	fieldsByPage := make(map[int][]reader.AnnotationInfo)
	for _, field := range fields {
		if _, ok := data[field.Name]; !ok {
			continue
		}
		fieldsByPage[field.Page-1] = append(fieldsByPage[field.Page-1], field)
	}
	if len(fieldsByPage) == 0 {
		return nil, errors.New("forms: no supplied data matched PDF field names")
	}
	return reader.CopyPages(pdfData, reader.CopyOptions{
		Overlay: func(pageIndex int, page *document.Page) []byte {
			return fillOverlay(page, fieldsByPage[pageIndex], data)
		},
	})
}

// FillFormJSON fills a PDF form from a JSON object.
func FillFormJSON(pdfData []byte, jsonData []byte) ([]byte, error) {
	values := make(map[string]interface{})
	if err := json.Unmarshal(jsonData, &values); err != nil {
		return nil, fmt.Errorf("forms: parsing JSON data: %w", err)
	}
	if values == nil {
		return nil, errors.New("forms: JSON data must be an object")
	}
	data := make(FormData, len(values))
	for key, value := range values {
		data[key] = formValueString(value)
	}
	return FillForm(pdfData, data)
}

func fillOverlay(page *document.Page, fields []reader.AnnotationInfo, data FormData) []byte {
	if len(fields) == 0 {
		return nil
	}
	page.Fonts["FFill"] = 0
	var out strings.Builder
	for _, field := range fields {
		value := data[field.Name]
		x1, y1, x2, y2 := field.Rect[0], field.Rect[1], field.Rect[2], field.Rect[3]
		w := x2 - x1
		h := y2 - y1
		if w <= 0 || h <= 0 {
			continue
		}
		switch field.Field {
		case "Btn":
			if truthyFormValue(value) {
				out.WriteString(fmt.Sprintf("\nq 0 0 0 rg %.3f %.3f %.3f %.3f re f Q\n", x1+2, y1+2, w-4, h-4))
			}
		case "Sig":
			out.WriteString(fmt.Sprintf("\nq 0 G %.3f %.3f m %.3f %.3f l S Q\nBT /FFill 9 Tf %.3f %.3f Td (%s) Tj ET\n",
				x1+2, y1+4, x2-2, y1+4, x1+4, y1+h/2-3, escapeStringForStream(value)))
		default:
			size := h * 0.45
			if size > 12 {
				size = 12
			}
			if size < 7 {
				size = 7
			}
			out.WriteString(fmt.Sprintf("\nBT /FFill %.2f Tf %.3f %.3f Td (%s) Tj ET\n",
				size, x1+3, y1+(h-size)/2, escapeStringForStream(value)))
		}
	}
	return []byte(out.String())
}

func formValueString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	}
}

func truthyFormValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on", "checked", "y":
		return true
	default:
		return false
	}
}
