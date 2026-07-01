package export

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/oarkflow/pdf/md/internal/markdown"
)

type PDF struct{}

type pdfPage struct{ c bytes.Buffer }

type pdfDestination struct {
	Page int
	X    float64
	Y    float64
}

type pdfLink struct {
	Page   int
	X1     float64
	Y1     float64
	X2     float64
	Y2     float64
	Target string
}

type pdfWriter struct {
	pages        []pdfPage
	pageW        float64
	pageH        float64
	margin       float64
	y            float64
	contentW     float64
	pageNumber   int
	destinations map[string]pdfDestination
	links        []pdfLink
}

func (PDF) Export(d markdown.Doc, o Options) ([]byte, error) {
	applyMeta(&o, d)
	pw := &pdfWriter{pageW: 595, pageH: 842, margin: o.Margin, destinations: make(map[string]pdfDestination)}
	if strings.EqualFold(o.PageSize, "letter") {
		pw.pageW, pw.pageH = 612, 792
	}
	if pw.margin <= 0 {
		pw.margin = 54
	}
	// The PDF renderer follows a clean modern business-report layout: generous
	// margins, strong typography, subtle section separators, airy paragraphs and
	// borderless striped tables with dark header bands. This intentionally matches
	// the provided CARE status proposal style while keeping mdany fully native.
	pw.contentW = pw.pageW - 2*pw.margin
	pw.newPage()
	if o.TOC && len(d.Headings) > 1 {
		pw.pdfTOC(d.Headings)
		pw.newPage()
	}
	for _, n := range d.Nodes {
		pw.node(n)
	}
	return buildPDF(pw.pages, pw.destinations, pw.links, o, pw.pageW, pw.pageH, pw.margin)
}

func (p *pdfWriter) newPage() {
	p.pages = append(p.pages, pdfPage{})
	p.pageNumber = len(p.pages)
	p.y = p.pageH - p.margin
}
func (p *pdfWriter) page() *bytes.Buffer { return &p.pages[len(p.pages)-1].c }
func (p *pdfWriter) ensure(h float64) {
	if p.y-h < p.margin+24 {
		p.newPage()
	}
}
func (p *pdfWriter) text(font string, size, x, y float64, s string) {
	fmt.Fprintf(p.page(), "BT /%s %.1f Tf 0 g 1 0 0 1 %.1f %.1f Tm (%s) Tj ET\n", font, size, x, y, pdfEsc(s))
}
func (p *pdfWriter) rect(x, y, w, h, gray float64) {
	fmt.Fprintf(p.page(), "%.3f g %.1f %.1f %.1f %.1f re f 0 g\n", gray, x, y, w, h)
}
func (p *pdfWriter) strokeRect(x, y, w, h float64) {
	fmt.Fprintf(p.page(), "0.72 G 0.5 w %.1f %.1f %.1f %.1f re S 0 G\n", x, y, w, h)
}
func (p *pdfWriter) rgbRect(x, y, w, h, r, g, b float64) {
	fmt.Fprintf(p.page(), "%.3f %.3f %.3f rg %.1f %.1f %.1f %.1f re f 0 g\n", r, g, b, x, y, w, h)
}
func (p *pdfWriter) colorText(font string, size, x, y float64, s string, r, g, b float64) {
	fmt.Fprintf(p.page(), "BT /%s %.1f Tf %.3f %.3f %.3f rg 1 0 0 1 %.1f %.1f Tm (%s) Tj ET 0 g\n", font, size, r, g, b, x, y, pdfEsc(s))
}
func (p *pdfWriter) line(x1, y1, x2, y2 float64) {
	fmt.Fprintf(p.page(), "0.72 G 0.7 w %.1f %.1f m %.1f %.1f l S 0 G\n", x1, y1, x2, y2)
}
func (p *pdfWriter) subtleLine(x1, y1, x2, y2 float64) {
	fmt.Fprintf(p.page(), "0.82 0.86 0.90 RG 0.9 w %.1f %.1f m %.1f %.1f l S 0 G\n", x1, y1, x2, y2)
}
func (p *pdfWriter) whiteText(font string, size, x, y float64, s string) {
	fmt.Fprintf(p.page(), "BT /%s %.1f Tf 1 1 1 rg 1 0 0 1 %.1f %.1f Tm (%s) Tj ET 0 g\n", font, size, x, y, pdfEsc(s))
}
func (p *pdfWriter) cover(o Options) {
	// The CARE-style PDF theme does not render a separate cover card. The first
	// level-one heading in the Markdown is treated as the document title so output
	// stays close to professional proposal PDFs authored in word processors.
}
func (p *pdfWriter) pdfTOC(hs []markdown.HeadingRef) {
	p.ensure(90)
	p.colorText("F2", 28, p.margin, p.y, "Contents", 0.055, 0.085, 0.120)
	p.subtleLine(p.margin, p.y-15, p.margin+p.contentW, p.y-15)
	p.y -= 44
	for _, h := range hs {
		if h.Level > 3 {
			continue
		}
		indent := float64(h.Level-1) * 18
		font := "F1"
		size := 10.5
		leading := 1.25
		if h.Level == 1 {
			font = "F2"
			size = 11.5
		}
		p.writeTOCEntry(h, font, size, p.margin+indent, p.contentW-indent, leading, 5.5)
	}
	p.y -= 4
}

