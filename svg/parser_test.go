package svg

import "testing"

func TestParseSimpleSVG(t *testing.T) {
	data := []byte(`<svg width="100" height="100"><rect x="10" y="10" width="80" height="80"/></svg>`)
	node, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if node.Tag != "svg" {
		t.Errorf("root tag = %q, want svg", node.Tag)
	}
	if node.Attrs["width"] != "100" {
		t.Errorf("width = %q", node.Attrs["width"])
	}
}

func TestParseChildren(t *testing.T) {
	data := []byte(`<svg><g><rect/><circle/></g></svg>`)
	node, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child (g), got %d", len(node.Children))
	}
	g := node.Children[0]
	if g.Tag != "g" {
		t.Errorf("tag = %q, want g", g.Tag)
	}
	if len(g.Children) != 2 {
		t.Errorf("g has %d children, want 2", len(g.Children))
	}
}

func TestParseStyleAttr(t *testing.T) {
	data := []byte(`<svg><rect style="fill:red;stroke:blue"/></svg>`)
	node, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	rect := node.Children[0]
	if rect.Style["fill"] != "red" {
		t.Errorf("fill = %q", rect.Style["fill"])
	}
	if rect.Style["stroke"] != "blue" {
		t.Errorf("stroke = %q", rect.Style["stroke"])
	}
}

func TestFindByID(t *testing.T) {
	data := []byte(`<svg><g id="group1"><rect id="r1"/></g></svg>`)
	node, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	found := node.FindByID("r1")
	if found == nil {
		t.Fatal("FindByID returned nil")
	}
	if found.Tag != "rect" {
		t.Errorf("tag = %q", found.Tag)
	}
}

func TestFindByIDNotFound(t *testing.T) {
	data := []byte(`<svg><rect id="r1"/></svg>`)
	node, _ := Parse(data)
	if node.FindByID("nonexistent") != nil {
		t.Error("expected nil for non-existent ID")
	}
}

func TestNodeAttrStyleOverride(t *testing.T) {
	data := []byte(`<svg><rect fill="green" style="fill:red"/></svg>`)
	node, _ := Parse(data)
	rect := node.Children[0]
	if rect.Attr("fill") != "red" {
		t.Errorf("Attr(fill) = %q, want red (style should override)", rect.Attr("fill"))
	}
}

func TestParseText(t *testing.T) {
	data := []byte(`<svg><text>Hello</text></svg>`)
	node, _ := Parse(data)
	if len(node.Children) == 0 {
		t.Fatal("no children")
	}
	if node.Children[0].Text != "Hello" {
		t.Errorf("text = %q", node.Children[0].Text)
	}
}

func TestParseEmpty(t *testing.T) {
	node, err := Parse([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if node == nil {
		t.Fatal("node should not be nil")
	}
}
