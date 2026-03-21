package html

import (
	"strings"
	"testing"
)

func FuzzParseHTML(f *testing.F) {
	f.Add("<html><body><p>Hello</p></body></html>")
	f.Add("<div class='x'><span>test</span></div>")
	f.Add("")
	f.Add("<")
	f.Add("<<<>>>")

	f.Fuzz(func(t *testing.T, data string) {
		node, err := ParseHTML(strings.NewReader(data))
		if err != nil {
			return
		}
		_ = node.TextContent()
		_ = node.GetBody()
		_ = node.GetHead()
		node.FindFirst("div")
		node.FindAll("p")
	})
}

func FuzzParseCSS(f *testing.F) {
	f.Add("body { color: red; font-size: 12px; }")
	f.Add(".cls { margin: 0; }")
	f.Add("")
	f.Add("@media print { .x { display: none; } }")
	f.Add("a { b: c; } d { e: f !important; }")

	f.Fuzz(func(t *testing.T, data string) {
		_, _ = ParseCSS(data)
	})
}