func (p *pdfWriter) writeTOCEntry(h markdown.HeadingRef, font string, size, x, w, leading, after float64) {
	max := int(w / (size * 0.45))
	if max < 12 {
		max = 12
	}
	lines := wrapText(plainInline(h.Text), max)
	lineH := size * leading
	p.ensure(float64(len(lines))*lineH + after)
	for _, ln := range lines {
		p.text(font, size, x, p.y, ln)
		// The visible text remains clean, but the whole TOC row is a PDF link
		// annotation. PDF annotations are added during final object assembly
		// once the destination page object IDs are known.
		p.addLink(x, p.y-2, x+w, p.y+size+2, h.ID)
		p.y -= lineH
	}
	p.y -= after
}

func (p *pdfWriter) addLink(x1, y1, x2, y2 float64, target string) {
	if target == "" {
		return
	}
	p.links = append(p.links, pdfLink{Page: p.pageNumber, X1: x1, Y1: y1, X2: x2, Y2: y2, Target: target})
}

func (p *pdfWriter) node(n markdown.Node) {
	switch n.Kind {
	case markdown.Heading:
		p.writeHeading(n)
	case markdown.Paragraph:
		p.writeWrapped(clean(n.Text), "F1", 11.0, p.margin, p.contentW, 1.34, 8)
	case markdown.BlockQuote:
		p.ensure(36)
		beforeY := p.y
		p.writeWrapped(clean(n.Text), "F1", 10.8, p.margin+18, p.contentW-24, 1.36, 8)
		p.subtleLine(p.margin+5, beforeY+3, p.margin+5, p.y+8)
	case markdown.Alert:
		p.writeAlert(n)
	case markdown.HorizontalRule:
		// Thematic breaks are a gentle content pause in PDFs. Avoid visible
		// full-width rules; they were visually noisy in long business docs.
		p.y -= 4
	case markdown.CodeBlock:
		p.writeCodeBlock(n)
	case markdown.BulletList:
		for _, c := range n.Children {
			marker := "•"
			if c.Checked != nil {
				marker = "[ ]"
				if *c.Checked {
					marker = "[x]"
				}
			}
			p.writeListItem(marker, clean(c.Text), 10.4)
		}
		p.y -= 2
	case markdown.OrderedList:
		for i, c := range n.Children {
			p.writeListItem(strconv.Itoa(i+1)+".", clean(c.Text), 10.4)
		}
		p.y -= 2
	case markdown.Table:
		p.writeTable(n)
	}
}
func (p *pdfWriter) recordDestination(id string, x, y float64) {
	if id == "" {
		return
	}
	p.destinations[id] = pdfDestination{Page: p.pageNumber, X: x, Y: y}
}

