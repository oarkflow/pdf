package html

import (
	"testing"
)

func TestParseCSS_BasicRules(t *testing.T) {
	css := `p { color: red; font-size: 14px; } .big { font-size: 24px; }`
	sheet, err := ParseCSS(css)
	if err != nil {
		t.Fatalf("ParseCSS() error = %v", err)
	}
	if len(sheet.Rules) != 2 {
		t.Errorf("got %d rules, want 2", len(sheet.Rules))
	}
}

func TestParseCSS_Comments(t *testing.T) {
	css := `/* comment */ p { color: blue; } /* another */`
	sheet, err := ParseCSS(css)
	if err != nil {
		t.Fatalf("ParseCSS() error = %v", err)
	}
	if len(sheet.Rules) != 1 {
		t.Errorf("got %d rules, want 1", len(sheet.Rules))
	}
}

func TestParseCSS_MediaRule(t *testing.T) {
	tests := []struct {
		name      string
		css       string
		wantRules int
	}{
		{"print media", `@media print { .x { color: red; } }`, 1},
		{"screen media", `@media screen { .x { color: red; } }`, 1},
		{"all media", `@media all { .x { color: red; } }`, 1},
		{"unknown media", `@media speech { .x { color: red; } }`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheet, _ := ParseCSS(tt.css)
			if len(sheet.Rules) != tt.wantRules {
				t.Errorf("got %d rules, want %d", len(sheet.Rules), tt.wantRules)
			}
		})
	}
}

func TestParseCSS_FontFace(t *testing.T) {
	css := `@font-face { font-family: "MyFont"; src: url(font.woff); font-weight: bold; }`
	sheet, _ := ParseCSS(css)
	if len(sheet.FontFaces) != 1 {
		t.Fatalf("got %d font faces, want 1", len(sheet.FontFaces))
	}
	if sheet.FontFaces[0].Family != "MyFont" {
		t.Errorf("family = %q, want MyFont", sheet.FontFaces[0].Family)
	}
	if sheet.FontFaces[0].Weight != "bold" {
		t.Errorf("weight = %q, want bold", sheet.FontFaces[0].Weight)
	}
}

func TestParseCSS_PageRule(t *testing.T) {
	css := `@page { margin: 1in; }`
	sheet, _ := ParseCSS(css)
	if len(sheet.Pages) != 1 {
		t.Fatalf("got %d page rules, want 1", len(sheet.Pages))
	}
}

func TestParseInlineStyle(t *testing.T) {
	tests := []struct {
		style string
		key   string
		value string
	}{
		{"color: red", "color", "red"},
		{"font-size: 16px; color: blue", "font-size", "16px"},
		{"color: red !important", "color", "red"},
	}
	for _, tt := range tests {
		t.Run(tt.style, func(t *testing.T) {
			props := ParseInlineStyle(tt.style)
			if props[tt.key].Value != tt.value {
				t.Errorf("got %q, want %q", props[tt.key].Value, tt.value)
			}
		})
	}
}

func TestParseInlineStyle_Important(t *testing.T) {
	props := ParseInlineStyle("color: red !important")
	if !props["color"].Priority {
		t.Error("expected !important priority")
	}
}

func TestParseInlineStyle_PreservesComplexValues(t *testing.T) {
	style := `background: linear-gradient(135deg, var(--from) 0%, var(--to) 100%); box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1); content: "a:b;c";`
	props := ParseInlineStyle(style)

	if props["background-image"].Value != "linear-gradient(135deg, var(--from) 0%, var(--to) 100%)" {
		t.Fatalf("background-image = %q", props["background-image"].Value)
	}
	if props["box-shadow"].Value != "0 10px 15px -3px rgba(0, 0, 0, 0.1)" {
		t.Fatalf("box-shadow = %q", props["box-shadow"].Value)
	}
	if props["content"].Value != `"a:b;c"` {
		t.Fatalf("content = %q", props["content"].Value)
	}
}

