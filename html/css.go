package html

import (
	"fmt"
	"strings"
	"unicode"
)

// CSSRule represents a single CSS rule with selectors and properties.
type CSSRule struct {
	Selectors   []Selector
	Properties  map[string]CSSValue
	Specificity [4]int
}

// CSSValue represents a CSS property value with optional !important.
type CSSValue struct {
	Value    string
	Priority bool // !important
}

// Stylesheet is a collection of CSS rules.
type Stylesheet struct {
	Rules     []CSSRule
	FontFaces []FontFaceRule
	Pages     []PageAtRule
}

// FontFaceRule represents a @font-face rule.
type FontFaceRule struct {
	Family string
	Src    string
	Weight string
	Style  string
}

// PageAtRule represents a @page rule.
type PageAtRule struct {
	Properties map[string]CSSValue
}

// ParseCSS parses a CSS string into a Stylesheet.
func ParseCSS(css string) (*Stylesheet, error) {
	p := &cssParser{input: css, pos: 0}
	return p.parse()
}

// ParseInlineStyle parses an inline style attribute value.
func ParseInlineStyle(style string) map[string]CSSValue {
	props := make(map[string]CSSValue)
	declarations := splitDeclarations(style)
	for _, decl := range declarations {
		name, val := parseDeclaration(decl)
		if name != "" {
			expanded := expandShorthand(name, val)
			for k, v := range expanded {
				props[k] = v
			}
		}
	}
	return props
}

type cssParser struct {
	input string
	pos   int
}

func (p *cssParser) parse() (*Stylesheet, error) {
	sheet := &Stylesheet{}
	p.skipWhitespaceAndComments()
	for p.pos < len(p.input) {
		p.skipWhitespaceAndComments()
		if p.pos >= len(p.input) {
			break
		}
		if p.input[p.pos] == '@' {
			p.parseAtRule(sheet)
		} else {
			rule := p.parseRule()
			if rule != nil {
				sheet.Rules = append(sheet.Rules, *rule)
			}
		}
		p.skipWhitespaceAndComments()
	}
	return sheet, nil
}

func (p *cssParser) parseAtRule(sheet *Stylesheet) {
	p.pos++ // skip @
	keyword := p.readIdent()
	p.skipWhitespaceAndComments()

	switch strings.ToLower(keyword) {
	case "media":
		p.parseMediaRule(sheet)
	case "font-face":
		p.parseFontFace(sheet)
	case "page":
		p.parsePageRule(sheet)
	case "import":
		// Skip @import for now (handled by fetcher)
		p.skipUntil(';')
		if p.pos < len(p.input) {
			p.pos++
		}
	default:
		// Skip unknown at-rules
		depth := 0
		for p.pos < len(p.input) {
			if p.input[p.pos] == '{' {
				depth++
			} else if p.input[p.pos] == '}' {
				depth--
				if depth <= 0 {
					p.pos++
					break
				}
			} else if depth == 0 && p.input[p.pos] == ';' {
				p.pos++
				break
			}
			p.pos++
		}
	}
}

func (p *cssParser) parseMediaRule(sheet *Stylesheet) {
	// Read media query
	mediaQuery := strings.TrimSpace(p.readUntil('{'))
	if p.pos < len(p.input) {
		p.pos++ // skip {
	}

	// For simplicity, only include screen and print and all
	include := strings.Contains(mediaQuery, "print") ||
		strings.Contains(mediaQuery, "all") ||
		strings.Contains(mediaQuery, "screen")

	depth := 1
	start := p.pos
	for p.pos < len(p.input) && depth > 0 {
		if p.input[p.pos] == '{' {
			depth++
		} else if p.input[p.pos] == '}' {
			depth--
		}
		if depth > 0 {
			p.pos++
		}
	}
	content := p.input[start:p.pos]
	if p.pos < len(p.input) {
		p.pos++ // skip }
	}

	if include {
		innerParser := &cssParser{input: content, pos: 0}
		innerSheet, _ := innerParser.parse()
		if innerSheet != nil {
			sheet.Rules = append(sheet.Rules, innerSheet.Rules...)
		}
	}
}

