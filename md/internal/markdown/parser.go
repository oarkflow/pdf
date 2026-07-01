package markdown

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type Parser struct{ opt Options }

type sourceLine struct {
	Text string
	No   int
}

func NewParser(opt Options) *Parser {
	if opt.MaxLineBytes <= 0 {
		opt.MaxLineBytes = 4 << 20
	}
	return &Parser{opt: opt}
}

func (p *Parser) Parse(r io.Reader) (Doc, error) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), p.opt.MaxLineBytes)
	var lines []sourceLine
	ln := 0
	for s.Scan() {
		ln++
		lines = append(lines, sourceLine{Text: strings.TrimRight(s.Text(), "\r"), No: ln})
	}
	if err := s.Err(); err != nil {
		return Doc{}, err
	}
	d := Doc{Meta: map[string]string{}, Refs: map[string]LinkRef{}}
	start := parseFrontMatterSL(lines, d.Meta)
	body := make([]sourceLine, 0, len(lines)-start)
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if id, ref, ok := linkDefinition(line.Text); ok {
			d.Refs[strings.ToLower(id)] = ref
			continue
		}
		if id, txt, ok := footnoteDefinition(line.Text); ok {
			d.Footnotes = append(d.Footnotes, Footnote{ID: id, Text: txt})
			continue
		}
		body = append(body, line)
	}
	d.Nodes = parseBlocks(body, &d, 0)
	if len(d.Footnotes) > 0 {
		var fn Node
		fn.Kind = FootnoteList
		fn.Start = Position{Line: d.Footnotes[0].NodesLine()}
		for _, f := range d.Footnotes {
			text := f.Text
			children := f.Nodes
			if len(children) == 0 && text != "" {
				children = []Node{{Kind: Paragraph, Text: text}}
			}
			fn.Children = append(fn.Children, Node{Kind: FootnoteDefinition, ID: f.ID, Text: text, Children: children})
		}
		d.Nodes = append(d.Nodes, fn)
	}
	return d, nil
}

func (f Footnote) NodesLine() int { return 0 }

