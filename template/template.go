package template

import (
	"bytes"
	"fmt"
	"strings"
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

// Render renders the template into a document. If data is non-nil,
// all {{...}} placeholders in text elements are resolved using Go's
// text/template syntax.
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

	// Resolve {{...}} placeholders in text content when data is provided
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

// resolveString replaces {{key}} placeholders in s with values from data.
// Supports map[string]string, map[string]interface{}, and falls back to
// Go text/template for struct types. Placeholders use simple {{key}} syntax.
func resolveString(s string, data interface{}) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}

	// Try flat map replacement first
	switch m := data.(type) {
	case map[string]string:
		return ReplacePlaceholders(s, func(key string) (string, bool) {
			v, ok := m[key]
			return v, ok
		}), nil
	case map[string]interface{}:
		return ReplacePlaceholders(s, func(key string) (string, bool) {
			v, ok := m[key]
			if !ok {
				return "", false
			}
			return fmt.Sprintf("%v", v), true
		}), nil
	default:
		// Fall back to Go text/template for struct types (uses {{.Field}} syntax)
		t, err := template.New("").Parse(s)
		if err != nil {
			return s, err
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return s, err
		}
		return buf.String(), nil
	}
}

// ReplacePlaceholders replaces all {{key}} occurrences in s using the lookup function.
// Whitespace inside braces is trimmed, so {{ key }} and {{key}} both work.
func ReplacePlaceholders(s string, lookup func(string) (string, bool)) string {
	var result strings.Builder
	for {
		start := strings.Index(s, "{{")
		if start == -1 {
			result.WriteString(s)
			break
		}
		end := strings.Index(s[start:], "}}")
		if end == -1 {
			result.WriteString(s)
			break
		}
		end += start

		result.WriteString(s[:start])
		key := strings.TrimSpace(s[start+2 : end])
		if val, ok := lookup(key); ok {
			result.WriteString(val)
		} else {
			// Keep unresolved placeholder as-is
			result.WriteString(s[start : end+2])
		}
		s = s[end+2:]
	}
	return result.String()
}

// resolvePlaceholders walks a layout element and resolves all {{...}}
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

// ReplaceMap replaces all {{key}} placeholders in s with values from the map.
func ReplaceMap(s string, data map[string]string) string {
	return ReplacePlaceholders(s, func(key string) (string, bool) {
		v, ok := data[key]
		return v, ok
	})
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