func (p *cssParser) parseFontFace(sheet *Stylesheet) {
	p.skipWhitespaceAndComments()
	if p.pos < len(p.input) && p.input[p.pos] == '{' {
		p.pos++
	}
	content := p.readUntil('}')
	if p.pos < len(p.input) {
		p.pos++
	}
	props := ParseInlineStyle(content)
	ff := FontFaceRule{
		Family: stripQuotes(props["font-family"].Value),
		Src:    props["src"].Value,
		Weight: props["font-weight"].Value,
		Style:  props["font-style"].Value,
	}
	sheet.FontFaces = append(sheet.FontFaces, ff)
}

func (p *cssParser) parsePageRule(sheet *Stylesheet) {
	p.skipWhitespaceAndComments()
	if p.pos < len(p.input) && p.input[p.pos] == '{' {
		p.pos++
	}
	content := p.readUntil('}')
	if p.pos < len(p.input) {
		p.pos++
	}
	props := ParseInlineStyle(content)
	sheet.Pages = append(sheet.Pages, PageAtRule{Properties: props})
}

func (p *cssParser) parseRule() *CSSRule {
	selectorStr := strings.TrimSpace(p.readUntil('{'))
	if p.pos < len(p.input) {
		p.pos++ // skip {
	}
	if selectorStr == "" {
		p.skipUntil('}')
		if p.pos < len(p.input) {
			p.pos++
		}
		return nil
	}

	propsStr := p.readUntil('}')
	if p.pos < len(p.input) {
		p.pos++ // skip }
	}

	selectors, err := ParseSelectorList(selectorStr)
	if err != nil {
		return nil
	}

	props := ParseInlineStyle(propsStr)
	if len(props) == 0 && len(selectors) == 0 {
		return nil
	}

	return &CSSRule{
		Selectors:  selectors,
		Properties: props,
	}
}

func (p *cssParser) skipWhitespaceAndComments() {
	for p.pos < len(p.input) {
		if unicode.IsSpace(rune(p.input[p.pos])) {
			p.pos++
		} else if p.pos+1 < len(p.input) && p.input[p.pos] == '/' && p.input[p.pos+1] == '*' {
			p.pos += 2
			for p.pos+1 < len(p.input) {
				if p.input[p.pos] == '*' && p.input[p.pos+1] == '/' {
					p.pos += 2
					break
				}
				p.pos++
			}
		} else {
			break
		}
	}
}

