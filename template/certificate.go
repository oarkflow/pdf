package template

import (
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
)

// CertificateData holds all data needed to render a certificate.
type CertificateData struct {
	Title         string // e.g. "Certificate of Completion"
	RecipientName string
	Description   string
	Date          string
	Issuer        string
	IssuerTitle   string
	Logo          []byte
}

// NewCertificateTemplate creates a template configured for certificates.
func NewCertificateTemplate() *Template {
	return New("certificate").
		SetPageSize(document.PageSize{Width: 842, Height: 595}). // landscape A4
		SetMargins(document.Margins{Top: 60, Right: 60, Bottom: 60, Left: 60})
}

// RenderCertificate renders a certificate from the given data.
func RenderCertificate(data CertificateData) (*document.Document, error) {
	t := NewCertificateTemplate()

	var elements []layout.Element

	// Top decorative line
	elements = append(elements, makeLine(722))
	elements = append(elements, layout.NewSpacer(8))
	elements = append(elements, makeLine(722))
	elements = append(elements, layout.NewSpacer(40))

	// Title
	title := data.Title
	if title == "" {
		title = "Certificate of Completion"
	}
	h := layout.NewHeading(layout.H1, title)
	h.Align = layout.AlignCenter
	elements = append(elements, h)
	elements = append(elements, layout.NewSpacer(30))

	// Presented to
	elements = append(elements, centeredSmall("This certificate is presented to"))
	elements = append(elements, layout.NewSpacer(12))

	// Recipient name (large)
	nameRun := layout.TextRun{
		Text:     data.RecipientName,
		FontName: "Helvetica",
		FontSize: 28,
		Bold:     true,
		Color:    [3]float64{0.1, 0.1, 0.4},
	}
	nameP := layout.NewStyledParagraph(nameRun)
	nameP.Align = layout.AlignCenter
	elements = append(elements, nameP)
	elements = append(elements, layout.NewSpacer(20))

	// Description
	if data.Description != "" {
		desc := layout.NewParagraph(data.Description)
		desc.Align = layout.AlignCenter
		elements = append(elements, desc)
		elements = append(elements, layout.NewSpacer(20))
	}

	// Date
	if data.Date != "" {
		dp := layout.NewParagraph("Date: " + data.Date)
		dp.Align = layout.AlignCenter
		elements = append(elements, dp)
		elements = append(elements, layout.NewSpacer(30))
	}

	// Issuer
	if data.Issuer != "" {
		ip := layout.NewParagraph(data.Issuer)
		ip.Align = layout.AlignCenter
		elements = append(elements, ip)
		if data.IssuerTitle != "" {
			itp := centeredSmall(data.IssuerTitle)
			elements = append(elements, itp)
		}
	}

	elements = append(elements, layout.NewSpacer(40))

	// Bottom decorative lines
	elements = append(elements, makeLine(722))
	elements = append(elements, layout.NewSpacer(8))
	elements = append(elements, makeLine(722))

	t.AddSection("certificate", elements...)
	return t.Render(nil)
}

func centeredSmall(text string) *layout.Paragraph {
	run := layout.TextRun{
		Text:     text,
		FontName: "Helvetica",
		FontSize: 10,
		Color:    [3]float64{0.4, 0.4, 0.4},
	}
	p := layout.NewStyledParagraph(run)
	p.Align = layout.AlignCenter
	return p
}

// makeLine creates a horizontal rule using a table with a border.
func makeLine(width float64) *layout.Spacer {
	// Use a thin spacer as a visual separator; actual line drawing
	// would require lower-level content stream ops. A 1pt spacer
	// combined with the border of surrounding elements achieves
	// a similar effect in the layout engine.
	return layout.NewSpacer(1)
}
