package html

import (
	"strings"
	"testing"
)

func TestExecuteScripts_InlineScript(t *testing.T) {
	dom, _ := ParseHTML(strings.NewReader(`<html><head><script>
		var x = 42;
	</script></head><body><p>Hello</p></body></html>`))
	err := ExecuteScripts(dom, NewFetcher(""))
	if err != nil {
		t.Fatalf("ExecuteScripts() error = %v", err)
	}
}

func TestExecuteScripts_EmptyScript(t *testing.T) {
	dom, _ := ParseHTML(strings.NewReader(`<html><head><script></script></head><body></body></html>`))
	err := ExecuteScripts(dom, NewFetcher(""))
	if err != nil {
		t.Fatalf("ExecuteScripts() error = %v", err)
	}
}

func TestShouldSkipCDN(t *testing.T) {
	tests := []struct {
		src  string
		want bool
	}{
		{"https://cdn.tailwindcss.com/v3", true},
		{"https://cdn.jsdelivr.net/npm/alpinejs@3", true},
		{"https://example.com/script.js", false},
		{"https://cdnjs.cloudflare.com/ajax/libs/jspdf/2.0/jspdf.min.js", true},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			if got := shouldSkipCDN(tt.src); got != tt.want {
				t.Errorf("shouldSkipCDN(%q) = %v, want %v", tt.src, got, tt.want)
			}
		})
	}
}

func TestExecuteScripts_AlpineXData(t *testing.T) {
	dom, _ := ParseHTML(strings.NewReader(`<html><body>
		<div x-data="{ name: 'World' }">
			<span x-text="name">placeholder</span>
		</div>
	</body></html>`))
	err := ExecuteScripts(dom, NewFetcher(""))
	if err != nil {
		t.Fatalf("ExecuteScripts() error = %v", err)
	}
	span := dom.FindFirst("span")
	if span == nil {
		t.Fatal("span not found")
	}
	text := span.TextContent()
	if text != "World" {
		t.Errorf("x-text resolved to %q, want %q", text, "World")
	}
}

func TestExecuteScripts_AlpineXShow(t *testing.T) {
	dom, _ := ParseHTML(strings.NewReader(`<html><body>
		<div x-data="{ visible: false }">
			<span x-show="visible">Hidden</span>
		</div>
	</body></html>`))
	ExecuteScripts(dom, NewFetcher(""))
	span := dom.FindFirst("span")
	if span == nil {
		t.Fatal("span not found")
	}
	style := span.GetAttribute("style")
	if !strings.Contains(style, "display:none") {
		t.Errorf("style = %q, want to contain display:none", style)
	}
}

func TestExecuteScripts_AlpineXBind(t *testing.T) {
	dom, _ := ParseHTML(strings.NewReader(`<html><body>
		<div x-data="{ url: 'https://example.com' }">
			<a x-bind:href="url">Link</a>
		</div>
	</body></html>`))
	ExecuteScripts(dom, NewFetcher(""))
	a := dom.FindFirst("a")
	if a == nil {
		t.Fatal("a not found")
	}
	if a.GetAttribute("href") != "https://example.com" {
		t.Errorf("href = %q, want https://example.com", a.GetAttribute("href"))
	}
}

func TestQuerySelect(t *testing.T) {
	dom, _ := ParseHTML(strings.NewReader(`<html><body><div id="main" class="container"><p class="text">Hello</p></div></body></html>`))
	tests := []struct {
		sel  string
		want bool
	}{
		{"#main", true},
		{".container", true},
		{".text", true},
		{"p", true},
		{"#nonexist", false},
	}
	for _, tt := range tests {
		t.Run(tt.sel, func(t *testing.T) {
			n := querySelect(dom, tt.sel)
			if (n != nil) != tt.want {
				t.Errorf("querySelect(%q) found=%v, want %v", tt.sel, n != nil, tt.want)
			}
		})
	}
}

func TestQuerySelectAll(t *testing.T) {
	dom, _ := ParseHTML(strings.NewReader(`<html><body><p>A</p><p>B</p><p>C</p></body></html>`))
	nodes := querySelectAll(dom, "p")
	if len(nodes) != 3 {
		t.Errorf("querySelectAll(p) got %d, want 3", len(nodes))
	}
}
