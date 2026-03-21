package html

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"unicode"

	"github.com/oarkflow/fasttpl"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/tailwind"
)

// applyTextTransform applies the CSS text-transform property to text.
func applyTextTransform(text, transform string) string {
	switch transform {
	case "uppercase":
		return strings.ToUpper(text)
	case "lowercase":
		return strings.ToLower(text)
	case "capitalize":
		runes := []rune(text)
		inWord := false
		for i, r := range runes {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				if !inWord {
					runes[i] = unicode.ToUpper(r)
					inWord = true
				}
			} else {
				inWord = false
			}
		}
		return string(runes)
	default:
		return text
	}
}

// Options configures the HTML to PDF conversion.
type Options struct {
	DefaultFontSize   float64
	DefaultFontFamily string
	PageSize          [2]float64 // width, height in points
	Margins           [4]float64 // top, right, bottom, left
	BaseURL           string
	MediaType         string // "screen" or "print"
	UserStylesheet    string
	EnableJavaScript  bool
	UseTailwind       bool
	// TemplateData, when non-nil, causes the HTML to be processed as a fasttpl
	// template before parsing. Supports {{ if }}, {{ range }}, {{ filters }},
	// nested keys like {{ user.name }}, etc.
	TemplateData map[string]any
}

// PageConfig holds page configuration.
type PageConfig struct {
	Width, Height float64
	Margins       [4]float64
}

// ConvertResult holds the conversion results.
type ConvertResult struct {
	Elements []layout.Element
	Config   PageConfig
	Metadata map[string]string
}

// Convert converts an HTML string to layout elements.
func Convert(htmlString string, opts Options) (*ConvertResult, error) {
	if opts.DefaultFontSize <= 0 {
		opts.DefaultFontSize = 12
	}
	if opts.DefaultFontFamily == "" {
		opts.DefaultFontFamily = "Helvetica"
	}
	if opts.PageSize == [2]float64{} {
		opts.PageSize = [2]float64{595.28, 841.89} // A4
	}
	if opts.Margins == [4]float64{} {
		opts.Margins = [4]float64{72, 72, 72, 72} // 1 inch
	}
	if opts.MediaType == "" {
		opts.MediaType = "print"
	}

	// Process fasttpl template if TemplateData is provided
	if opts.TemplateData != nil && strings.Contains(htmlString, "{{") {
		tpl, err := fasttpl.Compile(htmlString)
		if err != nil {
			return nil, fmt.Errorf("compiling template: %w", err)
		}
		htmlString, err = tpl.RenderString(opts.TemplateData)
		if err != nil {
			return nil, fmt.Errorf("rendering template: %w", err)
		}
	}

	// Parse HTML
	dom, err := ParseHTML(strings.NewReader(htmlString))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	c := &converter{
		opts:         opts,
		rootFontSize: opts.DefaultFontSize,
		fetcher:      NewFetcher(opts.BaseURL),
		counters:     NewCounterState(),
	}

	// Execute JavaScript if enabled (before CSS collection so Alpine bindings are resolved)
	if opts.EnableJavaScript {
		if err := ExecuteScripts(dom, c.fetcher); err != nil {
			// Log warning but don't fail — partial JS is OK
			log.Printf("Warning: JavaScript execution: %v", err)
		}
	}

	// Auto-detect Tailwind CDN or use explicit flag
	useTailwind := opts.UseTailwind

	// Collect stylesheets
	var allCSS strings.Builder

	// Default user-agent stylesheet
	allCSS.WriteString(defaultStylesheet)
	allCSS.WriteString("\n")

	// Extract <style> and <link rel="stylesheet"> from <head>
	head := dom.GetHead()
	if head != nil {
		for _, child := range head.Children {
			if child.Tag == "style" {
				allCSS.WriteString(child.TextContent())
				allCSS.WriteString("\n")
			}
			if child.Tag == "link" && child.GetAttribute("rel") == "stylesheet" {
				href := child.GetAttribute("href")
				if href != "" {
					// Skip CDN links that aren't actual stylesheets
					if strings.Contains(href, "cdn.tailwindcss.com") {
						useTailwind = true
						continue
					}
					css, fetchErr := c.fetcher.FetchCSS(href)
					if fetchErr == nil {
						allCSS.WriteString(css)
						allCSS.WriteString("\n")
					}
				}
			}
			// Skip JS CDN scripts (Tailwind, Alpine, etc.)
			if child.Tag == "script" {
				src := child.GetAttribute("src")
				if strings.Contains(src, "cdn.tailwindcss.com") {
					useTailwind = true
					continue
				}
			}
		}
	}

	// Also check body for inline <style> tags
	body := dom.GetBody()
	if body != nil {
		for _, child := range body.Children {
			if child.Tag == "style" {
				allCSS.WriteString(child.TextContent())
				allCSS.WriteString("\n")
			}
		}
	}

	// User stylesheet
	if opts.UserStylesheet != "" {
		allCSS.WriteString(opts.UserStylesheet)
		allCSS.WriteString("\n")
	}

	// Parse all CSS
	stylesheet, err := ParseCSS(allCSS.String())
	if err != nil {
		return nil, fmt.Errorf("parsing CSS: %w", err)
	}
	c.stylesheet = stylesheet

	// Initialize Tailwind parser if needed
	if useTailwind {
		c.tailwindParser = tailwind.New()
	}

	// Extract metadata
	metadata := make(map[string]string)
	if head != nil {
		for _, child := range head.Children {
			if child.Tag == "title" {
				metadata["title"] = child.TextContent()
			}
			if child.Tag == "meta" {
				name := child.GetAttribute("name")
				content := child.GetAttribute("content")
				if name != "" && content != "" {
					metadata[name] = content
				}
			}
		}
	}

	// Apply styles to DOM
	rootStyle := NewDefaultStyle()
	rootStyle.FontSize = opts.DefaultFontSize
	rootStyle.FontFamily = opts.DefaultFontFamily
	c.applyStyles(dom, rootStyle)

	// Handle page rules
	pageConfig := PageConfig{
		Width:   opts.PageSize[0],
		Height:  opts.PageSize[1],
		Margins: opts.Margins,
	}
	if pageRule := parsePageRules(stylesheet); pageRule != nil {
		if pageRule.Size[0] > 0 {
			pageConfig.Width = pageRule.Size[0]
		}
		if pageRule.Size[1] > 0 {
			pageConfig.Height = pageRule.Size[1]
		}
		if pageRule.Margins[0] > 0 || pageRule.Margins[1] > 0 || pageRule.Margins[2] > 0 || pageRule.Margins[3] > 0 {
			pageConfig.Margins = pageRule.Margins
		}
	}

	// Convert body to elements
	target := body
	if target == nil {
		target = dom
	}

	var elements []layout.Element
	for _, child := range target.Children {
		if el := c.convertNode(child); el != nil {
			elements = append(elements, el)
		}
	}

	return &ConvertResult{
		Elements: elements,
		Config:   pageConfig,
		Metadata: metadata,
	}, nil
}

