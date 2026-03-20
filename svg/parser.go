package svg

import (
	"encoding/xml"
	"io"
	"strings"
)

// SVGNode represents a node in the SVG document tree.
type SVGNode struct {
	Tag      string
	Attrs    map[string]string
	Children []*SVGNode
	Text     string
	Style    map[string]string // parsed style attribute
}

// Parse parses SVG XML data into a node tree.
func Parse(data []byte) (*SVGNode, error) {
	return ParseReader(strings.NewReader(string(data)))
}

// ParseReader parses SVG XML from a reader into a node tree.
func ParseReader(r io.Reader) (*SVGNode, error) {
	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var root *SVGNode
	var stack []*SVGNode

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			node := &SVGNode{
				Tag:   stripNS(t.Name),
				Attrs: make(map[string]string),
				Style: make(map[string]string),
			}
			for _, attr := range t.Attr {
				name := stripNSAttr(attr.Name)
				node.Attrs[name] = attr.Value
			}
			// Parse style attribute into Style map
			if styleStr, ok := node.Attrs["style"]; ok {
				node.Style = parseStyleAttr(styleStr)
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			} else {
				root = node
			}
			stack = append(stack, node)

		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			text := strings.TrimSpace(string(t))
			if text != "" && len(stack) > 0 {
				current := stack[len(stack)-1]
				current.Text += text
			}
		}
	}

	if root == nil {
		root = &SVGNode{
			Tag:   "svg",
			Attrs: make(map[string]string),
			Style: make(map[string]string),
		}
	}
	return root, nil
}

// stripNS strips the namespace prefix from an XML element name.
func stripNS(name xml.Name) string {
	return name.Local
}

// stripNSAttr strips the namespace prefix from an XML attribute name.
func stripNSAttr(name xml.Name) string {
	if name.Space != "" {
		// Handle known namespaces
		switch {
		case strings.Contains(name.Space, "xlink"):
			return "xlink:" + name.Local
		default:
			return name.Local
		}
	}
	return name.Local
}

// parseStyleAttr parses a CSS style attribute string into key-value pairs.
func parseStyleAttr(s string) map[string]string {
	m := make(map[string]string)
	parts := strings.Split(s, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.IndexByte(part, ':')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(part[:idx])
		val := strings.TrimSpace(part[idx+1:])
		m[key] = val
	}
	return m
}

// Attr returns the value of an attribute, checking the style map first,
// then the attrs map.
func (n *SVGNode) Attr(name string) string {
	if v, ok := n.Style[name]; ok {
		return v
	}
	return n.Attrs[name]
}

// FindByID searches the tree for a node with the given id attribute.
func (n *SVGNode) FindByID(id string) *SVGNode {
	if n.Attrs["id"] == id {
		return n
	}
	for _, child := range n.Children {
		if found := child.FindByID(id); found != nil {
			return found
		}
	}
	return nil
}
