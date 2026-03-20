package template

import (
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
)

// ReportData holds all data needed to render a report.
type ReportData struct {
	Title    string
	Subtitle string
	Author   string
	Date     string
	Sections []ReportSection
	TOC      bool
	CoverPage bool
	Logo     []byte
}

// ReportSection represents a section of a report.
type ReportSection struct {
	Title   string
	Level   int // 1-3
	Content string
	Items   []string // bullet points
}

// NewReportTemplate creates a template configured for reports.
func NewReportTemplate() *Template {
	return New("report").
		SetPageSize(document.A4).
		SetMargins(document.DefaultMargins())
}

// RenderReport renders a report from the given data.
func RenderReport(data ReportData) (*document.Document, error) {
	t := NewReportTemplate()

	var elements []layout.Element

	// Cover page
	if data.CoverPage {
		elements = append(elements, layout.NewSpacer(200))
		elements = append(elements, centeredHeading(layout.H1, data.Title))
		if data.Subtitle != "" {
			elements = append(elements, layout.NewSpacer(12))
			elements = append(elements, centeredHeading(layout.H2, data.Subtitle))
		}
		elements = append(elements, layout.NewSpacer(40))
		if data.Author != "" {
			elements = append(elements, centeredParagraph("By "+data.Author))
		}
		if data.Date != "" {
			elements = append(elements, layout.NewSpacer(8))
			elements = append(elements, centeredParagraph(data.Date))
		}
		elements = append(elements, layout.NewPageBreak())
	}

	// Table of contents
	if data.TOC && len(data.Sections) > 0 {
		elements = append(elements, layout.NewHeading(layout.H2, "Table of Contents"))
		elements = append(elements, layout.NewSpacer(12))
		for _, sec := range data.Sections {
			indent := float64((sec.Level - 1) * 20)
			p := layout.NewParagraph(sec.Title)
			p.Indent = indent
			elements = append(elements, p)
		}
		elements = append(elements, layout.NewPageBreak())
	}

	// Sections
	for _, sec := range data.Sections {
		level := layout.HeadingLevel(sec.Level)
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		elements = append(elements, layout.NewHeading(level, sec.Title))
		elements = append(elements, layout.NewSpacer(4))

		if sec.Content != "" {
			elements = append(elements, layout.NewParagraph(sec.Content))
			elements = append(elements, layout.NewSpacer(4))
		}

		if len(sec.Items) > 0 {
			elements = append(elements, layout.NewList(layout.UnorderedList, sec.Items...))
			elements = append(elements, layout.NewSpacer(4))
		}

		elements = append(elements, layout.NewSpacer(8))
	}

	t.AddSection("report", elements...)
	return t.Render(nil)
}

func centeredHeading(level layout.HeadingLevel, text string) *layout.Heading {
	h := layout.NewHeading(level, text)
	h.Align = layout.AlignCenter
	return h
}

func centeredParagraph(text string) *layout.Paragraph {
	p := layout.NewParagraph(text)
	p.Align = layout.AlignCenter
	return p
}