func parseBlocks(lines []sourceLine, d *Doc, baseIndent int) []Node {
	var nodes []Node
	flushPara := func(buf *[]sourceLine) {
		if len(*buf) == 0 {
			return
		}
		textParts := make([]string, 0, len(*buf))
		for _, l := range *buf {
			textParts = append(textParts, strings.TrimSpace(l.Text))
		}
		text := strings.Join(textParts, " ")
		nodes = append(nodes, Node{Kind: Paragraph, Text: text, Start: pos((*buf)[0], 1), End: pos((*buf)[len(*buf)-1], len((*buf)[len(*buf)-1].Text)+1)})
		*buf = (*buf)[:0]
	}
	var para []sourceLine
	for i := 0; i < len(lines); {
		line := lines[i]
		trim := strings.TrimSpace(line.Text)
		indent := leadingSpaces(line.Text)
		if trim == "" {
			flushPara(&para)
			i++
			continue
		}
		if indent < baseIndent && baseIndent > 0 {
			break
		}
		// Fenced code blocks.
		if fence, info, ok := codeFence(trim); ok {
			flushPara(&para)
			var code bytes.Buffer
			start := line
			i++
			for i < len(lines) {
				if strings.HasPrefix(strings.TrimSpace(lines[i].Text), fence) {
					i++
					break
				}
				code.WriteString(lines[i].Text)
				code.WriteByte('\n')
				i++
			}
			nodes = append(nodes, Node{Kind: CodeBlock, Info: info, Text: strings.TrimRight(code.String(), "\n"), Start: pos(start, indent+1)})
			continue
		}
		// Indented code block.
		if indent >= baseIndent+4 {
			flushPara(&para)
			start := line
			var code bytes.Buffer
			for i < len(lines) {
				if strings.TrimSpace(lines[i].Text) == "" {
					code.WriteByte('\n')
					i++
					continue
				}
				if leadingSpaces(lines[i].Text) < baseIndent+4 {
					break
				}
				code.WriteString(removeIndent(lines[i].Text, baseIndent+4))
				code.WriteByte('\n')
				i++
			}
			nodes = append(nodes, Node{Kind: CodeBlock, Text: strings.TrimRight(code.String(), "\n"), Start: pos(start, indent+1)})
			continue
		}
		// HTML block (safe parsing; renderers decide whether to emit raw HTML).
		if isHTMLBlockStart(trim) {
			flushPara(&para)
			start := line
			var b strings.Builder
			for i < len(lines) {
				if strings.TrimSpace(lines[i].Text) == "" && b.Len() > 0 {
					break
				}
				b.WriteString(lines[i].Text)
				b.WriteByte('\n')
				i++
				if strings.Contains(lines[i-1].Text, ">") && strings.HasPrefix(trim, "<!--") && strings.Contains(lines[i-1].Text, "-->") {
					break
				}
			}
			nodes = append(nodes, Node{Kind: HTMLBlock, Text: strings.TrimRight(b.String(), "\n"), Start: pos(start, indent+1)})
			continue
		}
		// Tables: require header row followed by delimiter row.
		if i+1 < len(lines) {
			if header, ok := tableRow(line.Text); ok {
				if al, ok := tableSeparator(lines[i+1].Text); ok {
					flushPara(&para)
					rows := [][]string{header}
					i += 2
					for i < len(lines) {
						if row, ok := tableRow(lines[i].Text); ok {
							rows = append(rows, row)
							i++
							continue
						}
						break
					}
					nodes = append(nodes, Node{Kind: Table, Rows: rows, Aligns: normalizeAligns(al, len(header)), Start: pos(line, indent+1)})
					continue
				}
			}
		}
		if isHR(trim) {
			flushPara(&para)
			nodes = append(nodes, Node{Kind: HorizontalRule, Start: pos(line, indent+1)})
			i++
			continue
		}
		if lvl, txt, id, ok := heading(trim); ok {
			flushPara(&para)
			if id == "" {
				id = uniqueSlug(txt, d.Headings)
			}
			n := Node{Kind: Heading, Level: lvl, Text: txt, ID: id, Start: pos(line, indent+1)}
			nodes = append(nodes, n)
			d.Headings = append(d.Headings, HeadingRef{Level: lvl, Text: txt, ID: id})
			i++
			continue
		}
		// Setext heading: paragraph line followed by === or ---.
		if i+1 < len(lines) && len(para) == 0 {
			if lvl, ok := setext(lines[i+1].Text); ok && trim != "" {
				text := strings.TrimSpace(line.Text)
				id := uniqueSlug(text, d.Headings)
				nodes = append(nodes, Node{Kind: Heading, Level: lvl, Text: text, ID: id, Start: pos(line, indent+1)})
				d.Headings = append(d.Headings, HeadingRef{Level: lvl, Text: text, ID: id})
				i += 2
				continue
			}
		}
		if alertKind, alertText, ok := alert(line.Text); ok {
			flushPara(&para)
			body, next := collectQuote(lines, i)
			if alertText != "" && len(body) > 0 {
				body[0].Text = strings.TrimSpace(alertText)
			}
			children := parseBlocks(body, d, 0)
			text := plainTextNodes(children)
			nodes = append(nodes, Node{Kind: Alert, Info: strings.ToLower(alertKind), Text: text, Children: children, Start: pos(line, indent+1)})
			i = next
			continue
		}
		if strings.HasPrefix(trim, ">") {
			flushPara(&para)
			body, next := collectQuote(lines, i)
			children := parseBlocks(body, d, 0)
			nodes = append(nodes, Node{Kind: BlockQuote, Text: plainTextNodes(children), Children: children, Start: pos(line, indent+1)})
			i = next
			continue
		}
		if _, _, _, ok := listMarker(line.Text, baseIndent); ok {
			flushPara(&para)
			list, next := parseList(lines, i, d, baseIndent)
			nodes = append(nodes, list)
			i = next
			continue
		}
		// Definition list: Term\n: definition
		if i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1].Text), ": ") {
			flushPara(&para)
			term := strings.TrimSpace(line.Text)
			def := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i+1].Text), ":"))
			nodes = append(nodes, Node{Kind: DefinitionList, Children: []Node{{Kind: DefinitionTerm, Text: term}, {Kind: DefinitionItem, Text: def}}, Start: pos(line, indent+1)})
			i += 2
			continue
		}
		para = append(para, line)
		i++
	}
	flushPara(&para)
	return nodes
}

