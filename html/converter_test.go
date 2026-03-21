package html

import (
	"testing"
)

func TestConvert_BasicHTML(t *testing.T) {
	html := `<html><body><p>Hello World</p></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Elements) == 0 {
		t.Error("no elements produced")
	}
}

func TestConvert_DefaultOptions(t *testing.T) {
	result, err := Convert("<p>Test</p>", Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	// Check defaults applied
	if result.Config.Width == 0 {
		t.Error("page width should be set")
	}
	if result.Config.Height == 0 {
		t.Error("page height should be set")
	}
}

func TestConvert_Headings(t *testing.T) {
	html := `<html><body><h1>Title</h1><h2>Subtitle</h2><h3>Section</h3></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) < 3 {
		t.Errorf("got %d elements, want at least 3", len(result.Elements))
	}
}

func TestConvert_Paragraphs(t *testing.T) {
	html := `<html><body><p>First</p><p>Second</p></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) < 2 {
		t.Errorf("got %d elements, want at least 2", len(result.Elements))
	}
}

func TestConvert_List(t *testing.T) {
	html := `<html><body><ul><li>A</li><li>B</li></ul></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) == 0 {
		t.Error("no elements")
	}
}

func TestConvert_Table(t *testing.T) {
	html := `<html><body><table><tr><td>Cell</td></tr></table></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) == 0 {
		t.Error("no elements")
	}
}

func TestConvert_Metadata(t *testing.T) {
	html := `<html><head><title>My Page</title><meta name="author" content="John"></head><body><p>X</p></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if result.Metadata["title"] != "My Page" {
		t.Errorf("title = %q, want My Page", result.Metadata["title"])
	}
	if result.Metadata["author"] != "John" {
		t.Errorf("author = %q, want John", result.Metadata["author"])
	}
}

func TestConvert_FlexLayout(t *testing.T) {
	html := `<html><body><div style="display: flex"><div>A</div><div>B</div></div></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) == 0 {
		t.Error("no elements for flex layout")
	}
}

func TestConvert_GridLayout(t *testing.T) {
	html := `<html><body><div style="display: grid; grid-template-columns: 1fr 1fr"><div>A</div><div>B</div></div></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) == 0 {
		t.Error("no elements for grid layout")
	}
}

func TestConvert_DisplayNone(t *testing.T) {
	html := `<html><body><div style="display: none">Hidden</div><p>Visible</p></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	// The hidden div should be skipped
	if len(result.Elements) < 1 {
		t.Error("expected at least 1 visible element")
	}
}

func TestConvert_UserStylesheet(t *testing.T) {
	html := `<html><body><p>Styled</p></body></html>`
	result, err := Convert(html, Options{
		UserStylesheet: "p { font-size: 24pt; }",
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestResolveVarReferences(t *testing.T) {
	tests := []struct {
		value string
		props map[string]string
		want  string
	}{
		{"var(--x)", map[string]string{"--x": "10px"}, "10px"},
		{"var(--missing, 5px)", map[string]string{}, "5px"},
		{"var(--a) var(--b)", map[string]string{"--a": "1", "--b": "2"}, "1 2"},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := resolveVarReferences(tt.value, tt.props)
			if got != tt.want {
				t.Errorf("resolveVarReferences(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestResolveCalcExpressions(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"calc(10px + 5px)", "15px"},
		{"calc(0.25rem * 4)", "1rem"},
		{"calc(100px - 20px)", "80px"},
		{"no-calc", "no-calc"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveCalcExpressions(tt.input)
			if got != tt.want {
				t.Errorf("resolveCalcExpressions(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseValueUnit(t *testing.T) {
	tests := []struct {
		input string
		val   float64
		unit  string
	}{
		{"10px", 10, "px"},
		{"1.5rem", 1.5, "rem"},
		{"50%", 50, "%"},
		{"42", 42, ""},
		{"0", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, u := parseValueUnit(tt.input)
			if v != tt.val || u != tt.unit {
				t.Errorf("parseValueUnit(%q) = (%v, %q), want (%v, %q)", tt.input, v, u, tt.val, tt.unit)
			}
		})
	}
}

func TestCollapseWhitespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello   world  ", " hello world "},
		{"no\n\nnewlines", "no newlines"},
		{"a\tb", "a b"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := collapseWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("collapseWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
