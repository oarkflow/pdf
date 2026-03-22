package html

import (
	"strconv"
	"strings"
)

// Selector represents a single CSS selector.
type Selector struct {
	Type          string // element type or *
	Classes       []string
	ID            string
	PseudoClass   string // :first-child, :last-child, :nth-child(n)
	PseudoElement string // ::before, ::after
	Combinator    string // "", " ", ">", "+", "~"
	Next          *Selector
	Attribute     *AttrSelector
}

// AttrSelector represents an attribute selector like [href^="https"].
type AttrSelector struct {
	Name  string
	Op    string // "=", "~=", "|=", "^=", "$=", "*="
	Value string
}

// ParseSelectorList parses a comma-separated list of selectors.
func ParseSelectorList(s string) ([]Selector, error) {
	parts := splitSelectorList(s)
	var selectors []Selector
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		sel, err := ParseSelector(part)
		if err != nil {
			continue
		}
		selectors = append(selectors, sel...)
	}
	return selectors, nil
}

func splitSelectorList(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// ParseSelector parses a single compound selector (may contain combinators).
func ParseSelector(s string) ([]Selector, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	// Tokenize into parts separated by combinators
	var root *Selector
	current := &Selector{}
	root = current

	i := 0
	for i < len(s) {
		ch := s[i]
		switch {
		case ch == '#':
			i++
			current.ID = readSelectorIdent(s, &i)

		case ch == '.':
			i++
			current.Classes = append(current.Classes, readSelectorIdent(s, &i))

		case ch == '[':
			i++
			attr := parseAttrSelector(s, &i)
			current.Attribute = attr

		case ch == ':':
			i++
			if i < len(s) && s[i] == ':' {
				// Pseudo-element
				i++
				start := i
				for i < len(s) && isIdentCharByte(s[i]) {
					i++
				}
				current.PseudoElement = s[start:i]
			} else {
				// Pseudo-class
				start := i
				for i < len(s) && (isIdentCharByte(s[i]) || s[i] == '(' || s[i] == ')' || s[i] == 'n' || s[i] == '+' || s[i] == '-') {
					if s[i] == '(' {
						for i < len(s) && s[i] != ')' {
							i++
						}
						if i < len(s) {
							i++
						}
						break
					}
					i++
				}
				current.PseudoClass = s[start:i]
			}

		case ch == '>' || ch == '+' || ch == '~':
			i++
			for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
				i++
			}
			next := &Selector{}
			current.Combinator = string(ch)
			current.Next = next
			current = next

		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
				i++
			}
			if i < len(s) && s[i] != '>' && s[i] != '+' && s[i] != '~' && s[i] != ',' {
				next := &Selector{}
				current.Combinator = " "
				current.Next = next
				current = next
			}

		case ch == '*':
			current.Type = "*"
			i++

		default:
			if isIdentCharByte(ch) {
				current.Type = readSelectorIdent(s, &i)
			} else {
				i++
			}
		}
	}

	return []Selector{*root}, nil
}

func isIdentCharByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

func readSelectorIdent(s string, i *int) string {
	start := *i
	var b strings.Builder
	for *i < len(s) {
		switch {
		case isIdentCharByte(s[*i]):
			b.WriteByte(s[*i])
			*i++
		case s[*i] == '\\' && *i+1 < len(s):
			*i++
			b.WriteByte(s[*i])
			*i++
		default:
			if b.Len() == 0 {
				return s[start:*i]
			}
			return b.String()
		}
	}
	if b.Len() == 0 {
		return s[start:*i]
	}
	return b.String()
}