func parseList(lines []sourceLine, start int, d *Doc, baseIndent int) (Node, int) {
	_, ordered, _, _ := listMarker(lines[start].Text, baseIndent)
	kind := BulletList
	if ordered {
		kind = OrderedList
	}
	list := Node{Kind: kind, Start: pos(lines[start], baseIndent+1)}
	i := start
	for i < len(lines) {
		text, ord, checked, ok := listMarker(lines[i].Text, baseIndent)
		if !ok || ord != ordered {
			break
		}
		itemStart := lines[i]
		i++
		var childLines []sourceLine
		// The first line is stored as item text; subsequent indented continuation
		// lines become nested children so multi-paragraph and nested lists survive.
		for i < len(lines) {
			trim := strings.TrimSpace(lines[i].Text)
			ind := leadingSpaces(lines[i].Text)
			if trim == "" {
				if i+1 < len(lines) && leadingSpaces(lines[i+1].Text) > baseIndent {
					childLines = append(childLines, lines[i])
					i++
					continue
				}
				break
			}
			if _, _, _, ok := listMarker(lines[i].Text, baseIndent); ok {
				break
			}
			if ind <= baseIndent {
				break
			}
			childLines = append(childLines, sourceLine{Text: removeIndent(lines[i].Text, baseIndent+2), No: lines[i].No})
			i++
		}
		item := Node{Kind: ListItem, Text: text, Checked: checked, Start: pos(itemStart, baseIndent+1)}
		if len(childLines) > 0 {
			item.Children = parseBlocks(childLines, d, 0)
		}
		list.Children = append(list.Children, item)
		if i < len(lines) && strings.TrimSpace(lines[i].Text) == "" {
			i++
			if i >= len(lines) || leadingSpaces(lines[i].Text) < baseIndent || strings.TrimSpace(lines[i].Text) == "" {
				break
			}
		}
	}
	return list, i
}

func parseFrontMatterSL(lines []sourceLine, meta map[string]string) int {
	if len(lines) == 0 || strings.TrimSpace(lines[0].Text) != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i].Text)
		if trim == "---" {
			return i + 1
		}
		if k, v, ok := strings.Cut(lines[i].Text, ":"); ok {
			meta[strings.ToLower(strings.TrimSpace(k))] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	}
	return 0
}

var headingIDRe = regexp.MustCompile(`\s*\{#([A-Za-z0-9_-]+)\}\s*$`)

func heading(s string) (int, string, string, bool) {
	n := 0
	for n < len(s) && s[n] == '#' && n < 6 {
		n++
	}
	if n > 0 && n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		txt := strings.TrimSpace(strings.TrimRight(s[n+1:], "# "))
		id := ""
		if m := headingIDRe.FindStringSubmatch(txt); len(m) == 2 {
			id = m[1]
			txt = strings.TrimSpace(headingIDRe.ReplaceAllString(txt, ""))
		}
		return n, txt, id, true
	}
	return 0, "", "", false
}

func setext(s string) (int, bool) {
	trim := strings.TrimSpace(s)
	if len(trim) < 2 || strings.Contains(trim, "|") {
		return 0, false
	}
	ch := trim[0]
	if ch != '=' && ch != '-' {
		return 0, false
	}
	for _, r := range trim {
		if r != rune(ch) && r != ' ' {
			return 0, false
		}
	}
	if ch == '=' {
		return 1, true
	}
	return 2, true
}

