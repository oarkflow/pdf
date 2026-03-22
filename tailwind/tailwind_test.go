package tailwind_test

import (
	"fmt"
	"testing"

	"github.com/oarkflow/pdf/tailwind"
)

func TestParser_Display(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class    string
		expected map[string]any
	}{
		{"block", map[string]any{"display": "block"}},
		{"inline-block", map[string]any{"display": "inline-block"}},
		{"flex", map[string]any{"display": "flex"}},
		{"grid", map[string]any{"display": "grid"}},
		{"hidden", map[string]any{"display": "none"}},
		{"inline-flex", map[string]any{"display": "inline-flex"}},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		for k, v := range tt.expected {
			if got[k] != v {
				t.Errorf("class=%q: %q = %q, want %q", tt.class, k, got[k], v)
			}
		}
	}
}

func TestParser_AspectRatio(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		want  string
	}{
		{"aspect-square", "1 / 1"},
		{"aspect-video", "16 / 9"},
		{"aspect-9/10", "9 / 10"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got["aspect-ratio"] != tt.want {
			t.Errorf("class=%q: aspect-ratio = %v, want %q", tt.class, got["aspect-ratio"], tt.want)
		}
	}
}

func TestParser_Spacing(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"p-4", "padding", "1rem"},
		{"px-2", "padding-left", "0.5rem"},
		{"py-8", "padding-top", "2rem"},
		{"m-auto", "margin", "auto"},
		{"mt-6", "margin-top", "1.5rem"},
		{"mx-4", "margin-left", "1rem"},
		{"p-[20px]", "padding", "20px"},
		{"-mt-4", "margin-top", "-1rem"},
		{"space-x-4", "margin-left", "calc(1rem * calc(1 - var(--tw-space-x-reverse)))"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Colors(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"bg-red-500", "background-color", "#ef4444"},
		{"text-blue-600", "color", "#2563eb"},
		{"border-green-400", "border-color", "#4ade80"},
		{"bg-transparent", "background-color", "transparent"},
		{"text-white", "color", "#ffffff"},
		{"bg-[#ff0000]", "background", "#ff0000"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Typography(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"text-xs", "font-size", "0.75rem"},
		{"text-2xl", "font-size", "1.5rem"},
		{"font-bold", "font-weight", "700"},
		{"font-light", "font-weight", "300"},
		{"leading-tight", "line-height", "1.25"},
		{"tracking-wide", "letter-spacing", "0.025em"},
		{"uppercase", "text-transform", "uppercase"},
		{"italic", "font-style", "italic"},
		{"underline", "text-decoration-line", "underline"},
		{"text-left", "text-align", "left"},
		{"text-center", "text-align", "center"},
		{"antialiased", "-webkit-font-smoothing", "antialiased"},
		{"truncate", "text-overflow", "ellipsis"},
		{"whitespace-nowrap", "white-space", "nowrap"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Flexbox(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"flex-row", "flex-direction", "row"},
		{"flex-col", "flex-direction", "column"},
		{"flex-wrap", "flex-wrap", "wrap"},
		{"items-center", "align-items", "center"},
		{"justify-between", "justify-content", "space-between"},
		{"justify-center", "justify-content", "center"},
		{"self-end", "align-self", "flex-end"},
		{"flex-1", "flex", "1 1 0%"},
		{"flex-none", "flex", "none"},
		{"grow", "flex-grow", "1"},
		{"shrink-0", "flex-shrink", "0"},
		{"order-first", "order", "-9999"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Grid(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"grid-cols-3", "grid-template-columns", "repeat(3, minmax(0, 1fr))"},
		{"grid-rows-2", "grid-template-rows", "repeat(2, minmax(0, 1fr))"},
		{"col-span-2", "grid-column", "span 2 / span 2"},
		{"col-span-full", "grid-column", "1 / -1"},
		{"row-span-3", "grid-row", "span 3 / span 3"},
		{"gap-4", "gap", "1rem"},
		{"gap-x-8", "column-gap", "2rem"},
		{"gap-y-2", "row-gap", "0.5rem"},
		{"grid-flow-col", "grid-auto-flow", "column"},
		{"col-start-2", "grid-column-start", "2"},
		{"auto-cols-fr", "grid-auto-columns", "minmax(0, 1fr)"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Sizing(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"w-full", "width", "100%"},
		{"w-1/2", "width", "50%"},
		{"h-screen", "height", "100vh"},
		{"min-w-0", "min-width", "0px"},
		{"max-w-sm", "max-width", "24rem"},
		{"w-[200px]", "width", "200px"},
		{"h-auto", "height", "auto"},
		{"size-4", "width", "1rem"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Border(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"rounded", "border-radius", "0.25rem"},
		{"rounded-full", "border-radius", "9999px"},
		{"rounded-lg", "border-radius", "0.5rem"},
		{"border", "border-width", "1px"},
		{"border-2", "border-width", "2px"},
		{"border-solid", "border-style", "solid"},
		{"border-dashed", "border-style", "dashed"},
		{"border-red-500", "border-color", "#ef4444"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Position(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"absolute", "position", "absolute"},
		{"relative", "position", "relative"},
		{"fixed", "position", "fixed"},
		{"sticky", "position", "sticky"},
		{"top-4", "top", "1rem"},
		{"left-0", "left", "0px"},
		{"inset-0", "top", "0px"},
		{"-top-4", "top", "-1rem"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Effects(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"opacity-50", "opacity", "0.5"},
		{"opacity-0", "opacity", "0"},
		{"opacity-100", "opacity", "1"},
		{"shadow", "box-shadow", "0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)"},
		{"shadow-lg", "box-shadow", "0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)"},
		{"shadow-none", "box-shadow", "0 0 #0000"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_ShadowArbitraryValueNormalizesTailwindEncoding(t *testing.T) {
	p := tailwind.New()
	got := p.Parse("shadow-[0_10px_15px_-3px_rgb(0_0_0_/_0.1)]")
	want := "0 10px 15px -3px rgb(0 0 0 / 0.1)"
	if got["box-shadow"] != want {
		t.Fatalf("box-shadow = %v, want %q", got["box-shadow"], want)
	}
}

func TestParser_ShadowColorUtilityAppliesToResolvedBoxShadow(t *testing.T) {
	p := tailwind.New()
	got := p.Parse("shadow-lg shadow-red-500/25")
	want := "0 10px 15px -3px rgba(239, 68, 68, 0.25), 0 4px 6px -4px rgba(239, 68, 68, 0.25)"
	if got["box-shadow"] != want {
		t.Fatalf("box-shadow = %v, want %q", got["box-shadow"], want)
	}
}

func TestParser_Transforms(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"rotate-45", "--tw-rotate", "45deg"},
		{"-rotate-90", "--tw-rotate", "-90deg"},
		{"scale-50", "--tw-scale-x", "0.5"},
		{"translate-x-4", "--tw-translate-x", "1rem"},
		{"-translate-y-2", "--tw-translate-y", "-0.5rem"},
		{"skew-x-6", "--tw-skew-x", "6deg"},
		{"origin-center", "transform-origin", "center"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Filters(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"blur", "filter", "blur(8px)"},
		{"blur-lg", "filter", "blur(16px)"},
		{"grayscale", "filter", "grayscale(100%)"},
		{"grayscale-0", "filter", "grayscale(0)"},
		{"invert", "filter", "invert(100%)"},
		{"sepia", "filter", "sepia(100%)"},
		{"brightness-50", "filter", "brightness(50%)"},
		{"contrast-150", "filter", "contrast(150%)"},
		{"backdrop-blur-md", "backdrop-filter", "blur(12px)"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Transitions(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"transition", "transition", "color, background-color, border-color, text-decoration-color, fill, stroke, opacity, box-shadow, transform, filter, backdrop-filter 150ms cubic-bezier(0.4,0,0.2,1)"},
		{"duration-300", "transition-duration", "300ms"},
		{"ease-in-out", "transition-timing-function", "cubic-bezier(0.4, 0, 0.2, 1)"},
		{"delay-100", "transition-delay", "100ms"},
		{"animate-spin", "animation", "spin 1s linear infinite"},
		{"animate-pulse", "animation", "pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_Variants(t *testing.T) {
	p := tailwind.New()
	result := p.Parse("hover:bg-blue-500 md:flex dark:text-white sm:hidden")

	// Variants are grouped under :variant:xxx keys
	hoverProps, ok := result[":variant:hover"].(map[string]any)
	if !ok {
		t.Fatal("expected :variant:hover key")
	}
	if hoverProps["background-color"] != "#3b82f6" {
		t.Errorf("hover:bg-blue-500: got %v", hoverProps["background-color"])
	}

	mdProps, ok2 := result[":variant:md"].(map[string]any)
	if !ok2 {
		t.Fatal("expected :variant:md key")
	}
	if mdProps["display"] != "flex" {
		t.Errorf("md:flex: got %v", mdProps["display"])
	}
}

func TestParser_ArbitraryProperties(t *testing.T) {
	p := tailwind.New()
	tests := []struct {
		class string
		prop  string
		want  string
	}{
		{"[color:red]", "color", "red"},
		{"[margin-top:20px]", "margin-top", "20px"},
		{"[grid-template-areas:'a_b']", "grid-template-areas", "'a b'"},
	}
	for _, tt := range tests {
		got := p.Parse(tt.class)
		if got[tt.prop] != tt.want {
			t.Errorf("class=%q: %q = %v, want %q", tt.class, tt.prop, got[tt.prop], tt.want)
		}
	}
}

func TestParser_MultipleClasses(t *testing.T) {
	p := tailwind.New()
	result := p.Parse("flex items-center justify-between p-4 bg-white rounded-lg shadow")

	checks := map[string]string{
		"display":          "flex",
		"align-items":      "center",
		"justify-content":  "space-between",
		"padding":          "1rem",
		"background-color": "#ffffff",
		"border-radius":    "0.5rem",
		"box-shadow":       "0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)",
	}

	for prop, want := range checks {
		if result[prop] != want {
			t.Errorf("prop=%q: got %v, want %q", prop, result[prop], want)
		}
	}
}

func ExampleParser_Parse() {
	p := tailwind.New()
	props := p.Parse("flex items-center gap-4 p-6 bg-blue-500 text-white rounded-xl")
	fmt.Println("display:", props["display"])
	fmt.Println("align-items:", props["align-items"])
	fmt.Println("gap:", props["gap"])
	fmt.Println("padding:", props["padding"])
	fmt.Println("background-color:", props["background-color"])
	fmt.Println("color:", props["color"])
	fmt.Println("border-radius:", props["border-radius"])
	// Output:
	// display: flex
	// align-items: center
	// gap: 1rem
	// padding: 1.5rem
	// background-color: #3b82f6
	// color: #ffffff
	// border-radius: 0.75rem
}

func BenchmarkParser_Parse(b *testing.B) {
	p := tailwind.New()
	classes := "flex flex-col items-center justify-between gap-4 p-6 m-4 bg-white text-gray-900 rounded-xl shadow-lg border border-gray-200 hover:bg-gray-50 md:flex-row"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Parse(classes)
	}
}