type converter struct {
	opts            Options
	rootFontSize    float64
	stylesheet      *Stylesheet
	fetcher         *Fetcher
	counters        *CounterState
	tailwindParser  *tailwind.Parser
}

// appliedRule pairs a rule with its order for cascade sorting.
type appliedRule struct {
	properties  map[string]CSSValue
	specificity [4]int
	order       int
	important   bool
}

func (c *converter) applyStyles(node *Node, parentStyle *ComputedStyle) {
	if node.IsText() {
		node.Style = InheritStyle(parentStyle)
		return
	}

	// Start with inherited style
	style := InheritStyle(parentStyle)

	// Apply tag-based default display
	applyTagDefaults(node, style)

	// Apply Tailwind classes (before CSS rules, so document CSS can override)
	if c.tailwindParser != nil && len(node.Classes) > 0 {
		classes := strings.Join(node.Classes, " ")
		twProps := c.tailwindParser.Parse(classes)
		if len(twProps) > 0 {
			// For PDF, apply responsive variants up to the page width.
			// A4 is ~795px, so sm (640), md (768) apply; lg (1024) does not.
			pageWidthPx := c.opts.PageSize[0] * 96 / 72 // points to px
			activeVariants := []string{":variant:sm", ":variant:md"}
			if pageWidthPx >= 1024 {
				activeVariants = append(activeVariants, ":variant:lg")
			}
			if pageWidthPx >= 1280 {
				activeVariants = append(activeVariants, ":variant:xl")
			}
			// Merge active variant props into base props
			for _, vk := range activeVariants {
				if vprops, ok := twProps[vk]; ok {
					for k, v := range vprops.(map[string]any) {
						twProps[k] = v
					}
				}
			}

			cssProps := make(map[string]CSSValue)
			// Collect CSS custom properties first for var() resolution
			customProps := make(map[string]string)
			for k, v := range twProps {
				if strings.HasPrefix(k, "--") {
					customProps[k] = fmt.Sprintf("%v", v)
				}
			}
			for k, v := range twProps {
				// Skip variant-prefixed and CSS custom properties
				if strings.HasPrefix(k, ":variant:") || strings.HasPrefix(k, "--") {
					continue
				}
				vs := fmt.Sprintf("%v", v)
				// Resolve var() references using custom properties
				if strings.Contains(vs, "var(") {
					vs = resolveVarReferences(vs, customProps)
					// If still has unresolvable var(), skip
					if strings.Contains(vs, "var(") {
						continue
					}
				}
				// Resolve calc() expressions with simple arithmetic
				if strings.Contains(vs, "calc(") {
					vs = resolveCalcExpressions(vs)
				}
				val := CSSValue{Value: vs}
				// Expand shorthand properties
				switch k {
				case "padding":
					expandBoxShorthand("padding", val, cssProps)
				case "margin":
					expandBoxShorthand("margin", val, cssProps)
				case "border-width":
					expandBoxShorthand("border", val, cssProps)
					// Rename to border-*-width
					for _, side := range []string{"top", "right", "bottom", "left"} {
						if sv, ok := cssProps["border-"+side]; ok {
							cssProps["border-"+side+"-width"] = sv
							delete(cssProps, "border-"+side)
						}
					}
				case "border-color":
					for _, side := range []string{"top", "right", "bottom", "left"} {
						cssProps["border-"+side+"-color"] = val
					}
				case "border-style":
					for _, side := range []string{"top", "right", "bottom", "left"} {
						cssProps["border-"+side+"-style"] = val
					}
				default:
					cssProps[k] = val
				}
			}
			style.Apply(cssProps, parentStyle, c.rootFontSize)
		}
	}

	// Collect matching rules
	var rules []appliedRule
	for i, rule := range c.stylesheet.Rules {
		for _, sel := range rule.Selectors {
			if sel.Matches(node) {
				spec := Specificity(&sel)
				rules = append(rules, appliedRule{
					properties:  rule.Properties,
					specificity: spec,
					order:       i,
				})
				break
			}
		}
	}

	// Sort by specificity then order
	sort.SliceStable(rules, func(i, j int) bool {
		cmp := CompareSpecificity(rules[i].specificity, rules[j].specificity)
		if cmp != 0 {
			return cmp < 0
		}
		return rules[i].order < rules[j].order
	})

	// Apply rules (non-important first, then important)
	for _, rule := range rules {
		nonImportant := make(map[string]CSSValue)
		for k, v := range rule.properties {
			if !v.Priority {
				nonImportant[k] = v
			}
		}
		style.Apply(nonImportant, parentStyle, c.rootFontSize)
	}
	for _, rule := range rules {
		important := make(map[string]CSSValue)
		for k, v := range rule.properties {
			if v.Priority {
				important[k] = v
			}
		}
		if len(important) > 0 {
			style.Apply(important, parentStyle, c.rootFontSize)
		}
	}

	// Apply inline styles last (highest priority)
	if inlineStyle := node.GetAttribute("style"); inlineStyle != "" {
		props := ParseInlineStyle(inlineStyle)
		style.Apply(props, parentStyle, c.rootFontSize)
	}

	node.Style = style

	// Handle counters
	if style.CounterReset != "" {
		parts := strings.Fields(style.CounterReset)
		for i := 0; i < len(parts); i++ {
			name := parts[i]
			val := 0
			if i+1 < len(parts) {
				if v, err := fmt.Sscanf(parts[i+1], "%d", &val); err == nil && v == 1 {
					i++
				}
			}
			c.counters.Reset(name, val)
		}
	}
	if style.CounterIncrement != "" {
		parts := strings.Fields(style.CounterIncrement)
		for i := 0; i < len(parts); i++ {
			name := parts[i]
			val := 1
			if i+1 < len(parts) {
				if v, err := fmt.Sscanf(parts[i+1], "%d", &val); err == nil && v == 1 {
					i++
				}
			}
			c.counters.Increment(name, val)
		}
	}

	// Recurse children
	for _, child := range node.Children {
		c.applyStyles(child, style)
	}
}