func parseAttrSelector(s string, i *int) *AttrSelector {
	attr := &AttrSelector{}
	start := *i
	for *i < len(s) && s[*i] != '=' && s[*i] != ']' && s[*i] != '~' && s[*i] != '|' && s[*i] != '^' && s[*i] != '$' && s[*i] != '*' {
		*i++
	}
	attr.Name = strings.TrimSpace(s[start:*i])

	if *i < len(s) && s[*i] == ']' {
		*i++
		return attr
	}

	// Read operator
	if *i < len(s) {
		if *i+1 < len(s) && s[*i+1] == '=' {
			attr.Op = string(s[*i]) + "="
			*i += 2
		} else if s[*i] == '=' {
			attr.Op = "="
			*i++
		}
	}

	// Read value
	for *i < len(s) && (s[*i] == ' ' || s[*i] == '\t') {
		*i++
	}
	if *i < len(s) && (s[*i] == '"' || s[*i] == '\'') {
		quote := s[*i]
		*i++
		start = *i
		for *i < len(s) && s[*i] != quote {
			*i++
		}
		attr.Value = s[start:*i]
		if *i < len(s) {
			*i++
		}
	} else {
		start = *i
		for *i < len(s) && s[*i] != ']' {
			*i++
		}
		attr.Value = strings.TrimSpace(s[start:*i])
	}

	for *i < len(s) && s[*i] != ']' {
		*i++
	}
	if *i < len(s) {
		*i++
	}
	return attr
}

// Matches returns true if the selector matches the given node.
func (sel *Selector) Matches(node *Node) bool {
	if node == nil || !node.IsElement() {
		return false
	}
	return matchSelector(sel, node)
}

func matchSelector(sel *Selector, node *Node) bool {
	// Match the last part first, then walk up
	var chain []*Selector
	for s := sel; s != nil; s = s.Next {
		chain = append(chain, s)
	}

	// Match from right to left
	return matchChain(chain, 0, node)
}

func matchChain(chain []*Selector, idx int, node *Node) bool {
	if idx >= len(chain) {
		return true
	}
	if node == nil {
		return false
	}

	last := len(chain) - 1
	sel := chain[last-idx]

	if !matchSimple(sel, node) {
		return false
	}

	if idx == last {
		return true
	}

	next := chain[last-idx-1]
	combinator := sel.Combinator
	if combinator == "" && idx > 0 {
		combinator = chain[last-idx+1].Combinator
	}
	_ = next

	switch sel.Combinator {
	case "", " ":
		// Descendant: any ancestor must match
		for p := node.Parent; p != nil; p = p.Parent {
			if matchChain(chain, idx+1, p) {
				return true
			}
		}
		return false

	case ">":
		// Child: parent must match
		return node.Parent != nil && matchChain(chain, idx+1, node.Parent)

	case "+":
		// Adjacent sibling
		prev := node.PreviousElementSibling()
		return prev != nil && matchChain(chain, idx+1, prev)

	case "~":
		// General sibling
		if node.Parent == nil {
			return false
		}
		for _, c := range node.Parent.Children {
			if c == node {
				return false
			}
			if c.IsElement() && matchChain(chain, idx+1, c) {
				return true
			}
		}
		return false
	}

	return false
}

func matchSimple(sel *Selector, node *Node) bool {
	// Type
	if sel.Type != "" && sel.Type != "*" && sel.Type != node.Tag {
		return false
	}

	// ID
	if sel.ID != "" && sel.ID != node.ID {
		return false
	}

	// Classes
	for _, cls := range sel.Classes {
		if !node.HasClass(cls) {
			return false
		}
	}

	// Attribute
	if sel.Attribute != nil {
		if !matchAttribute(sel.Attribute, node) {
			return false
		}
	}

	// Pseudo-class
	if sel.PseudoClass != "" {
		if !matchPseudoClass(sel.PseudoClass, node) {
			return false
		}
	}

	return true
}

