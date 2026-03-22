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

func TestConvert_FlexLayoutPreservesEmptySpacer(t *testing.T) {
	html := `<html><body><div style="display: flex"><div style="flex: 2"></div><div style="flex: 1; border: 1px solid #000">Totals</div></div></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) != 1 {
		t.Fatalf("got %d top-level elements, want 1", len(result.Elements))
	}
	flex, ok := result.Elements[0].(*FlexContainerElement)
	if !ok {
		t.Fatalf("top-level element = %T, want *FlexContainerElement", result.Elements[0])
	}
	if len(flex.Children) != 2 {
		t.Fatalf("got %d flex children, want 2", len(flex.Children))
	}
	if _, ok := flex.Children[0].Element.(*DivElement); !ok {
		t.Fatalf("spacer element = %T, want *DivElement", flex.Children[0].Element)
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

func TestConvert_StyleTagAppliesBorderRadiusShorthand(t *testing.T) {
	html := `<!DOCTYPE html><html><head><style>.totals-box { border-radius: 0 0 6px 6px; overflow: hidden; background-color: #f8f9fb; }</style></head><body><div class="totals-box">Totals</div></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(result.Elements))
	}
	div, ok := result.Elements[0].(*DivElement)
	if !ok {
		t.Fatalf("element type = %T, want *DivElement", result.Elements[0])
	}
	if div.BoxModel.BorderTopLeftRadius != 0 || div.BoxModel.BorderTopRightRadius != 0 {
		t.Fatalf("top radii = (%v, %v), want (0, 0)", div.BoxModel.BorderTopLeftRadius, div.BoxModel.BorderTopRightRadius)
	}
	if div.BoxModel.BorderBottomRightRadius <= 0 || div.BoxModel.BorderBottomLeftRadius <= 0 {
		t.Fatalf("expected bottom radii > 0, got (%v, %v)", div.BoxModel.BorderBottomRightRadius, div.BoxModel.BorderBottomLeftRadius)
	}
	if div.Style == nil || div.Style.Overflow != "hidden" {
		t.Fatalf("overflow = %q, want hidden", div.Style.Overflow)
	}
}

func TestConvert_StyleTagAppliesBackgroundLonghands(t *testing.T) {
	html := `<!DOCTYPE html><html><head><style>.hero { background-image: linear-gradient(90deg, #000, #fff); background-position: center top; background-size: 50% 25%; background-repeat: repeat-x; }</style></head><body><div class="hero">Hero</div></body></html>`
	result, err := Convert(html, Options{})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	div, ok := result.Elements[0].(*DivElement)
	if !ok {
		t.Fatalf("element type = %T, want *DivElement", result.Elements[0])
	}
	if div.BoxModel.BackgroundPosition != "center top" {
		t.Fatalf("background-position = %q", div.BoxModel.BackgroundPosition)
	}
	if div.BoxModel.BackgroundSize != "50% 25%" {
		t.Fatalf("background-size = %q", div.BoxModel.BackgroundSize)
	}
	if div.BoxModel.BackgroundRepeat != "repeat-x" {
		t.Fatalf("background-repeat = %q", div.BoxModel.BackgroundRepeat)
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

func TestComputeBoxModel_SuppressesHiddenBorders(t *testing.T) {
	style := NewDefaultStyle()
	style.Apply(ParseInlineStyle("border: 1px solid #000; border-top: none"), nil, 12)

	c := &converter{rootFontSize: 12}
	box := c.computeBoxModel(style)

	if box.BorderTopWidth != 0 {
		t.Fatalf("BorderTopWidth = %v, want 0", box.BorderTopWidth)
	}
	if box.BorderRightWidth == 0 || box.BorderBottomWidth == 0 || box.BorderLeftWidth == 0 {
		t.Fatalf("expected remaining borders to stay visible, got right=%v bottom=%v left=%v", box.BorderRightWidth, box.BorderBottomWidth, box.BorderLeftWidth)
	}
}

func TestComputeBoxModel_PreservesVisibleBorderColorsWhenTopIsHidden(t *testing.T) {
	style := NewDefaultStyle()
	style.Apply(ParseInlineStyle("border: 1px solid #eaedf1; border-top: none"), nil, 12)

	c := &converter{rootFontSize: 12}
	box := c.computeBoxModel(style)

	want := [3]float64{0.9176470588235294, 0.9294117647058824, 0.9450980392156862}
	if box.BorderRightColor != want || box.BorderBottomColor != want || box.BorderLeftColor != want {
		t.Fatalf("visible border colors = right:%v bottom:%v left:%v, want %v", box.BorderRightColor, box.BorderBottomColor, box.BorderLeftColor, want)
	}
	if box.BorderColor != want {
		t.Fatalf("fallback border color = %v, want %v", box.BorderColor, want)
	}
}

func TestComputeBoxModel_PreservesCornerRadii(t *testing.T) {
	style := NewDefaultStyle()
	style.Apply(ParseInlineStyle("border-radius: 0 0 6px 6px"), nil, 12)

	c := &converter{rootFontSize: 12}
	box := c.computeBoxModel(style)

	if box.BorderTopLeftRadius != 0 || box.BorderTopRightRadius != 0 {
		t.Fatalf("top radii = (%v, %v), want (0, 0)", box.BorderTopLeftRadius, box.BorderTopRightRadius)
	}
	if box.BorderBottomRightRadius <= 0 || box.BorderBottomLeftRadius <= 0 {
		t.Fatalf("expected bottom radii > 0, got (%v, %v)", box.BorderBottomRightRadius, box.BorderBottomLeftRadius)
	}
}