func applyTagDefaults(node *Node, style *ComputedStyle) {
	switch node.Tag {
	case "div", "section", "article", "main", "header", "footer", "nav", "aside",
		"figure", "figcaption", "blockquote", "address", "details", "summary":
		style.Display = "block"
	case "span", "mark", "small", "sub", "sup",
		"kbd", "samp", "var", "abbr", "cite", "dfn", "q", "time":
		style.Display = "inline"
	case "strong", "b":
		style.Display = "inline"
		style.FontWeight = 700
	case "em", "i":
		style.Display = "inline"
		style.FontStyle = "italic"
	case "u":
		style.Display = "inline"
		style.TextDecoration = "underline"
	case "s", "del":
		style.Display = "inline"
		style.TextDecoration = "line-through"
	case "code":
		style.Display = "inline"
		style.FontFamily = "Courier"
	case "p":
		style.Display = "block"
	case "h1":
		style.Display = "block"
		style.FontSize = 24
		style.FontWeight = 700
	case "h2":
		style.Display = "block"
		style.FontSize = 20
		style.FontWeight = 700
	case "h3":
		style.Display = "block"
		style.FontSize = 16
		style.FontWeight = 700
	case "h4":
		style.Display = "block"
		style.FontSize = 14
		style.FontWeight = 700
	case "h5":
		style.Display = "block"
		style.FontSize = 12
		style.FontWeight = 700
	case "h6":
		style.Display = "block"
		style.FontSize = 10
		style.FontWeight = 700
	case "a":
		style.Display = "inline"
		style.Color = [3]float64{0.067, 0.333, 0.800}
		style.TextDecoration = "underline"
	case "ul", "ol":
		style.Display = "block"
	case "li":
		style.Display = "list-item"
	case "table":
		style.Display = "table"
	case "tr":
		style.Display = "table-row"
	case "td", "th":
		style.Display = "table-cell"
	case "thead":
		style.Display = "table-header-group"
	case "tbody":
		style.Display = "table-row-group"
	case "tfoot":
		style.Display = "table-footer-group"
	case "pre":
		style.Display = "block"
		style.FontFamily = "Courier"
		style.WhiteSpace = "pre"
	case "img":
		style.Display = "inline-block"
	case "br":
		style.Display = "inline"
	case "hr":
		style.Display = "block"
	case "input", "textarea", "select", "button":
		style.Display = "inline-block"
	}
}