func matchAttribute(attr *AttrSelector, node *Node) bool {
	val, exists := node.Attrs[attr.Name]
	if attr.Op == "" {
		return exists
	}
	if !exists {
		return false
	}
	switch attr.Op {
	case "=":
		return val == attr.Value
	case "~=":
		for _, w := range strings.Fields(val) {
			if w == attr.Value {
				return true
			}
		}
		return false
	case "|=":
		return val == attr.Value || strings.HasPrefix(val, attr.Value+"-")
	case "^=":
		return strings.HasPrefix(val, attr.Value)
	case "$=":
		return strings.HasSuffix(val, attr.Value)
	case "*=":
		return strings.Contains(val, attr.Value)
	}
	return false
}

func matchPseudoClass(pseudo string, node *Node) bool {
	switch {
	case pseudo == "first-child":
		return node.ElementIndex() == 0
	case pseudo == "last-child":
		return node.ElementIndex() == node.ElementCount()-1
	case pseudo == "only-child":
		return node.ElementCount() == 1
	case pseudo == "first-of-type":
		if node.Parent == nil {
			return false
		}
		for _, c := range node.Parent.Children {
			if c.IsElement() && c.Tag == node.Tag {
				return c == node
			}
		}
		return false
	case pseudo == "last-of-type":
		if node.Parent == nil {
			return false
		}
		var last *Node
		for _, c := range node.Parent.Children {
			if c.IsElement() && c.Tag == node.Tag {
				last = c
			}
		}
		return last == node
	case strings.HasPrefix(pseudo, "nth-child("):
		expr := pseudo[10:]
		if idx := strings.IndexByte(expr, ')'); idx >= 0 {
			expr = expr[:idx]
		}
		return matchNth(expr, node.ElementIndex()+1)
	case pseudo == "empty":
		return len(node.Children) == 0
	case pseudo == "link" || pseudo == "visited":
		return node.Tag == "a" && node.GetAttribute("href") != ""
	case pseudo == "hover", pseudo == "active", pseudo == "focus":
		return false // Not applicable in print
	case strings.HasPrefix(pseudo, "not("):
		// Simple :not() support
		inner := pseudo[4:]
		if idx := strings.IndexByte(inner, ')'); idx >= 0 {
			inner = inner[:idx]
		}
		innerSels, err := ParseSelector(inner)
		if err != nil || len(innerSels) == 0 {
			return true
		}
		return !innerSels[0].Matches(node)
	case strings.HasPrefix(pseudo, "is("), strings.HasPrefix(pseudo, "where("):
		open := strings.IndexByte(pseudo, '(')
		if open < 0 {
			return false
		}
		inner := pseudo[open+1:]
		if idx := strings.LastIndexByte(inner, ')'); idx >= 0 {
			inner = inner[:idx]
		}
		innerSels, err := ParseSelectorList(inner)
		if err != nil || len(innerSels) == 0 {
			return false
		}
		for i := range innerSels {
			if innerSels[i].Matches(node) {
				return true
			}
		}
		return false
	}
	return false
}

func matchNth(expr string, index int) bool {
	expr = strings.TrimSpace(strings.ToLower(expr))
	switch expr {
	case "odd":
		return index%2 == 1
	case "even":
		return index%2 == 0
	default:
		n, err := strconv.Atoi(expr)
		if err == nil {
			return index == n
		}
		// Simple an+b parsing
		// Handle cases like 2n, 2n+1, -n+3, n+2
		return false // Simplified
	}
}

// Specificity calculates the specificity of a selector chain.
func Specificity(sel *Selector) [4]int {
	var spec [4]int
	for s := sel; s != nil; s = s.Next {
		if s.ID != "" {
			spec[1]++
		}
		spec[2] += len(s.Classes)
		if s.Attribute != nil {
			spec[2]++
		}
		if s.PseudoClass != "" {
			spec[2]++
		}
		if s.Type != "" && s.Type != "*" {
			spec[3]++
		}
		if s.PseudoElement != "" {
			spec[3]++
		}
	}
	return spec
}

// CompareSpecificity compares two specificities. Returns -1, 0, or 1.
func CompareSpecificity(a, b [4]int) int {
	for i := 0; i < 4; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
