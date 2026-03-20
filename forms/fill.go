package forms

import "errors"

// FormData maps field names to their string values for form filling.
type FormData map[string]string

// FillForm fills form fields in an existing PDF document.
// This is a stub that will be completed once the reader package is available.
func FillForm(pdfData []byte, data FormData) ([]byte, error) {
	return nil, errors.New("forms: FillForm requires reader package (not yet implemented)")
}