func (c *converter) convertNode(node *Node) layout.Element {
	if node == nil {
		return nil
	}

	style := node.Style
	if style == nil {
		style = NewDefaultStyle()
	}

	if style.Display == "none" || style.Visibility == "hidden" {
		return nil
	}

	// Skip pure whitespace text
	if node.IsText() {
		text := node.Text
		if strings.TrimSpace(text) == "" {
			return nil
		}
		return nil // Text nodes are collected by parent
	}

	switch node.Tag {
	case "style", "script", "link", "meta", "title", "head":
		return nil
	case "br":
		return nil // Handled inline by paragraph builder
	}

	switch style.Display {
	case "flex":
		return convertFlexContainer(node, style, c)
	case "grid":
		return convertGridContainer(node, style, c)
	case "table":
		return c.convertTable(node)
	}

	switch node.Tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return c.convertHeading(node)
	case "p":
		return c.convertParagraph(node)
	case "ul", "ol":
		return c.convertList(node)
	case "table":
		return c.convertTable(node)
	case "img":
		return c.convertImage(node)
	case "hr":
		return c.convertHR(node)
	case "blockquote":
		return c.convertBlockquote(node)
	case "pre":
		return c.convertPre(node)
	}

	// Default: treat as div container
	return c.convertDiv(node)
}

func (c *converter) convertHeading(node *Node) layout.Element {
	level := 1
	if len(node.Tag) == 2 && node.Tag[0] == 'h' {
		level = int(node.Tag[1] - '0')
	}
	runs := c.collectTextRuns(node)
	return &HeadingElement{
		Level:    level,
		Runs:     runs,
		Style:    node.Style,
		BoxModel: c.computeBoxModel(node.Style),
	}
}

func (c *converter) convertParagraph(node *Node) layout.Element {
	runs := c.collectTextRuns(node)
	if len(runs) == 0 {
		// Check for block children
		return c.convertDiv(node)
	}
	return &ParagraphElement{
		Runs:     runs,
		Style:    node.Style,
		BoxModel: c.computeBoxModel(node.Style),
	}
}