func isHR(s string) bool {
	if len(s) < 3 || strings.Contains(s, "|") {
		return false
	}
	c := s[0]
	if c != '-' && c != '*' && c != '_' {
		return false
	}
	count := 0
	for _, r := range s {
		if r == rune(c) {
			count++
			continue
		}
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return count >= 3
}

func listMarker(s string, baseIndent int) (string, bool, *bool, bool) {
	ind := leadingSpaces(s)
	if ind < baseIndent || ind > baseIndent+3 {
		return "", false, nil, false
	}
	trim := strings.TrimLeft(s, " \t")
	if len(trim) > 2 && (trim[0] == '-' || trim[0] == '*' || trim[0] == '+') && (trim[1] == ' ' || trim[1] == '\t') {
		text := strings.TrimSpace(trim[2:])
		checked, text := parseTask(text)
		return text, false, checked, true
	}
	i := 0
	for i < len(trim) && trim[i] >= '0' && trim[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(trim) && (trim[i] == '.' || trim[i] == ')') && (trim[i+1] == ' ' || trim[i+1] == '\t') {
		return strings.TrimSpace(trim[i+2:]), true, nil, true
	}
	return "", false, nil, false
}

func parseTask(text string) (*bool, string) {
	if len(text) >= 3 && text[0] == '[' && text[2] == ']' && (text[1] == ' ' || text[1] == 'x' || text[1] == 'X') {
		v := text[1] == 'x' || text[1] == 'X'
		return &v, strings.TrimSpace(text[3:])
	}
	return nil, text
}

func alert(s string) (string, string, bool) {
	trim := strings.TrimSpace(s)
	if !strings.HasPrefix(trim, ">") {
		return "", "", false
	}
	body := strings.TrimSpace(strings.TrimPrefix(trim, ">"))
	if !strings.HasPrefix(body, "[!") {
		return "", "", false
	}
	end := strings.IndexByte(body, ']')
	if end < 3 {
		return "", "", false
	}
	kind := strings.TrimSpace(body[2:end])
	if kind == "" {
		return "", "", false
	}
	return kind, strings.TrimSpace(body[end+1:]), true
}

func collectQuote(lines []sourceLine, start int) ([]sourceLine, int) {
	var body []sourceLine
	i := start
	for i < len(lines) {
		trim := strings.TrimSpace(lines[i].Text)
		if trim == "" {
			body = append(body, sourceLine{Text: "", No: lines[i].No})
			i++
			continue
		}
		if !strings.HasPrefix(trim, ">") {
			break
		}
		unq := strings.TrimSpace(strings.TrimPrefix(trim, ">"))
		body = append(body, sourceLine{Text: unq, No: lines[i].No})
		i++
	}
	return body, i
}

func codeFence(trim string) (string, string, bool) {
	if strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") {
		ch := trim[0]
		n := 0
		for n < len(trim) && trim[n] == ch {
			n++
		}
		if n >= 3 {
			return strings.Repeat(string(ch), n), strings.TrimSpace(trim[n:]), true
		}
	}
	return "", "", false
}

func isHTMLBlockStart(s string) bool {
	low := strings.ToLower(s)
	return strings.HasPrefix(low, "<!--") || strings.HasPrefix(low, "<!doctype") || strings.HasPrefix(low, "<div") || strings.HasPrefix(low, "<section") || strings.HasPrefix(low, "<table") || strings.HasPrefix(low, "<pre") || strings.HasPrefix(low, "<script") || strings.HasPrefix(low, "<style") || strings.HasPrefix(low, "<hr") || strings.HasPrefix(low, "<p")
}

func splitPipe(s string) []string {
	t := strings.TrimSpace(s)
	if !strings.Contains(t, "|") {
		return nil
	}
	if strings.HasPrefix(t, "|") {
		t = t[1:]
	}
	if strings.HasSuffix(t, "|") {
		t = t[:len(t)-1]
	}
	parts := splitEscaped(t, '|')
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(strings.ReplaceAll(p, `\|`, "|"))
	}
	return cells
}
func splitEscaped(s string, sep rune) []string {
	var out []string
	var b strings.Builder
	esc := false
	for _, r := range s {
		if esc {
			if r != sep {
				b.WriteRune('\\')
			}
			b.WriteRune(r)
			esc = false
			continue
		}
		if r == '\\' {
			esc = true
			continue
		}
		if r == sep {
			out = append(out, b.String())
			b.Reset()
			continue
		}
		b.WriteRune(r)
	}
	if esc {
		b.WriteRune('\\')
	}
	out = append(out, b.String())
	return out
}
func tableRow(s string) ([]string, bool) {
	cells := splitPipe(s)
	if len(cells) < 2 {
		return nil, false
	}
	if _, ok := tableSeparator(s); ok {
		return nil, false
	}
	return cells, true
}
func tableSeparator(s string) ([]Align, bool) {
	cells := splitPipe(s)
	if len(cells) < 2 {
		return nil, false
	}
	aligns := make([]Align, len(cells))
	for i, c := range cells {
		c = strings.TrimSpace(c)
		if len(c) < 3 {
			return nil, false
		}
		dash := 0
		for _, r := range c {
			if r == '-' {
				dash++
				continue
			}
			if r != ':' {
				return nil, false
			}
		}
		if dash < 3 {
			return nil, false
		}
		left, right := strings.HasPrefix(c, ":"), strings.HasSuffix(c, ":")
		switch {
		case left && right:
			aligns[i] = AlignCenter
		case right:
			aligns[i] = AlignRight
		default:
			aligns[i] = AlignLeft
		}
	}
	return aligns, true
}
func normalizeAligns(a []Align, cols int) []Align {
	if cols <= 0 {
		return a
	}
	out := make([]Align, cols)
	copy(out, a)
	return out
}

