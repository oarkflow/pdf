package template

import (
	"bytes"
	"text/template"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
)

// Template is a reusable PDF template with sections and page configuration.
type Template struct {
	name     string
	sections []Section
	pageSize document.PageSize
	margins  document.Margins
}

// Section groups named elements within a template.
type Section struct {
	Name     string
	Elements []layout.Element
}

// New creates a new template with the given name and default A4/margins.
func New(name string) *Template {
	return &Template{
		name:     name,
		pageSize: document.A4,
		margins:  document.DefaultMargins(),
	}
}

// SetPageSize sets the page size for the template.
func (t *Template) SetPageSize(ps document.PageSize) *Template {
	t.pageSize = ps
	return t
}

// SetMargins sets the page margins for the template.
func (t *Template) SetMargins(m document.Margins) *Template {
	t.margins = m
	return t
}

// AddSection adds a named section with elements.
func (t *Template) AddSection(name string, elements ...layout.Element) *Template {
	t.sections = append(t.sections, Section{Name: name, Elements: elements})
	return t
}

// Render renders the template into a document.
func (t *Template) Render(data interface{}) (*document.Document, error) {
	doc := document.NewDocument(t.pageSize)
	doc.SetMargins(t.margins)

	var allElements []layout.Element
	for _, sec := range t.sections {
		allElements = append(allElements, sec.Elements...)
	}

	pages := layout.RenderPages(
		allElements,
		t.pageSize.Width, t.pageSize.Height,
		t.margins.Top, t.margins.Right, t.margins.Bottom, t.margins.Left,
	)

	for _, pr := range pages {
		p := document.NewPage(document.PageSize{Width: pr.Width, Height: pr.Height})
		p.Contents = pr.Content
		for name, fe := range pr.Fonts {
			p.Fonts[name] = fe.ObjectNum
		}
		for name, ie := range pr.Images {
			p.Images[name] = ie
		}
		doc.AddPage(p)
	}

	return doc, nil
}

// FromString processes a Go text/template string with data and returns the result.
func FromString(tmpl string, data interface{}) (string, error) {
	t, err := template.New("pdf").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Execute renders the template and saves to the given path.
func (t *Template) Execute(data interface{}, path string) error {
	doc, err := t.Render(data)
	if err != nil {
		return err
	}
	return doc.Save(path)
}