func (c *converter) convertDiv(node *Node) layout.Element {
	var children []layout.Element

	// Collect inline runs and block elements
	var currentRuns []layout.TextRun
	for _, child := range node.Children {
		if child.IsText() {
			text := child.Text
			if strings.TrimSpace(text) == "" {
				continue
			}
			runs := c.textRunsFromText(child)
			currentRuns = append(currentRuns, runs...)
			continue
		}

		childStyle := child.Style
		if childStyle == nil {
			childStyle = NewDefaultStyle()
		}

		if child.Tag == "img" {
			// Replaced elements should stay as elements, even when inline/inline-block.
			if len(currentRuns) > 0 {
				children = append(children, &ParagraphElement{
					Runs:  currentRuns,
					Style: node.Style,
				})
				currentRuns = nil
			}
			if el := c.convertNode(child); el != nil {
				children = append(children, el)
			}
			continue
		}

		if isInlineTag(child.Tag) || childStyle.Display == "inline" || childStyle.Display == "inline-block" {
			runs := c.collectTextRuns(child)
			currentRuns = append(currentRuns, runs...)
			continue
		}

		// Flush inline content as paragraph
		if len(currentRuns) > 0 {
			children = append(children, &ParagraphElement{
				Runs:  currentRuns,
				Style: node.Style,
			})
			currentRuns = nil
		}

		if el := c.convertNode(child); el != nil {
			children = append(children, el)
		}
	}

	// Flush remaining inline content
	if len(currentRuns) > 0 {
		children = append(children, &ParagraphElement{
			Runs:  currentRuns,
			Style: node.Style,
		})
	}

	if len(children) == 0 {
		return nil
	}

	return &DivElement{
		Children: children,
		Style:    node.Style,
		BoxModel: c.computeBoxModel(node.Style),
	}
}

func (c *converter) convertList(node *Node) layout.Element {
	ordered := node.Tag == "ol"
	var items []ListItem
	itemNum := 1

	for _, child := range node.Children {
		if child.Tag != "li" {
			continue
		}
		runs := c.collectTextRuns(child)
		var subChildren []layout.Element
		for _, grandchild := range child.Children {
			if grandchild.Tag == "ul" || grandchild.Tag == "ol" {
				if el := c.convertNode(grandchild); el != nil {
					subChildren = append(subChildren, el)
				}
			}
		}
		marker := "\u2022 " // bullet
		if ordered {
			marker = fmt.Sprintf("%d. ", itemNum)
			itemNum++
		}
		items = append(items, ListItem{
			Marker:   marker,
			Runs:     runs,
			Children: subChildren,
		})
	}

	return &ListElement{
		Items:    items,
		Ordered:  ordered,
		Style:    node.Style,
		BoxModel: c.computeBoxModel(node.Style),
	}
}

func (c *converter) convertTable(node *Node) layout.Element {
	var rows []TableRow
	var headerRows []TableRow

	// Collect rows from thead, tbody, tfoot, or direct tr children
	for _, child := range node.Children {
		switch child.Tag {
		case "thead":
			for _, tr := range child.Children {
				if tr.Tag == "tr" {
					row := c.convertTableRow(tr, true)
					// Inherit thead background color to row/cells if not set
					if child.Style != nil && child.Style.BackgroundColor != nil {
						if row.Style == nil || row.Style.BackgroundColor == nil {
							if row.Style == nil {
								row.Style = &ComputedStyle{}
							}
							row.Style.BackgroundColor = child.Style.BackgroundColor
						}
						for i := range row.Cells {
							if row.Cells[i].Style == nil || row.Cells[i].Style.BackgroundColor == nil {
								if row.Cells[i].Style == nil {
									row.Cells[i].Style = &ComputedStyle{}
								}
								row.Cells[i].Style.BackgroundColor = child.Style.BackgroundColor
							}
						}
					}
					// Inherit thead text color to cells if not set
					if child.Style != nil && (child.Style.Color != [3]float64{}) {
						for i := range row.Cells {
							if row.Cells[i].Style == nil {
								row.Cells[i].Style = &ComputedStyle{}
							}
							if row.Cells[i].Style.Color == ([3]float64{}) {
								row.Cells[i].Style.Color = child.Style.Color
							}
						}
					}
					headerRows = append(headerRows, row)
				}
			}
		case "tbody":
			for _, tr := range child.Children {
				if tr.Tag == "tr" {
					rows = append(rows, c.convertTableRow(tr, false))
				}
			}
		case "tfoot":
			for _, tr := range child.Children {
				if tr.Tag == "tr" {
					rows = append(rows, c.convertTableRow(tr, false))
				}
			}
		case "tr":
			rows = append(rows, c.convertTableRow(child, false))
		}
	}

	allRows := append(headerRows, rows...)

	return &TableElement{
		Rows:     allRows,
		Style:    node.Style,
		BoxModel: c.computeBoxModel(node.Style),
	}
}

func (c *converter) convertTableRow(tr *Node, isHeader bool) TableRow {
	var cells []TableCell
	for _, td := range tr.Children {
		if td.Tag != "td" && td.Tag != "th" {
			continue
		}
		runs := c.collectTextRuns(td)
		cells = append(cells, TableCell{
			Runs:     runs,
			IsHeader: td.Tag == "th" || isHeader,
			Style:    td.Style,
			Colspan:  parseIntAttr(td.GetAttribute("colspan"), 1),
			Rowspan:  parseIntAttr(td.GetAttribute("rowspan"), 1),
		})
	}
	return TableRow{Cells: cells, Style: tr.Style}
}