var refRe = regexp.MustCompile(`^\s{0,3}\[([^\]]+)\]:\s*(\S+)(?:\s+"([^"]*)")?\s*$`)
var footRe = regexp.MustCompile(`^\s{0,3}\[\^([^\]]+)\]:\s*(.*)$`)

func linkDefinition(s string) (string, LinkRef, bool) {
	m := refRe.FindStringSubmatch(s)
	if len(m) == 0 {
		return "", LinkRef{}, false
	}
	return m[1], LinkRef{URL: strings.Trim(m[2], "<>"), Title: m[3]}, true
}
func footnoteDefinition(s string) (string, string, bool) {
	m := footRe.FindStringSubmatch(s)
	if len(m) == 0 {
		return "", "", false
	}
	return m[1], strings.TrimSpace(m[2]), true
}

func uniqueSlug(s string, existing []HeadingRef) string {
	base := slug(s)
	if base == "" {
		base = "section"
	}
	seen := make(map[string]struct{}, len(existing))
	for _, h := range existing {
		seen[h.ID] = struct{}{}
	}
	if _, ok := seen[base]; !ok {
		return base
	}
	for i := 2; ; i++ {
		v := base + "-" + strconv.Itoa(i)
		if _, ok := seen[v]; !ok {
			return v
		}
	}
}
func slug(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(stripInlineMarkers(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func stripInlineMarkers(s string) string {
	repl := []string{"**", "", "__", "", "~~", "", "`", "", "*", "", "_", ""}
	return strings.NewReplacer(repl...).Replace(s)
}

func plainTextNodes(nodes []Node) string {
	var parts []string
	var walk func([]Node)
	walk = func(ns []Node) {
		for _, n := range ns {
			if n.Text != "" {
				parts = append(parts, n.Text)
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(nodes)
	return strings.TrimSpace(strings.Join(parts, " "))
}

func leadingSpaces(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 4
			continue
		}
		break
	}
	return n
}
func removeIndent(s string, n int) string {
	removed := 0
	for i, r := range s {
		if removed >= n {
			return s[i:]
		}
		if r == ' ' {
			removed++
			continue
		}
		if r == '\t' {
			removed += 4
			continue
		}
		return s[i:]
	}
	return ""
}
func pos(l sourceLine, col int) Position { return Position{Line: l.No, Column: col} }

func (k Kind) String() string { return fmt.Sprintf("Kind(%d)", k) }
