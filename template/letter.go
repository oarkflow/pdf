package template

import (
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
)

// LetterData holds all data needed to render a business letter.
type LetterData struct {
	From       Company
	To         Company
	Date       string
	Subject    string
	Salutation string
	Body       []string // paragraphs
	Closing    string
	Signature  string
	Logo       []byte
}

// NewLetterTemplate creates a template configured for business letters.
func NewLetterTemplate() *Template {
	return New("letter").
		SetPageSize(document.Letter).
		SetMargins(document.Margins{Top: 72, Right: 72, Bottom: 72, Left: 72})
}

// RenderLetter renders a business letter from the given data.
func RenderLetter(data LetterData) (*document.Document, error) {
	t := NewLetterTemplate()

	var elements []layout.Element

	// From address block
	elements = append(elements, layout.NewParagraph(data.From.Name))
	if data.From.Address != "" {
		elements = append(elements, layout.NewParagraph(data.From.fullAddress()))
	}
	if data.From.Phone != "" || data.From.Email != "" {
		elements = append(elements, contactParagraph(data.From.Phone, data.From.Email))
	}
	elements = append(elements, layout.NewSpacer(20))

	// Date
	if data.Date != "" {
		elements = append(elements, layout.NewParagraph(data.Date))
		elements = append(elements, layout.NewSpacer(20))
	}

	// To address block
	elements = append(elements, layout.NewParagraph(data.To.Name))
	if data.To.Address != "" {
		elements = append(elements, layout.NewParagraph(data.To.fullAddress()))
	}
	elements = append(elements, layout.NewSpacer(20))

	// Subject
	if data.Subject != "" {
		subRun := layout.TextRun{
			Text:     "Re: " + data.Subject,
			FontName: "Helvetica",
			FontSize: 12,
			Bold:     true,
			Color:    [3]float64{0, 0, 0},
		}
		elements = append(elements, layout.NewStyledParagraph(subRun))
		elements = append(elements, layout.NewSpacer(16))
	}

	// Salutation
	if data.Salutation != "" {
		elements = append(elements, layout.NewParagraph(data.Salutation))
		elements = append(elements, layout.NewSpacer(12))
	}

	// Body paragraphs
	for _, para := range data.Body {
		p := layout.NewParagraph(para)
		p.SpaceAfter = 8
		elements = append(elements, p)
	}
	elements = append(elements, layout.NewSpacer(20))

	// Closing
	if data.Closing != "" {
		elements = append(elements, layout.NewParagraph(data.Closing))
		elements = append(elements, layout.NewSpacer(40))
	}

	// Signature
	if data.Signature != "" {
		elements = append(elements, layout.NewParagraph(data.Signature))
	}

	t.AddSection("letter", elements...)
	return t.Render(nil)
}