func (c *converter) convertImage(node *Node) layout.Element {
	src := node.GetAttribute("src")
	alt := node.GetAttribute("alt")
	if src == "" {
		return nil
	}

	return &ImageElement{
		Src:      src,
		Alt:      alt,
		Style:    node.Style,
		BoxModel: c.computeBoxModel(node.Style),
		Fetcher:  c.fetcher,
	}
}

func (c *converter) convertHR(node *Node) layout.Element {
	style := node.Style
	if style == nil {
		style = NewDefaultStyle()
	}
	return &HRElement{
		Style: style,
	}
}

func (c *converter) convertBlockquote(node *Node) layout.Element {
	div := c.convertDiv(node)
	if div == nil {
		return nil
	}
	// Wrap with blockquote styling
	bm := c.computeBoxModel(node.Style)
	bm.PaddingLeft += 15
	bm.BorderLeftWidth = 3
	bm.BorderColor = [3]float64{0.7, 0.7, 0.7}
	if d, ok := div.(*DivElement); ok {
		d.BoxModel = bm
		return d
	}
	return div
}

func (c *converter) convertPre(node *Node) layout.Element {
	text := node.TextContent()
	runs := []layout.TextRun{{
		Text:     text,
		FontName: "Courier",
		FontSize: node.Style.FontSize,
		Color:    node.Style.Color,
	}}
	bm := c.computeBoxModel(node.Style)
	bg := [3]float64{0.96, 0.96, 0.96}
	bm.Background = &bg
	if bm.PaddingTop == 0 {
		bm.PaddingTop = 8
		bm.PaddingRight = 8
		bm.PaddingBottom = 8
		bm.PaddingLeft = 8
	}
	return &ParagraphElement{
		Runs:     runs,
		Style:    node.Style,
		BoxModel: bm,
		PreWrap:  true,
	}
}

// collectTextRuns collects all inline text from a node tree.
func (c *converter) collectTextRuns(node *Node) []layout.TextRun {
	var runs []layout.TextRun
	c.collectTextRunsRecursive(node, &runs, true)
	return runs
}

func (c *converter) collectTextRunsRecursive(node *Node, runs *[]layout.TextRun, isRoot bool) {
	if node.IsText() {
		text := node.Text
		// Normalize whitespace: collapse runs of whitespace to single space (unless pre)
		isPreformatted := false
		if node.Parent != nil && node.Parent.Style != nil {
			ws := node.Parent.Style.WhiteSpace
			isPreformatted = ws == "pre" || ws == "pre-wrap" || ws == "pre-line"
		}
		if !isPreformatted {
			text = collapseWhitespace(text)
		}
		if strings.TrimSpace(text) == "" && text != " " {
			return
		}
		style := node.Style
		if style == nil {
			if node.Parent != nil {
				style = node.Parent.Style
			}
			if style == nil {
				style = NewDefaultStyle()
			}
		}
		text = applyTextTransform(text, style.TextTransform)
		*runs = append(*runs, layout.TextRun{
			Text:      text,
			FontName:  style.FontFamily,
			FontSize:  style.FontSize,
			Bold:      style.FontWeight >= 600,
			Italic:    style.FontStyle == "italic" || style.FontStyle == "oblique",
			Color:     style.Color,
			Underline: style.TextDecoration == "underline",
			Strike:    style.TextDecoration == "line-through",
		})
		return
	}

	if node.Tag == "br" {
		*runs = append(*runs, layout.TextRun{Text: "\n"})
		return
	}

	// Skip non-inline block elements — but not the root node we were asked to collect from
	// Also don't skip if we're inside a table cell (td/th) — collect all nested text
	if !isRoot && node.Style != nil && !isInlineDisplay(node.Style.Display) && node.Parent != nil {
		// Check if any ancestor is a table cell — if so, keep collecting
		isInCell := false
		for p := node.Parent; p != nil; p = p.Parent {
			if p.Tag == "td" || p.Tag == "th" {
				isInCell = true
				break
			}
		}
		if !isInCell {
			return
		}
	}

	// Handle link
	isLink := node.Tag == "a"
	href := ""
	if isLink {
		href = node.GetAttribute("href")
	}

	startIdx := len(*runs)
	for _, child := range node.Children {
		c.collectTextRunsRecursive(child, runs, false)
	}

	// Apply link to all runs added
	if isLink && href != "" {
		for i := startIdx; i < len(*runs); i++ {
			(*runs)[i].Link = href
		}
	}
}