func TestExpandShorthand_Margin(t *testing.T) {
	tests := []struct {
		value string
		top   string
		right string
	}{
		{"10px", "10px", "10px"},
		{"10px 20px", "10px", "20px"},
		{"10px 20px 30px", "10px", "20px"},
		{"10px 20px 30px 40px", "10px", "20px"},
	}
	for _, tt := range tests {
		props := ParseInlineStyle("margin: " + tt.value)
		if props["margin-top"].Value != tt.top {
			t.Errorf("margin: %s -> margin-top=%q, want %q", tt.value, props["margin-top"].Value, tt.top)
		}
		if props["margin-right"].Value != tt.right {
			t.Errorf("margin: %s -> margin-right=%q, want %q", tt.value, props["margin-right"].Value, tt.right)
		}
	}
}

func TestExpandShorthand_Border(t *testing.T) {
	props := ParseInlineStyle("border: 1px solid red")
	if props["border-top-width"].Value != "1px" {
		t.Errorf("border-top-width = %q, want 1px", props["border-top-width"].Value)
	}
	if props["border-top-style"].Value != "solid" {
		t.Errorf("border-top-style = %q, want solid", props["border-top-style"].Value)
	}
	if props["border-top-color"].Value != "red" {
		t.Errorf("border-top-color = %q, want red", props["border-top-color"].Value)
	}
}

func TestExpandShorthand_Flex(t *testing.T) {
	tests := []struct {
		value  string
		grow   string
		shrink string
		basis  string
	}{
		{"none", "0", "0", "auto"},
		{"auto", "1", "1", "auto"},
		{"2", "2", "1", "0"},
		{"2 3", "2", "3", "0"},
		{"2 3 100px", "2", "3", "100px"},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			props := ParseInlineStyle("flex: " + tt.value)
			if props["flex-grow"].Value != tt.grow {
				t.Errorf("flex-grow = %q, want %q", props["flex-grow"].Value, tt.grow)
			}
			if props["flex-shrink"].Value != tt.shrink {
				t.Errorf("flex-shrink = %q, want %q", props["flex-shrink"].Value, tt.shrink)
			}
		})
	}
}

func TestSelectorMatching(t *testing.T) {
	root := CreateElement("div")
	root.SetAttribute("id", "main")
	root.SetAttribute("class", "container wide")

	p := CreateElement("p")
	p.SetAttribute("class", "text")
	root.AppendChild(p)

	tests := []struct {
		sel  string
		node *Node
		want bool
	}{
		{"div", root, true},
		{"p", root, false},
		{".container", root, true},
		{".wide", root, true},
		{"#main", root, true},
		{"#other", root, false},
		{".text", p, true},
		{"p.text", p, true},
	}
	for _, tt := range tests {
		t.Run(tt.sel, func(t *testing.T) {
			sels, err := ParseSelector(tt.sel)
			if err != nil || len(sels) == 0 {
				t.Fatalf("ParseSelector(%q) failed", tt.sel)
			}
			got := sels[0].Matches(tt.node)
			if got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		sel  string
		want [4]int
	}{
		{"p", [4]int{0, 0, 0, 1}},
		{".cls", [4]int{0, 0, 1, 0}},
		{"#id", [4]int{0, 1, 0, 0}},
		{"div.cls", [4]int{0, 0, 1, 1}},
		{"div#id.cls", [4]int{0, 1, 1, 1}},
	}
	for _, tt := range tests {
		t.Run(tt.sel, func(t *testing.T) {
			sels, _ := ParseSelector(tt.sel)
			if len(sels) == 0 {
				t.Fatal("no selectors")
			}
			got := Specificity(&sels[0])
			if got != tt.want {
				t.Errorf("Specificity = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareSpecificity(t *testing.T) {
	a := [4]int{0, 1, 0, 0}
	b := [4]int{0, 0, 5, 5}
	if CompareSpecificity(a, b) != 1 {
		t.Error("ID should beat classes")
	}
	if CompareSpecificity(b, a) != -1 {
		t.Error("classes should lose to ID")
	}
	if CompareSpecificity(a, a) != 0 {
		t.Error("equal should return 0")
	}
}