func (p *pdfWriter) writeHeading(n markdown.Node) {
	sz := 13.0
	after := 8.0
	before := 12.0
	drawRule := false
	switch n.Level {
	case 1:
		sz, before, after, drawRule = 26.0, 0, 30, true
	case 2:
		sz, before, after, drawRule = 19.5, 23, 12, true
	case 3:
		sz, before, after = 14.5, 14, 6
	case 4:
		sz, before, after = 12.5, 11, 5
	}
	if before > 0 {
		p.y -= before
	}
	lines := wrapText(clean(n.Text), int(p.contentW/(sz*0.43)))
	req := float64(len(lines))*sz*1.10 + after
	if drawRule {
		req += 13
	}
	keepWithNext := 0.0
	if n.Level <= 2 {
		keepWithNext = 118
	} else if n.Level == 3 {
		keepWithNext = 46
	}
	if p.y-req-keepWithNext < p.margin+24 {
		p.newPage()
	} else {
		p.ensure(req)
	}
	p.recordDestination(n.ID, p.margin, p.y+sz)
	for _, ln := range lines {
		p.colorText("F2", sz, p.margin, p.y, ln, 0.055, 0.085, 0.120)
		p.y -= sz * 1.10
	}
	if drawRule {
		p.subtleLine(p.margin, p.y+10, p.margin+p.contentW, p.y+10)
	}
	p.y -= after
}
func (p *pdfWriter) writeAlert(n markdown.Node) {
	kind := strings.ToUpper(n.Info)
	if kind == "" {
		kind = "NOTE"
	}
	lines := wrapText(clean(n.Text), int((p.contentW-36)/(10.5*0.50)))
	h := float64(len(lines))*14.2 + 28
	p.ensure(h)
	p.rgbRect(p.margin, p.y-h+8, p.contentW, h, 0.965, 0.981, 0.996)
	p.rgbRect(p.margin, p.y-h+8, 5, h, 0.200, 0.290, 0.380)
	p.colorText("F2", 9.5, p.margin+16, p.y-9, kind, 0.090, 0.140, 0.190)
	p.y -= 23
	for _, ln := range lines {
		p.text("F1", 10.5, p.margin+16, p.y, ln)
		p.y -= 14.2
	}
	p.y -= 9
}
func (p *pdfWriter) writeCodeBlock(n markdown.Node) {
	lines := strings.Split(n.Text, "\n")
	lineH := 10.5
	for start := 0; start < len(lines); {
		available := int((p.y - (p.margin + 34)) / lineH)
		if available < 3 {
			p.newPage()
			available = int((p.y - (p.margin + 34)) / lineH)
		}
		end := start + available
		if end > len(lines) {
			end = len(lines)
		}
		h := float64(end-start)*lineH + 14
		p.ensure(h)
		p.rgbRect(p.margin, p.y-h+7, p.contentW, h, 0.958, 0.965, 0.972)
		ty := p.y - 9
		for _, ln := range lines[start:end] {
			p.text("F3", 8.8, p.margin+9, ty, truncateASCII(ln, 118))
			ty -= lineH
		}
		p.y -= h + 4
		start = end
	}
}
func (p *pdfWriter) writeWrapped(s, font string, size, x, w, leading, after float64) {
	max := int(w / (size * 0.45))
	if max < 12 {
		max = 12
	}
	lines := wrapText(s, max)
	p.ensure(float64(len(lines))*size*leading + after)
	for _, ln := range lines {
		p.text(font, size, x, p.y, ln)
		p.y -= size * leading
	}
	p.y -= after
}
func (p *pdfWriter) writeListItem(marker, s string, size float64) {
	max := int((p.contentW - 44) / (size * 0.45))
	if max < 12 {
		max = 12
	}
	lines := wrapText(s, max)
	lineH := size * 1.30
	p.ensure(float64(len(lines))*lineH + 4)
	p.text("F1", size, p.margin+24, p.y, marker)
	for i, ln := range lines {
		if i > 0 {
			p.y -= lineH
		}
		p.text("F1", size, p.margin+58, p.y, ln)
	}
	p.y -= lineH + 1
}
func (p *pdfWriter) writeTable(n markdown.Node) {
	if len(n.Rows) == 0 {
		return
	}
	cols := 1
	for _, r := range n.Rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	colWs := p.tableWidths(n.Rows, cols)
	pad := 7.0
	fontSize := 9.6
	if cols >= 5 {
		fontSize = 8.8
	}
	lineH := fontSize * 1.42
	p.y -= 8
	if len(n.Rows) <= 10 {
		totalH := 0.0
		for _, row := range n.Rows {
			totalH += p.tableRowHeight(row, cols, colWs, pad, fontSize, lineH)
		}
		if p.y-totalH < p.margin+34 {
			p.newPage()
		}
	}
	var header [][]string
	if len(n.Rows) > 0 {
		header = [][]string{n.Rows[0]}
	}
	for ri, row := range n.Rows {
		rowH := p.tableRowHeight(row, cols, colWs, pad, fontSize, lineH)
		if p.y-rowH < p.margin+34 {
			p.newPage()
			if ri > 0 && len(header) > 0 {
				p.drawTableRow(header[0], n.Aligns, cols, colWs, pad, fontSize, lineH, 0)
			}
		}
		p.drawTableRow(row, n.Aligns, cols, colWs, pad, fontSize, lineH, ri)
	}
	p.y -= 17
}
func (p *pdfWriter) tableRowHeight(row []string, cols int, colWs []float64, pad, fontSize, lineH float64) float64 {
	maxLines := 1
	for ci := 0; ci < cols; ci++ {
		text := ""
		if ci < len(row) {
			text = clean(row[ci])
		}
		maxChars := int((colWs[ci] - 2*pad) / (fontSize * 0.48))
		if maxChars < 8 {
			maxChars = 8
		}
		lines := wrapText(text, maxChars)
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	rowH := float64(maxLines)*lineH + 18
	if rowH < 35 {
		rowH = 35
	}
	return rowH
}
func (p *pdfWriter) drawTableRow(row []string, aligns []markdown.Align, cols int, colWs []float64, pad, fontSize, lineH float64, ri int) {
	rowH := p.tableRowHeight(row, cols, colWs, pad, fontSize, lineH)
	cellLines := make([][]string, cols)
	for ci := 0; ci < cols; ci++ {
		text := ""
		if ci < len(row) {
			text = clean(row[ci])
		}
		maxChars := int((colWs[ci] - 2*pad) / (fontSize * 0.48))
		if maxChars < 8 {
			maxChars = 8
		}
		lines := wrapText(text, maxChars)
		if len(lines) == 0 {
			lines = []string{""}
		}
		cellLines[ci] = lines
	}
	y0 := p.y - rowH + 6
	if ri == 0 {
		p.rgbRect(p.margin, y0, p.contentW, rowH, 0.200, 0.290, 0.380)
	} else if ri%2 == 0 {
		p.rgbRect(p.margin, y0, p.contentW, rowH, 0.965, 0.973, 0.982)
	}
	// CARE-style tables are intentionally light: dark header band, alternating
	// row fills and no boxed grid. Only subtle horizontal separators are used.
	if ri > 0 {
		p.subtleLine(p.margin, y0, p.margin+p.contentW, y0)
	}
	x := p.margin
	for ci := 0; ci < cols; ci++ {
		w := colWs[ci]
		font := "F1"
		if ri == 0 {
			font = "F2"
		}
		ty := p.y - 14
		for _, ln := range cellLines[ci] {
			tx := x + pad
			if ri > 0 && ci < len(row) {
				// Respect markdown column alignment without making the table look mechanical.
				// Exact text width is approximated by the same simple font metric used for wrapping.
				textW := float64(utf8.RuneCountInString(ln)) * fontSize * 0.50
				switch tableAlign(aligns, ci) {
				case markdown.AlignRight:
					tx = x + w - pad - textW
				case markdown.AlignCenter:
					tx = x + (w-textW)/2
				}
			}
			if ri == 0 {
				p.whiteText(font, fontSize, tx, ty, ln)
			} else {
				p.text(font, fontSize, tx, ty, ln)
			}
			ty -= lineH
		}
		x += w
	}
	p.y -= rowH
}
func tableAlign(aligns []markdown.Align, ci int) markdown.Align {
	if ci >= 0 && ci < len(aligns) {
		return aligns[ci]
	}
	return markdown.AlignLeft
}
func (p *pdfWriter) tableWidths(rows [][]string, cols int) []float64 {
	weights := make([]float64, cols)
	for i := range weights {
		weights[i] = 1
	}
	for _, row := range rows {
		for i := 0; i < cols && i < len(row); i++ {
			w := float64(utf8.RuneCountInString(clean(row[i])))
			if w > 42 {
				w = 42
			}
			weights[i] += w
		}
	}
	if cols >= 4 {
		for i := range weights {
			if weights[i] < 20 {
				weights[i] = 20
			}
		}
	}
	total := 0.0
	for _, w := range weights {
		total += w
	}
	out := make([]float64, cols)
	used := 0.0
	for i, w := range weights {
		out[i] = p.contentW * (w / total)
		min := 68.0
		if cols >= 5 {
			min = 56
		}
		if out[i] < min {
			out[i] = min
		}
		used += out[i]
	}
	if used != p.contentW {
		scale := p.contentW / used
		for i := range out {
			out[i] *= scale
		}
	}
	return out
}
func wrapText(s string, max int) []string {
	if max < 10 {
		max = 80
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var res []string
	line := ""
	for _, w := range words {
		if utf8.RuneCountInString(line)+utf8.RuneCountInString(w)+1 > max {
			if line != "" {
				res = append(res, line)
			}
			line = w
		} else {
			if line != "" {
				line += " "
			}
			line += w
		}
	}
	if line != "" {
		res = append(res, line)
	}
	return res
}
func buildPDF(pages []pdfPage, destinations map[string]pdfDestination, links []pdfLink, o Options, pageW, pageH, margin float64) ([]byte, error) {
	var objects []string
	add := func(s string) int { objects = append(objects, s); return len(objects) }
	fontReg := add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	fontBold := add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	fontMono := add("<< /Type /Font /Subtype /Type1 /BaseFont /Courier /Encoding /WinAnsiEncoding >>")
	var pageIDs, contentIDs []int
	for i := range pages {
		contentID := add(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", pages[i].c.Len(), pages[i].c.String()))
		contentIDs = append(contentIDs, contentID)
		pageIDs = append(pageIDs, add(""))
	}
	kids := bytes.Buffer{}
	for _, id := range pageIDs {
		kids.WriteString(fmt.Sprintf("%d 0 R ", id))
	}
	pagesID := add(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [ %s] >>", len(pageIDs), kids.String()))
	annotsByPage := make(map[int][]int)
	for _, l := range links {
		d, ok := destinations[l.Target]
		if !ok || l.Page < 1 || l.Page > len(pageIDs) || d.Page < 1 || d.Page > len(pageIDs) {
			continue
		}
		ann := add(fmt.Sprintf("<< /Type /Annot /Subtype /Link /Rect [%.1f %.1f %.1f %.1f] /Border [0 0 0] /Dest [%d 0 R /XYZ %.1f %.1f 0] >>", l.X1, l.Y1, l.X2, l.Y2, pageIDs[d.Page-1], d.X, d.Y))
		annotsByPage[l.Page] = append(annotsByPage[l.Page], ann)
	}
	for i, id := range pageIDs {
		annots := ""
		if ids := annotsByPage[i+1]; len(ids) > 0 {
			var ab bytes.Buffer
			for _, aid := range ids {
				ab.WriteString(fmt.Sprintf("%d 0 R ", aid))
			}
			annots = fmt.Sprintf(" /Annots [ %s]", ab.String())
		}
		objects[id-1] = fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.0f %.0f] /Resources << /Font << /F1 %d 0 R /F2 %d 0 R /F3 %d 0 R >> >> /Contents %d 0 R%s >>", pagesID, pageW, pageH, fontReg, fontBold, fontMono, contentIDs[i], annots)
	}
	title := o.Title
	if title == "" {
		title = "Markdown Document"
	}
	catalog := add(fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesID))
	info := add(fmt.Sprintf("<< /Title (%s) /Producer (mdany) /CreationDate (D:%s) >>", pdfEsc(title), time.Now().Format("20060102150405")))
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root %d 0 R /Info %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, catalog, info, xref)
	return out.Bytes(), nil
}
func pdfEsc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '(':
			b.WriteString(`\(`)
		case ')':
			b.WriteString(`\)`)
		case '•':
			b.WriteString(`\225`)
		case '–', '—':
			b.WriteByte('-')
		case '“', '”':
			b.WriteByte('"')
		case '’', '‘':
			b.WriteByte('\'')
		case '\n', '\t':
			b.WriteByte(' ')
		default:
			if r >= 32 && r < 127 {
				b.WriteRune(r)
			} else {
				b.WriteByte('?')
			}
		}
	}
	return b.String()
}
func sanitizePDF(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 32 && r < 127 {
			b.WriteRune(r)
		} else if r == '\n' || r == '\t' {
			b.WriteByte(' ')
		} else {
			b.WriteByte('?')
		}
	}
	return b.String()
}
func clean(s string) string { return plainInline(s) }
func truncateASCII(s string, n int) string {
	s = sanitizePDF(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}