func (c *converter) textRunsFromText(node *Node) []layout.TextRun {
	style := node.Style
	if style == nil {
		if node.Parent != nil {
			style = node.Parent.Style
		}
		if style == nil {
			style = NewDefaultStyle()
		}
	}
	text := node.Text
	isPreformatted := false
	if style.WhiteSpace == "pre" || style.WhiteSpace == "pre-wrap" || style.WhiteSpace == "pre-line" {
		isPreformatted = true
	}
	if !isPreformatted {
		text = collapseWhitespace(text)
	}
	text = applyTextTransform(text, style.TextTransform)
	return []layout.TextRun{{
		Text:     text,
		FontName: style.FontFamily,
		FontSize: style.FontSize,
		Bold:     style.FontWeight >= 700,
		Italic:   style.FontStyle == "italic",
		Color:    style.Color,
	}}
}

func (c *converter) computeBoxModel(style *ComputedStyle) layout.BoxModel {
	if style == nil {
		return layout.BoxModel{}
	}
	fs := style.FontSize
	rfs := c.rootFontSize
	return layout.BoxModel{
		MarginTop:         style.MarginTop.ToPoints(fs, rfs),
		MarginRight:       style.MarginRight.ToPoints(fs, rfs),
		MarginBottom:      style.MarginBottom.ToPoints(fs, rfs),
		MarginLeft:        style.MarginLeft.ToPoints(fs, rfs),
		PaddingTop:        style.PaddingTop.ToPoints(fs, rfs),
		PaddingRight:      style.PaddingRight.ToPoints(fs, rfs),
		PaddingBottom:     style.PaddingBottom.ToPoints(fs, rfs),
		PaddingLeft:       style.PaddingLeft.ToPoints(fs, rfs),
		BorderTopWidth:    style.BorderTopWidth,
		BorderRightWidth:  style.BorderRightWidth,
		BorderBottomWidth: style.BorderBottomWidth,
		BorderLeftWidth:   style.BorderLeftWidth,
		BorderColor:       style.BorderTopColor,
		Background:        style.BackgroundColor,
		BorderRadius:      style.BorderRadius,
	}
}

func isInlineTag(tag string) bool {
	switch tag {
	case "span", "strong", "b", "em", "i", "u", "s", "mark", "small", "sub", "sup",
		"code", "kbd", "samp", "var", "a", "abbr", "cite", "dfn", "q", "time",
		"del", "ins", "br":
		return true
	}
	return false
}

func isInlineDisplay(display string) bool {
	return display == "inline" || display == "inline-block"
}

func parseIntAttr(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := fmt.Sscanf(s, "%d", new(int))
	if err != nil || v == 0 {
		return def
	}
	n := 0
	fmt.Sscanf(s, "%d", &n)
	if n <= 0 {
		return def
	}
	return n
}

