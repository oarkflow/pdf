package html

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// Node represents a node in the HTML DOM tree.
type Node struct {
	Tag      string
	Attrs    map[string]string
	Children []*Node
	Text     string
	Parent   *Node
	Style    *ComputedStyle
	Classes  []string
	ID       string
}

// ParseHTML parses HTML from a reader and returns the root node.
func ParseHTML(r io.Reader) (*Node, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	root := parseNode(doc, nil)
	if root == nil {
		root = &Node{Tag: "html", Attrs: make(map[string]string)}
	}
	return root, nil
}

func parseNode(n *html.Node, parent *Node) *Node {
	switch n.Type {
	case html.DocumentNode:
		// Wrap children under a virtual root
		root := &Node{Tag: "", Attrs: make(map[string]string), Parent: parent}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := parseNode(c, root)
			if child != nil {
				root.Children = append(root.Children, child)
			}
		}
		// If there's an html element, return it directly
		for _, ch := range root.Children {
			if ch.Tag == "html" {
				ch.Parent = parent
				return ch
			}
		}
		if len(root.Children) == 1 {
			root.Children[0].Parent = parent
			return root.Children[0]
		}
		return root

	case html.ElementNode:
		node := &Node{
			Tag:   n.Data,
			Attrs: make(map[string]string),
			Parent: parent,
		}
		for _, a := range n.Attr {
			node.Attrs[a.Key] = a.Val
			switch a.Key {
			case "class":
				for _, cls := range strings.Fields(a.Val) {
					node.Classes = append(node.Classes, cls)
				}
			case "id":
				node.ID = a.Val
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := parseNode(c, node)
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}
		return node

	case html.TextNode:
		text := n.Data
		if strings.TrimSpace(text) == "" {
			// Keep whitespace-only text nodes as single space for inline formatting
			if text != "" {
				return &Node{Text: " ", Parent: parent, Attrs: make(map[string]string)}
			}
			return nil
		}
		return &Node{Text: text, Parent: parent, Attrs: make(map[string]string)}

	default:
		return nil
	}
}

// GetBody returns the <body> element, or the root if none found.
func (n *Node) GetBody() *Node {
	return n.FindFirst("body")
}

// GetHead returns the <head> element, or nil.
func (n *Node) GetHead() *Node {
	return n.FindFirst("head")
}

// FindFirst finds the first descendant with the given tag.
func (n *Node) FindFirst(tag string) *Node {
	if n.Tag == tag {
		return n
	}
	for _, c := range n.Children {
		if found := c.FindFirst(tag); found != nil {
			return found
		}
	}
	return nil
}

// FindAll finds all descendants with the given tag.
func (n *Node) FindAll(tag string) []*Node {
	var results []*Node
	if n.Tag == tag {
		results = append(results, n)
	}
	for _, c := range n.Children {
		results = append(results, c.FindAll(tag)...)
	}
	return results
}

// GetAttribute returns the attribute value or empty string.
func (n *Node) GetAttribute(name string) string {
	if n.Attrs == nil {
		return ""
	}
	return n.Attrs[name]
}

// IsElement returns true if this is an element node (not text).
func (n *Node) IsElement() bool {
	return n.Tag != "" && n.Text == ""
}

// IsText returns true if this is a text node.
func (n *Node) IsText() bool {
	return n.Tag == "" && n.Text != ""
}

// ChildIndex returns the index of this node among its parent's children, or -1.
func (n *Node) ChildIndex() int {
	if n.Parent == nil {
		return -1
	}
	for i, c := range n.Parent.Children {
		if c == n {
			return i
		}
	}
	return -1
}

// ElementIndex returns the index among sibling elements (ignoring text nodes).
func (n *Node) ElementIndex() int {
	if n.Parent == nil {
		return -1
	}
	idx := 0
	for _, c := range n.Parent.Children {
		if c == n {
			return idx
		}
		if c.IsElement() {
			idx++
		}
	}
	return -1
}

// ElementCount returns the number of element children of the parent.
func (n *Node) ElementCount() int {
	if n.Parent == nil {
		return 0
	}
	count := 0
	for _, c := range n.Parent.Children {
		if c.IsElement() {
			count++
		}
	}
	return count
}

// PreviousElementSibling returns the previous sibling element, or nil.
func (n *Node) PreviousElementSibling() *Node {
	if n.Parent == nil {
		return nil
	}
	var prev *Node
	for _, c := range n.Parent.Children {
		if c == n {
			return prev
		}
		if c.IsElement() {
			prev = c
		}
	}
	return nil
}

// NextElementSibling returns the next sibling element, or nil.
func (n *Node) NextElementSibling() *Node {
	if n.Parent == nil {
		return nil
	}
	found := false
	for _, c := range n.Parent.Children {
		if found && c.IsElement() {
			return c
		}
		if c == n {
			found = true
		}
	}
	return nil
}

// TextContent returns the concatenated text of all descendants.
func (n *Node) TextContent() string {
	if n.IsText() {
		return n.Text
	}
	var sb strings.Builder
	for _, c := range n.Children {
		sb.WriteString(c.TextContent())
	}
	return sb.String()
}

// SetAttribute sets an attribute on the node, keeping Classes and ID in sync.
func (n *Node) SetAttribute(name, value string) {
	if n.Attrs == nil {
		n.Attrs = make(map[string]string)
	}
	n.Attrs[name] = value
	switch name {
	case "class":
		n.Classes = strings.Fields(value)
	case "id":
		n.ID = value
	}
}

// SetTextContent clears children and sets a single text child.
func (n *Node) SetTextContent(text string) {
	n.Children = nil
	if text != "" {
		n.Children = []*Node{{Text: text, Parent: n, Attrs: make(map[string]string)}}
	}
}

// AppendChild appends a child node and sets its parent.
func (n *Node) AppendChild(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// RemoveChild removes a child from the children slice.
func (n *Node) RemoveChild(child *Node) {
	for i, c := range n.Children {
		if c == child {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			child.Parent = nil
			return
		}
	}
}

// CreateElement creates a new element node with the given tag.
func CreateElement(tag string) *Node {
	return &Node{Tag: tag, Attrs: make(map[string]string)}
}

// CreateTextNode creates a new text node.
func CreateTextNode(text string) *Node {
	return &Node{Text: text, Attrs: make(map[string]string)}
}

// HasClass returns true if the node has the given class.
func (n *Node) HasClass(class string) bool {
	for _, c := range n.Classes {
		if c == class {
			return true
		}
	}
	return false
}