func (p *cssParser) readIdent() string {
	start := p.pos
	for p.pos < len(p.input) && (isIdentChar(p.input[p.pos]) || p.input[p.pos] == '-') {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *cssParser) readUntil(ch byte) string {
	start := p.pos
	depth := 0
	var quote byte
	for p.pos < len(p.input) {
		if quote != 0 {
			if p.input[p.pos] == '\\' && p.pos+1 < len(p.input) {
				p.pos += 2
				continue
			}
			if p.input[p.pos] == quote {
				quote = 0
			}
		} else if p.input[p.pos] == '"' || p.input[p.pos] == '\'' {
			quote = p.input[p.pos]
		} else if p.input[p.pos] == '(' || p.input[p.pos] == '[' {
			depth++
		} else if p.input[p.pos] == ')' || p.input[p.pos] == ']' {
			depth--
		} else if depth == 0 && p.input[p.pos] == ch {
			return p.input[start:p.pos]
		}
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *cssParser) skipUntil(ch byte) {
	for p.pos < len(p.input) && p.input[p.pos] != ch {
		p.pos++
	}
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

func splitDeclarations(style string) []string {
	var result []string
	depth := 0
	start := 0
	var quote byte
	for i := 0; i < len(style); i++ {
		if quote != 0 {
			if style[i] == '\\' && i+1 < len(style) {
				i++
				continue
			}
			if style[i] == quote {
				quote = 0
			}
			continue
		}
		switch style[i] {
		case '"', '\'':
			quote = style[i]
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ';':
			if depth == 0 {
				decl := strings.TrimSpace(style[start:i])
				if decl != "" {
					result = append(result, decl)
				}
				start = i + 1
			}
		}
	}
	last := strings.TrimSpace(style[start:])
	if last != "" {
		result = append(result, last)
	}
	return result
}

func parseDeclaration(decl string) (string, CSSValue) {
	idx := findCSSDelimiter(decl, ':')
	if idx < 0 {
		return "", CSSValue{}
	}
	name := strings.TrimSpace(decl[:idx])
	value := strings.TrimSpace(decl[idx+1:])
	priority := false
	if strings.HasSuffix(strings.ToLower(value), "!important") {
		priority = true
		value = strings.TrimSpace(value[:len(value)-10])
	}
	return strings.ToLower(name), CSSValue{Value: value, Priority: priority}
}

func findCSSDelimiter(s string, delim byte) int {
	depth := 0
	var quote byte
	for i := 0; i < len(s); i++ {
		if quote != 0 {
			if s[i] == '\\' && i+1 < len(s) {
				i++
				continue
			}
			if s[i] == quote {
				quote = 0
			}
			continue
		}
		switch s[i] {
		case '"', '\'':
			quote = s[i]
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && s[i] == delim {
				return i
			}
		}
	}
	return -1
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// expandShorthand expands CSS shorthand properties.
func expandShorthand(name string, val CSSValue) map[string]CSSValue {
	result := make(map[string]CSSValue)
	switch name {
	case "margin":
		expandBoxShorthand("margin", val, result)
	case "padding":
		expandBoxShorthand("padding", val, result)
	case "border":
		expandBorderShorthand(val, result)
	case "border-top", "border-right", "border-bottom", "border-left":
		expandBorderSideShorthand(name, val, result)
	case "border-width":
		parts := splitCSSValues(val.Value)
		sides := expandFourValues(parts)
		result["border-top-width"] = CSSValue{Value: sides[0], Priority: val.Priority}
		result["border-right-width"] = CSSValue{Value: sides[1], Priority: val.Priority}
		result["border-bottom-width"] = CSSValue{Value: sides[2], Priority: val.Priority}
		result["border-left-width"] = CSSValue{Value: sides[3], Priority: val.Priority}
	case "border-style":
		parts := splitCSSValues(val.Value)
		sides := expandFourValues(parts)
		result["border-top-style"] = CSSValue{Value: sides[0], Priority: val.Priority}
		result["border-right-style"] = CSSValue{Value: sides[1], Priority: val.Priority}
		result["border-bottom-style"] = CSSValue{Value: sides[2], Priority: val.Priority}
		result["border-left-style"] = CSSValue{Value: sides[3], Priority: val.Priority}
	case "border-color":
		parts := splitCSSValues(val.Value)
		sides := expandFourValues(parts)
		result["border-top-color"] = CSSValue{Value: sides[0], Priority: val.Priority}
		result["border-right-color"] = CSSValue{Value: sides[1], Priority: val.Priority}
		result["border-bottom-color"] = CSSValue{Value: sides[2], Priority: val.Priority}
		result["border-left-color"] = CSSValue{Value: sides[3], Priority: val.Priority}
	case "border-radius":
		result["border-radius"] = val
	case "background":
		expandBackgroundShorthand(val, result)
	case "box-shadow":
		result["box-shadow"] = val
	case "font":
		expandFontShorthand(val, result)
	case "list-style":
		expandListStyleShorthand(val, result)
	case "flex":
		expandFlexShorthand(val, result)
	case "gap":
		result["gap"] = val
	case "flex-flow":
		parts := splitCSSValues(val.Value)
		for _, part := range parts {
			switch part {
			case "row", "row-reverse", "column", "column-reverse":
				result["flex-direction"] = CSSValue{Value: part, Priority: val.Priority}
			case "nowrap", "wrap", "wrap-reverse":
				result["flex-wrap"] = CSSValue{Value: part, Priority: val.Priority}
			}
		}
	case "text-decoration":
		result["text-decoration"] = val
	case "overflow":
		result["overflow"] = val
	default:
		result[name] = val
	}
	return result
}

func expandBoxShorthand(prefix string, val CSSValue, result map[string]CSSValue) {
	parts := splitCSSValues(val.Value)
	sides := expandFourValues(parts)
	result[prefix+"-top"] = CSSValue{Value: sides[0], Priority: val.Priority}
	result[prefix+"-right"] = CSSValue{Value: sides[1], Priority: val.Priority}
	result[prefix+"-bottom"] = CSSValue{Value: sides[2], Priority: val.Priority}
	result[prefix+"-left"] = CSSValue{Value: sides[3], Priority: val.Priority}
}

func expandFourValues(parts []string) [4]string {
	switch len(parts) {
	case 0:
		return [4]string{"0", "0", "0", "0"}
	case 1:
		return [4]string{parts[0], parts[0], parts[0], parts[0]}
	case 2:
		return [4]string{parts[0], parts[1], parts[0], parts[1]}
	case 3:
		return [4]string{parts[0], parts[1], parts[2], parts[1]}
	default:
		return [4]string{parts[0], parts[1], parts[2], parts[3]}
	}
}

func expandBorderShorthand(val CSSValue, result map[string]CSSValue) {
	parts := splitCSSValues(val.Value)
	width, style, color := "medium", "none", "currentcolor"
	for _, part := range parts {
		lower := strings.ToLower(part)
		if isBorderStyle(lower) {
			style = lower
		} else if isColor(lower) {
			color = part
		} else {
			width = part
		}
	}
	for _, side := range []string{"top", "right", "bottom", "left"} {
		result[fmt.Sprintf("border-%s-width", side)] = CSSValue{Value: width, Priority: val.Priority}
		result[fmt.Sprintf("border-%s-style", side)] = CSSValue{Value: style, Priority: val.Priority}
		result[fmt.Sprintf("border-%s-color", side)] = CSSValue{Value: color, Priority: val.Priority}
	}
}

func expandBorderSideShorthand(name string, val CSSValue, result map[string]CSSValue) {
	parts := splitCSSValues(val.Value)
	width, style, color := "medium", "none", "currentcolor"
	for _, part := range parts {
		lower := strings.ToLower(part)
		if isBorderStyle(lower) {
			style = lower
		} else if isColor(lower) {
			color = part
		} else {
			width = part
		}
	}
	result[name+"-width"] = CSSValue{Value: width, Priority: val.Priority}
	result[name+"-style"] = CSSValue{Value: style, Priority: val.Priority}
	result[name+"-color"] = CSSValue{Value: color, Priority: val.Priority}
}

func expandBackgroundShorthand(val CSSValue, result map[string]CSSValue) {
	v := strings.TrimSpace(val.Value)
	if v == "" {
		return
	}
	layer := strings.TrimSpace(firstTopLevelCSSLayer(v))
	if color := extractBackgroundColor(layer); color != "" {
		result["background-color"] = CSSValue{Value: color, Priority: val.Priority}
	}
	if position := extractBackgroundPosition(layer); position != "" {
		result["background-position"] = CSSValue{Value: position, Priority: val.Priority}
	}
	if size := extractBackgroundSize(layer); size != "" {
		result["background-size"] = CSSValue{Value: size, Priority: val.Priority}
	}
	if repeat := extractBackgroundRepeat(layer); repeat != "" {
		result["background-repeat"] = CSSValue{Value: repeat, Priority: val.Priority}
	}
	if image := extractBackgroundImage(layer); image != "" {
		result["background-image"] = CSSValue{Value: image, Priority: val.Priority}
		return
	}
	if _, ok := result["background-color"]; !ok {
		result["background-color"] = CSSValue{Value: layer, Priority: val.Priority}
	}
}

func extractBackgroundColor(v string) string {
	parts := splitCSSValues(v)
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		if isColor(lower) || strings.HasPrefix(lower, "#") || strings.HasPrefix(lower, "rgb") || strings.HasPrefix(lower, "hsl") {
			return part
		}
	}
	return ""
}

func extractBackgroundImage(v string) string {
	for _, part := range splitCSSValues(v) {
		if isBackgroundImageToken(strings.ToLower(strings.TrimSpace(part))) {
			return strings.TrimSpace(part)
		}
	}
	return ""
}

func extractBackgroundPosition(v string) string {
	positionPart, _ := splitBackgroundPositionSize(v)
	positionTokens := filterBackgroundPositionTokens(splitCSSValues(positionPart))
	if len(positionTokens) == 0 {
		return ""
	}
	return strings.Join(positionTokens, " ")
}

func extractBackgroundSize(v string) string {
	_, sizePart := splitBackgroundPositionSize(v)
	sizeTokens := filterBackgroundSizeTokens(splitCSSValues(sizePart))
	if len(sizeTokens) == 0 {
		return ""
	}
	return strings.Join(sizeTokens, " ")
}

func extractBackgroundRepeat(v string) string {
	for _, part := range splitCSSValues(v) {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "repeat", "repeat-x", "repeat-y", "no-repeat", "space", "round":
			return strings.ToLower(strings.TrimSpace(part))
		}
	}
	return ""
}

func splitBackgroundPositionSize(v string) (string, string) {
	depth := 0
	var quote byte
	for i := 0; i < len(v); i++ {
		if quote != 0 {
			if v[i] == '\\' && i+1 < len(v) {
				i++
				continue
			}
			if v[i] == quote {
				quote = 0
			}
			continue
		}
		switch v[i] {
		case '"', '\'':
			quote = v[i]
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '/':
			if depth == 0 {
				return strings.TrimSpace(v[:i]), strings.TrimSpace(v[i+1:])
			}
		}
	}
	return strings.TrimSpace(v), ""
}

func filterBackgroundPositionTokens(parts []string) []string {
	var out []string
	for _, part := range parts {
		lower := strings.ToLower(strings.TrimSpace(part))
		if lower == "" || isBackgroundImageToken(lower) || isBackgroundColorToken(lower) || isBackgroundRepeatToken(lower) ||
			isBackgroundAttachmentToken(lower) || isBackgroundOriginOrClipToken(lower) {
			continue
		}
		if isBackgroundPositionToken(lower) {
			out = append(out, part)
		}
	}
	return out
}

func filterBackgroundSizeTokens(parts []string) []string {
	var out []string
	for _, part := range parts {
		lower := strings.ToLower(strings.TrimSpace(part))
		if isBackgroundSizeToken(lower) {
			out = append(out, part)
		}
	}
	return out
}

func isBackgroundImageToken(v string) bool {
	return strings.Contains(v, "gradient(") || strings.HasPrefix(v, "url(")
}

func isBackgroundColorToken(v string) bool {
	return isColor(v) || strings.HasPrefix(v, "#") || strings.HasPrefix(v, "rgb") || strings.HasPrefix(v, "hsl")
}

func isBackgroundRepeatToken(v string) bool {
	switch v {
	case "repeat", "repeat-x", "repeat-y", "no-repeat", "space", "round":
		return true
	default:
		return false
	}
}

func isBackgroundAttachmentToken(v string) bool {
	switch v {
	case "scroll", "fixed", "local":
		return true
	default:
		return false
	}
}

func isBackgroundOriginOrClipToken(v string) bool {
	switch v {
	case "border-box", "padding-box", "content-box", "text":
		return true
	default:
		return false
	}
}

func isBackgroundPositionToken(v string) bool {
	switch v {
	case "left", "center", "right", "top", "bottom":
		return true
	default:
		l := parseLength(v)
		return l.Unit != "" || v == "0"
	}
}

func isBackgroundSizeToken(v string) bool {
	switch v {
	case "auto", "cover", "contain":
		return true
	default:
		l := parseLength(v)
		return l.Unit != "" || v == "0"
	}
}

func expandFontShorthand(val CSSValue, result map[string]CSSValue) {
	// font: [style] [weight] size[/line-height] family
	parts := splitCSSValues(val.Value)
	if len(parts) == 0 {
		return
	}
	idx := 0
	// Check for style
	if idx < len(parts) && (parts[idx] == "italic" || parts[idx] == "oblique" || parts[idx] == "normal") {
		result["font-style"] = CSSValue{Value: parts[idx], Priority: val.Priority}
		idx++
	}
	// Check for weight
	if idx < len(parts) && isFontWeight(parts[idx]) {
		result["font-weight"] = CSSValue{Value: parts[idx], Priority: val.Priority}
		idx++
	}
	// Size (possibly with /line-height)
	if idx < len(parts) {
		sizeStr := parts[idx]
		if slashIdx := strings.IndexByte(sizeStr, '/'); slashIdx >= 0 {
			result["font-size"] = CSSValue{Value: sizeStr[:slashIdx], Priority: val.Priority}
			result["line-height"] = CSSValue{Value: sizeStr[slashIdx+1:], Priority: val.Priority}
		} else {
			result["font-size"] = CSSValue{Value: sizeStr, Priority: val.Priority}
			// Check if next is /line-height
			if idx+1 < len(parts) && parts[idx+1] == "/" && idx+2 < len(parts) {
				result["line-height"] = CSSValue{Value: parts[idx+2], Priority: val.Priority}
				idx += 2
			}
		}
		idx++
	}
	// Rest is font-family
	if idx < len(parts) {
		family := strings.Join(parts[idx:], " ")
		result["font-family"] = CSSValue{Value: family, Priority: val.Priority}
	}
}

func expandListStyleShorthand(val CSSValue, result map[string]CSSValue) {
	parts := splitCSSValues(val.Value)
	for _, part := range parts {
		lower := strings.ToLower(part)
		if lower == "inside" || lower == "outside" {
			result["list-style-position"] = CSSValue{Value: lower, Priority: val.Priority}
		} else if lower == "none" || lower == "disc" || lower == "circle" || lower == "square" ||
			lower == "decimal" || lower == "lower-alpha" || lower == "upper-alpha" ||
			lower == "lower-roman" || lower == "upper-roman" {
			result["list-style-type"] = CSSValue{Value: lower, Priority: val.Priority}
		}
	}
}

func expandFlexShorthand(val CSSValue, result map[string]CSSValue) {
	v := strings.TrimSpace(val.Value)
	if v == "none" {
		result["flex-grow"] = CSSValue{Value: "0", Priority: val.Priority}
		result["flex-shrink"] = CSSValue{Value: "0", Priority: val.Priority}
		result["flex-basis"] = CSSValue{Value: "auto", Priority: val.Priority}
		return
	}
	if v == "auto" {
		result["flex-grow"] = CSSValue{Value: "1", Priority: val.Priority}
		result["flex-shrink"] = CSSValue{Value: "1", Priority: val.Priority}
		result["flex-basis"] = CSSValue{Value: "auto", Priority: val.Priority}
		return
	}
	parts := splitCSSValues(v)
	switch len(parts) {
	case 1:
		result["flex-grow"] = CSSValue{Value: parts[0], Priority: val.Priority}
		result["flex-shrink"] = CSSValue{Value: "1", Priority: val.Priority}
		result["flex-basis"] = CSSValue{Value: "0", Priority: val.Priority}
	case 2:
		result["flex-grow"] = CSSValue{Value: parts[0], Priority: val.Priority}
		result["flex-shrink"] = CSSValue{Value: parts[1], Priority: val.Priority}
		result["flex-basis"] = CSSValue{Value: "0", Priority: val.Priority}
	case 3:
		result["flex-grow"] = CSSValue{Value: parts[0], Priority: val.Priority}
		result["flex-shrink"] = CSSValue{Value: parts[1], Priority: val.Priority}
		result["flex-basis"] = CSSValue{Value: parts[2], Priority: val.Priority}
	}
}

func splitCSSValues(s string) []string {
	var parts []string
	depth := 0
	start := 0
	s = strings.TrimSpace(s)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ' ', '\t', '\n', '\r':
			if depth == 0 {
				part := strings.TrimSpace(s[start:i])
				if part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		case ',':
			if depth == 0 {
				part := strings.TrimSpace(s[start:i])
				if part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		}
	}
	if start < len(s) {
		part := strings.TrimSpace(s[start:])
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func isBorderStyle(s string) bool {
	switch s {
	case "none", "hidden", "dotted", "dashed", "solid", "double", "groove", "ridge", "inset", "outset":
		return true
	}
	return false
}

func isColor(s string) bool {
	s = strings.ToLower(s)
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "rgb") || strings.HasPrefix(s, "hsl") {
		return true
	}
	_, ok := namedColors[s]
	return ok
}

func isFontWeight(s string) bool {
	switch strings.ToLower(s) {
	case "bold", "bolder", "lighter", "100", "200", "300", "400", "500", "600", "700", "800", "900":
		return true
	}
	return false
}