// collapseWhitespace collapses sequences of whitespace (spaces, tabs, newlines)
// into single spaces, matching CSS normal whitespace behavior.
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			if !inSpace {
				b.WriteByte(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	return b.String()
}

// Default user-agent stylesheet for beautiful output.
const defaultStylesheet = `
body {
	font-family: Helvetica, Arial, sans-serif;
	font-size: 12pt;
	line-height: 1.5;
	color: #333333;
}
h1 { font-size: 24pt; font-weight: bold; margin-top: 18pt; margin-bottom: 12pt; }
h2 { font-size: 20pt; font-weight: bold; margin-top: 16pt; margin-bottom: 10pt; }
h3 { font-size: 16pt; font-weight: bold; margin-top: 14pt; margin-bottom: 8pt; }
h4 { font-size: 14pt; font-weight: bold; margin-top: 12pt; margin-bottom: 6pt; }
h5 { font-size: 12pt; font-weight: bold; margin-top: 10pt; margin-bottom: 4pt; }
h6 { font-size: 10pt; font-weight: bold; margin-top: 8pt; margin-bottom: 4pt; }
p { margin-top: 0; margin-bottom: 8pt; }
ul, ol { margin-top: 0; margin-bottom: 8pt; padding-left: 30pt; }
li { margin-bottom: 3pt; }
table { border-collapse: collapse; margin-top: 8pt; margin-bottom: 8pt; width: 100%; }
th { font-weight: bold; padding: 6pt 8pt; border-bottom: 2px solid #333; }
td { padding: 4pt 8pt; border-bottom: 1px solid #ddd; }
blockquote { margin: 8pt 0; padding: 8pt 15pt; border-left: 3pt solid #ccc; color: #666; font-style: italic; }
pre { background-color: #f5f5f5; padding: 8pt; margin: 8pt 0; font-family: Courier, monospace; font-size: 10pt; }
code { font-family: Courier, monospace; font-size: 10pt; background-color: #f5f5f5; padding: 1pt 3pt; }
a { color: #1155cc; text-decoration: underline; }
hr { border: none; border-top: 1px solid #ccc; margin: 12pt 0; }
img { max-width: 100%; }
mark { background-color: #ffff00; }
small { font-size: 0.85em; }
` + "\n"

// resolveVarReferences replaces var(--name) with values from customProps.
func resolveVarReferences(value string, customProps map[string]string) string {
	for i := 0; i < 10; i++ { // iterate to resolve nested vars
		idx := strings.Index(value, "var(")
		if idx < 0 {
			break
		}
		// Find matching closing paren
		depth := 0
		end := idx
		for j := idx; j < len(value); j++ {
			if value[j] == '(' {
				depth++
			} else if value[j] == ')' {
				depth--
				if depth == 0 {
					end = j
					break
				}
			}
		}
		varExpr := value[idx+4 : end] // inside var(...)
		// Handle var(--name, fallback)
		parts := strings.SplitN(varExpr, ",", 2)
		varName := strings.TrimSpace(parts[0])
		resolved, ok := customProps[varName]
		if !ok {
			if len(parts) > 1 {
				resolved = strings.TrimSpace(parts[1])
			} else {
				break // can't resolve
			}
		}
		value = value[:idx] + resolved + value[end+1:]
	}
	return value
}

// resolveCalcExpressions attempts to simplify calc() expressions with
// simple multiplication/addition of a numeric value and a unit.
func resolveCalcExpressions(value string) string {
	for {
		idx := strings.LastIndex(value, "calc(")
		if idx < 0 {
			break
		}
		// Find matching closing paren after "calc("
		start := idx + 5
		depth := 1
		end := -1
		for j := start; j < len(value); j++ {
			if value[j] == '(' {
				depth++
			} else if value[j] == ')' {
				depth--
				if depth == 0 {
					end = j
					break
				}
			}
		}
		if end < 0 || end <= start {
			break
		}
		inner := strings.TrimSpace(value[start:end])

		// Try to evaluate: "Xunit * Y" or "Y * Xunit" or "Xunit + Yunit"
		result := tryEvalCalc(inner)
		if result != "" {
			value = value[:idx] + result + value[end+1:]
		} else {
			// Can't simplify, leave it
			break
		}
	}
	return value
}

func tryEvalCalc(expr string) string {
	expr = strings.TrimSpace(expr)

	// Handle nested calc results: "0.25rem * 1" or "0.25rem * 0"
	// Pattern: <value><unit> * <number>
	for _, op := range []string{" * ", " + ", " - "} {
		parts := strings.SplitN(expr, op, 2)
		if len(parts) != 2 {
			continue
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])

		lVal, lUnit := parseValueUnit(left)
		rVal, rUnit := parseValueUnit(right)

		if lUnit != "" && rUnit == "" {
			// e.g., "0.25rem * 1"
			switch op {
			case " * ":
				return fmt.Sprintf("%g%s", lVal*rVal, lUnit)
			case " + ":
				return fmt.Sprintf("%g%s", lVal+rVal, lUnit)
			case " - ":
				return fmt.Sprintf("%g%s", lVal-rVal, lUnit)
			}
		}
		if lUnit == "" && rUnit != "" {
			switch op {
			case " * ":
				return fmt.Sprintf("%g%s", lVal*rVal, rUnit)
			case " + ":
				return fmt.Sprintf("%g%s", lVal+rVal, rUnit)
			case " - ":
				return fmt.Sprintf("%g%s", lVal-rVal, rUnit)
			}
		}
		if lUnit != "" && lUnit == rUnit {
			switch op {
			case " + ":
				return fmt.Sprintf("%g%s", lVal+rVal, lUnit)
			case " - ":
				return fmt.Sprintf("%g%s", lVal-rVal, lUnit)
			}
		}
	}
	return ""
}

func parseValueUnit(s string) (float64, string) {
	s = strings.TrimSpace(s)
	// Try common units
	for _, unit := range []string{"rem", "em", "px", "pt", "%", "vw", "vh"} {
		if strings.HasSuffix(s, unit) {
			numStr := strings.TrimSpace(s[:len(s)-len(unit)])
			var v float64
			if _, err := fmt.Sscanf(numStr, "%g", &v); err == nil {
				return v, unit
			}
		}
	}
	// Pure number
	var v float64
	if _, err := fmt.Sscanf(s, "%g", &v); err == nil {
		return v, ""
	}
	return 0, ""
}
