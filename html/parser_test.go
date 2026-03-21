package html

import (
	"strings"
	"testing"
)

func TestParseHTML_BasicDocument(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantTag string
	}{
		{"minimal", "<html><body><p>Hello</p></body></html>", "html"},
		{"no html tag", "<p>Hello</p>", "html"},
		{"empty", "", "html"},
		{"just text", "Hello world", "html"},
		{"nested", "<div><span>A</span><span>B</span></div>", "html"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := ParseHTML(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("ParseHTML() error = %v", err)
			}
			if root == nil {
				t.Fatal("ParseHTML() returned nil")
			}
			if root.Tag != tt.wantTag {
				t.Errorf("root tag = %q, want %q", root.Tag, tt.wantTag)
			}
		})
	}
}

func TestFindFirst(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(`<html><head><title>T</title></head><body><div><p>Hello</p></div></body></html>`))
	tests := []struct {
		tag  string
		want bool
	}{
		{"body", true},
		{"p", true},
		{"div", true},
		{"title", true},
		{"span", false},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			found := root.FindFirst(tt.tag)
			if (found != nil) != tt.want {
				t.Errorf("FindFirst(%q) found=%v, want=%v", tt.tag, found != nil, tt.want)
			}
		})
	}
}

func TestFindAll(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(`<html><body><p>A</p><div><p>B</p></div><p>C</p></body></html>`))
	ps := root.FindAll("p")
	if len(ps) != 3 {
		t.Errorf("FindAll(p) got %d, want 3", len(ps))
	}
}

func TestTextContent(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(`<div>Hello <strong>World</strong></div>`))
	div := root.FindFirst("div")
	if div == nil {
		t.Fatal("no div found")
	}
	got := div.TextContent()
	if !strings.Contains(got, "Hello") || !strings.Contains(got, "World") {
		t.Errorf("TextContent() = %q, want to contain Hello and World", got)
	}
}

func TestHasClass(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(`<div class="foo bar baz">X</div>`))
	div := root.FindFirst("div")
	tests := []struct {
		class string
		want  bool
	}{
		{"foo", true},
		{"bar", true},
		{"baz", true},
		{"qux", false},
	}
	for _, tt := range tests {
		if got := div.HasClass(tt.class); got != tt.want {
			t.Errorf("HasClass(%q) = %v, want %v", tt.class, got, tt.want)
		}
	}
}

func TestSetAttribute(t *testing.T) {
	node := CreateElement("div")
	node.SetAttribute("id", "main")
	if node.ID != "main" {
		t.Errorf("ID = %q, want %q", node.ID, "main")
	}
	node.SetAttribute("class", "a b c")
	if len(node.Classes) != 3 {
		t.Errorf("Classes = %v, want 3 classes", node.Classes)
	}
}

func TestSetTextContent(t *testing.T) {
	node := CreateElement("p")
	node.SetTextContent("hello")
	if len(node.Children) != 1 {
		t.Fatalf("Children len = %d, want 1", len(node.Children))
	}
	if node.TextContent() != "hello" {
		t.Errorf("TextContent() = %q, want %q", node.TextContent(), "hello")
	}
	node.SetTextContent("")
	if len(node.Children) != 0 {
		t.Errorf("Children len = %d after empty SetTextContent, want 0", len(node.Children))
	}
}

func TestAppendChild(t *testing.T) {
	parent := CreateElement("div")
	child := CreateElement("span")
	parent.AppendChild(child)
	if len(parent.Children) != 1 {
		t.Fatalf("Children len = %d, want 1", len(parent.Children))
	}
	if child.Parent != parent {
		t.Error("child.Parent not set correctly")
	}
}

func TestRemoveChild(t *testing.T) {
	parent := CreateElement("div")
	c1 := CreateElement("span")
	c2 := CreateElement("p")
	parent.AppendChild(c1)
	parent.AppendChild(c2)
	parent.RemoveChild(c1)
	if len(parent.Children) != 1 {
		t.Errorf("Children len = %d after remove, want 1", len(parent.Children))
	}
	if c1.Parent != nil {
		t.Error("removed child Parent should be nil")
	}
}

func TestCreateElement(t *testing.T) {
	el := CreateElement("div")
	if el.Tag != "div" {
		t.Errorf("Tag = %q, want %q", el.Tag, "div")
	}
	if el.Attrs == nil {
		t.Error("Attrs should not be nil")
	}
	if !el.IsElement() {
		t.Error("should be element")
	}
}

func TestCreateTextNode(t *testing.T) {
	tn := CreateTextNode("hello")
	if tn.Text != "hello" {
		t.Errorf("Text = %q, want %q", tn.Text, "hello")
	}
	if !tn.IsText() {
		t.Error("should be text node")
	}
	if tn.IsElement() {
		t.Error("should not be element")
	}
}

func TestNodeNavigation(t *testing.T) {
	root, _ := ParseHTML(strings.NewReader(`<div><span>A</span><p>B</p><em>C</em></div>`))
	div := root.FindFirst("div")
	p := root.FindFirst("p")
	if p == nil || div == nil {
		t.Fatal("missing nodes")
	}
	prev := p.PreviousElementSibling()
	if prev == nil || prev.Tag != "span" {
		t.Errorf("PreviousElementSibling = %v, want span", prev)
	}
	next := p.NextElementSibling()
	if next == nil || next.Tag != "em" {
		t.Errorf("NextElementSibling = %v, want em", next)
	}
}
