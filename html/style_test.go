package html

import (
	"math"
	"testing"
)

func TestNewDefaultStyle(t *testing.T) {
	s := NewDefaultStyle()
	if s.FontFamily != "Helvetica" {
		t.Errorf("FontFamily = %q, want Helvetica", s.FontFamily)
	}
	if s.FontSize != 12 {
		t.Errorf("FontSize = %v, want 12", s.FontSize)
	}
	if s.FontWeight != 400 {
		t.Errorf("FontWeight = %v, want 400", s.FontWeight)
	}
	if s.Opacity != 1 {
		t.Errorf("Opacity = %v, want 1", s.Opacity)
	}
}

func TestInheritStyle(t *testing.T) {
	parent := NewDefaultStyle()
	parent.FontFamily = "Times"
	parent.FontSize = 16
	parent.Color = [3]float64{1, 0, 0}
	parent.CustomProperties = map[string]string{"--x": "10px"}

	child := InheritStyle(parent)
	if child.FontFamily != "Times" {
		t.Errorf("FontFamily = %q, want Times", child.FontFamily)
	}
	if child.FontSize != 16 {
		t.Errorf("FontSize = %v, want 16", child.FontSize)
	}
	if child.Color != [3]float64{1, 0, 0} {
		t.Errorf("Color not inherited")
	}
	if child.CustomProperties["--x"] != "10px" {
		t.Error("custom properties not inherited")
	}
	// Non-inherited properties should be defaults
	if child.Display != "block" {
		t.Errorf("Display = %q, want block (default)", child.Display)
	}
}

func TestInheritStyle_NilParent(t *testing.T) {
	s := InheritStyle(nil)
	if s.FontFamily != "Helvetica" {
		t.Errorf("FontFamily = %q, want Helvetica", s.FontFamily)
	}
}

func TestCSSLengthToPoints(t *testing.T) {
	tests := []struct {
		length CSSLength
		parent float64
		root   float64
		want   float64
	}{
		{CSSLength{10, "px"}, 12, 12, 7.5},
		{CSSLength{12, "pt"}, 12, 12, 12},
		{CSSLength{2, "em"}, 12, 12, 24},
		{CSSLength{2, "rem"}, 12, 16, 32},
		{CSSLength{50, "%"}, 200, 12, 100},
		{CSSLength{1, "in"}, 12, 12, 72},
		{CSSLength{0, "auto"}, 12, 12, 0},
		{CSSLength{0, "px"}, 12, 12, 0},
		{CSSLength{1, "cm"}, 12, 12, 28.3465},
	}
	for _, tt := range tests {
		t.Run(tt.length.Unit, func(t *testing.T) {
			got := tt.length.ToPoints(tt.parent, tt.root)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("ToPoints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCSSLength_IsAuto(t *testing.T) {
	if !(CSSLength{0, "auto"}).IsAuto() {
		t.Error("auto should be auto")
	}
	if (CSSLength{10, "px"}).IsAuto() {
		t.Error("px should not be auto")
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		input string
		ok    bool
		r     float64
	}{
		{"red", true, 1},
		{"#ff0000", true, 1},
		{"#f00", true, 1},
		{"rgb(255, 0, 0)", true, 1},
		{"rgb(100%, 0%, 0%)", true, 1},
		{"black", true, 0},
		{"transparent", false, 0},
		{"", false, 0},
		{"currentcolor", false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c, ok := parseColor(tt.input)
			if ok != tt.ok {
				t.Errorf("parseColor(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok && math.Abs(c[0]-tt.r) > 0.01 {
				t.Errorf("red = %v, want %v", c[0], tt.r)
			}
		})
	}
}

func TestParseFontWeight(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"normal", 400},
		{"bold", 700},
		{"lighter", 300},
		{"600", 600},
		{"invalid", 400},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseFontWeight(tt.input); got != tt.want {
				t.Errorf("parseFontWeight(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseFontSize(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"medium", 12},
		{"large", 13.5},
		{"x-large", 18},
		{"smaller", 12 * 0.833},
		{"larger", 12 * 1.2},
		{"16px", 12}, // 16 * 0.75
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseFontSize(tt.input, 12, 12)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("parseFontSize(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestApply_BasicProperties(t *testing.T) {
	s := NewDefaultStyle()
	props := map[string]CSSValue{
		"font-family": {Value: "Courier"},
		"font-size":   {Value: "16pt"},
		"font-weight": {Value: "bold"},
		"color":       {Value: "#ff0000"},
		"display":     {Value: "flex"},
		"opacity":     {Value: "0.5"},
	}
	s.Apply(props, nil, 12)
	if s.FontFamily != "Courier" {
		t.Errorf("FontFamily = %q", s.FontFamily)
	}
	if s.FontSize != 16 {
		t.Errorf("FontSize = %v", s.FontSize)
	}
	if s.FontWeight != 700 {
		t.Errorf("FontWeight = %v", s.FontWeight)
	}
	if s.Display != "flex" {
		t.Errorf("Display = %q", s.Display)
	}
	if s.Opacity != 0.5 {
		t.Errorf("Opacity = %v", s.Opacity)
	}
}

func TestApply_CustomProperties(t *testing.T) {
	s := NewDefaultStyle()
	props := map[string]CSSValue{
		"--primary": {Value: "#0000ff"},
		"color":     {Value: "var(--primary)"},
	}
	s.Apply(props, nil, 12)
	if s.Color != [3]float64{0, 0, 1} {
		t.Errorf("Color = %v, want blue", s.Color)
	}
}

func TestApply_VarFallback(t *testing.T) {
	s := NewDefaultStyle()
	props := map[string]CSSValue{
		"color": {Value: "var(--missing, red)"},
	}
	s.Apply(props, nil, 12)
	if s.Color != [3]float64{1, 0, 0} {
		t.Errorf("Color = %v, want red (fallback)", s.Color)
	}
}

func TestResolveCalc(t *testing.T) {
	s := NewDefaultStyle()
	props := map[string]CSSValue{
		"width": {Value: "calc(100px + 50px)"},
	}
	s.Apply(props, nil, 12)
	if s.Width.Value != 150 || s.Width.Unit != "px" {
		t.Errorf("Width = %v %s, want 150px", s.Width.Value, s.Width.Unit)
	}
}

func TestParseLength(t *testing.T) {
	tests := []struct {
		input string
		value float64
		unit  string
	}{
		{"10px", 10, "px"},
		{"1.5em", 1.5, "em"},
		{"50%", 50, "%"},
		{"auto", 0, "auto"},
		{"0", 0, "px"},
		{"none", 0, ""},
		{"12pt", 12, "pt"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := parseLength(tt.input)
			if l.Value != tt.value || l.Unit != tt.unit {
				t.Errorf("parseLength(%q) = {%v, %q}, want {%v, %q}", tt.input, l.Value, l.Unit, tt.value, tt.unit)
			}
		})
	}
}
