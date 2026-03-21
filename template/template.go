package template

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/oarkflow/fasttpl"
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

// Render renders the template into a document. If data is non-nil,
// all {{ ... }} placeholders in text elements are resolved using fasttpl,
// which supports conditions ({{ if }}), ranges ({{ range }}), filters, etc.
func (t *Template) Render(data interface{}) (*document.Document, error) {
	doc, err := document.NewDocument(t.pageSize)
	if err != nil {
		return nil, err
	}
	doc.SetMargins(t.margins)

	var allElements []layout.Element
	for _, sec := range t.sections {
		allElements = append(allElements, sec.Elements...)
	}

	// Resolve {{ ... }} placeholders in text content when data is provided
	if data != nil {
		for i, el := range allElements {
			resolved, err := resolvePlaceholders(el, data)
			if err != nil {
				return nil, err
			}
			allElements[i] = resolved
		}
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

// resolveString replaces {{ ... }} placeholders in s with values from data
// using fasttpl. Supports conditions, ranges, filters, nested keys, etc.
func resolveString(s string, data interface{}) (string, error) {
	return renderWithFasttpl(s, toMap(data))
}

// renderWithFasttpl compiles and renders a template string using fasttpl.
// Returns the original string unchanged if it contains no {{ delimiters.
func renderWithFasttpl(s string, data map[string]any) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}
	tpl, err := fasttpl.Compile(s)
	if err != nil {
		return s, fmt.Errorf("fasttpl compile: %w", err)
	}
	result, err := tpl.RenderString(data)
	if err != nil {
		return s, fmt.Errorf("fasttpl render: %w", err)
	}
	return result, nil
}

// toMap converts data to map[string]any for fasttpl.
func toMap(data interface{}) map[string]any {
	switch m := data.(type) {
	case map[string]any:
		return m
	case map[string]string:
		result := make(map[string]any, len(m))
		for k, v := range m {
			result[k] = v
		}
		return result
	default:
		return map[string]any{"data": data}
	}
}

// RenderHTML compiles an HTML template string with fasttpl and renders it
// with the given data. Supports {{ if }}, {{ range }}, {{ include }}, filters, etc.
func RenderHTML(htmlTemplate string, data map[string]any, opts ...fasttpl.Option) (string, error) {
	tpl, err := fasttpl.Compile(htmlTemplate, opts...)
	if err != nil {
		return "", fmt.Errorf("fasttpl compile: %w", err)
	}
	return tpl.RenderString(data)
}

// RenderHTMLFile compiles an HTML template from a file with fasttpl and renders it
// with the given data. Supports {{ if }}, {{ range }}, {{ include }}, filters, etc.
func RenderHTMLFile(path string, data map[string]any) (string, error) {
	// Read the file, apply autoRaw, then compile
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading template file: %w", err)
	}
	return RenderHTML(string(content), data)
}

// RenderHTMLTo compiles an HTML template string and writes the result to w.
func RenderHTMLTo(w io.Writer, htmlTemplate string, data map[string]any, opts ...fasttpl.Option) error {
	tpl, err := fasttpl.Compile(htmlTemplate, opts...)
	if err != nil {
		return fmt.Errorf("fasttpl compile: %w", err)
	}
	return tpl.Render(w, data)
}

// resolvePlaceholders walks a layout element and resolves all {{ ... }}
// placeholders in its text content using the provided data.
func resolvePlaceholders(el layout.Element, data interface{}) (layout.Element, error) {
	switch e := el.(type) {
	case *layout.Paragraph:
		cp := *e
		cp.Runs = make([]layout.TextRun, len(e.Runs))
		copy(cp.Runs, e.Runs)
		for i, run := range cp.Runs {
			resolved, err := resolveString(run.Text, data)
			if err != nil {
				return nil, err
			}
			cp.Runs[i].Text = resolved
		}
		return &cp, nil

	case *layout.Heading:
		cp := *e
		resolved, err := resolveString(e.Text, data)
		if err != nil {
			return nil, err
		}
		cp.Text = resolved
		return &cp, nil

	case *layout.Div:
		cp := *e
		cp.Children = make([]layout.Element, len(e.Children))
		for i, child := range e.Children {
			resolved, err := resolvePlaceholders(child, data)
			if err != nil {
				return nil, err
			}
			cp.Children[i] = resolved
		}
		return &cp, nil

	case *layout.FlexContainer:
		cp := *e
		cp.Children = make([]layout.FlexChild, len(e.Children))
		for i, fc := range e.Children {
			resolved, err := resolvePlaceholders(fc.Element, data)
			if err != nil {
				return nil, err
			}
			cp.Children[i] = fc
			cp.Children[i].Element = resolved
		}
		return &cp, nil

	case *layout.List:
		cp := *e
		cp.Items = make([]layout.ListItem, len(e.Items))
		for i, item := range e.Items {
			cp.Items[i] = item
			if item.Content != nil {
				resolved, err := resolvePlaceholders(item.Content, data)
				if err != nil {
					return nil, err
				}
				cp.Items[i].Content = resolved
			}
			if item.Children != nil {
				resolvedList, err := resolvePlaceholders(item.Children, data)
				if err != nil {
					return nil, err
				}
				cp.Items[i].Children = resolvedList.(*layout.List)
			}
		}
		return &cp, nil

	case *layout.Table:
		cp := *e
		cp.Rows = make([]layout.TableRow, len(e.Rows))
		for i, row := range e.Rows {
			cp.Rows[i].Cells = make([]layout.TableCell, len(row.Cells))
			for j, cell := range row.Cells {
				cp.Rows[i].Cells[j] = cell
				if cell.Content != nil {
					resolved, err := resolvePlaceholders(cell.Content, data)
					if err != nil {
						return nil, err
					}
					cp.Rows[i].Cells[j].Content = resolved
				}
			}
		}
		return &cp, nil
	}
	return el, nil
}

// ReplaceMap renders all {{ ... }} expressions in s using fasttpl with the
// provided map data. Supports conditions, ranges, filters, nested keys, etc.
func ReplaceMap(s string, data map[string]string) string {
	m := make(map[string]any, len(data))
	for k, v := range data {
		m[k] = v
	}
	result, err := renderWithFasttpl(s, m)
	if err != nil {
		return s // return original on error for backward compat
	}
	return result
}

// ReplaceMapAny renders all {{ ... }} expressions in s using fasttpl with the
// provided map data. Supports conditions, ranges, filters, nested keys, etc.
func ReplaceMapAny(s string, data map[string]any) string {
	result, err := renderWithFasttpl(s, data)
	if err != nil {
		return s
	}
	return result
}

// FromString processes a fasttpl template string with data and returns the result.
// Supports {{ if }}, {{ range }}, filters, nested keys, etc.
func FromString(tmpl string, data interface{}) (string, error) {
	return resolveString(tmpl, data)
}

// Execute renders the template and saves to the given path.
func (t *Template) Execute(data interface{}, path string) error {
	doc, err := t.Render(data)
	if err != nil {
		return err
	}
	return doc.Save(path)
}
