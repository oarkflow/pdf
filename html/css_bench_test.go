package html

import (
	"strings"
	"testing"
)

func BenchmarkParseCSS(b *testing.B) {
	var sb strings.Builder
	selectors := []string{
		"body", "h1", "h2", "h3", "p", "a", "a:hover", "ul", "ol", "li",
		".container", ".header", ".footer", ".nav", ".sidebar",
		"#main", "#content", "#app", ".btn", ".btn-primary",
		"table", "th", "td", "tr", "thead", "tbody",
		".card", ".card-body", ".card-title", ".card-text",
		"input", "select", "textarea", ".form-group", ".form-control",
		"img", "figure", "figcaption", ".media", ".media-body",
		".row", ".col", ".col-md-6", ".col-lg-4", ".col-sm-12",
		".text-center", ".text-right", ".font-bold", ".text-lg", ".text-sm",
		"nav a", "header h1", "footer p", ".list-item", ".badge",
	}
	for _, sel := range selectors {
		sb.WriteString(sel)
		sb.WriteString(" { color: #333; font-size: 14px; margin: 0; padding: 8px 16px; display: block; }\n")
	}
	css := sb.String()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseCSS(css)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectorMatching(b *testing.B) {
	selectors := []string{
		"div.container", "p.intro", "#main", "ul li", "a.link",
		"table tr td", ".card .card-body", "h1", ".btn-primary", "nav a",
	}
	parsed := make([][]Selector, len(selectors))
	for i, s := range selectors {
		p, err := ParseSelectorList(s)
		if err != nil {
			b.Fatal(err)
		}
		parsed[i] = p
	}

	node := &Node{
		Tag:     "a",
		Classes: []string{"link", "active"},
		ID:      "main-link",
		Attrs:   map[string]string{"href": "https://example.com", "class": "link active", "id": "main-link"},
		Parent: &Node{
			Tag:     "nav",
			Classes: []string{"navbar"},
			Attrs:   map[string]string{"class": "navbar"},
			Parent: &Node{
				Tag:   "div",
				Attrs: map[string]string{},
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, sels := range parsed {
			for j := range sels {
				sels[j].Matches(node)
			}
		}
	}
}

func BenchmarkInlineStyleParsing(b *testing.B) {
	styles := []string{
		"color: #333; font-size: 14px; margin: 10px 20px; padding: 8px; display: flex; align-items: center; justify-content: space-between; background-color: #f0f0f0; border: 1px solid #ccc; border-radius: 4px;",
		"width: 100%; max-width: 1200px; margin: 0 auto; font-family: Arial, sans-serif; line-height: 1.6;",
		"position: relative; top: 0; left: 0; z-index: 100; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range styles {
			ParseInlineStyle(s)
		}
	}
}
