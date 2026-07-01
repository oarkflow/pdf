package export

import (
	"bytes"
	"html"
	"strconv"
	"strings"

	"github.com/oarkflow/pdf/md/internal/markdown"
)

type HTML struct{}

func (HTML) Export(d markdown.Doc, o Options) ([]byte, error) {
	applyMeta(&o, d)
	var b bytes.Buffer
	if !o.Standalone {
		o.Standalone = true
	}
	if o.Standalone {
		b.WriteString("<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
		b.WriteString(`<meta name="generator" content="mdany">`)
		if o.Author != "" {
			b.WriteString(`<meta name="author" content="` + attrEsc(o.Author) + `">`)
		}
		b.WriteString("<title>")
		b.WriteString(html.EscapeString(defaultTitle(o.Title)))
		b.WriteString("</title><style>")
		b.WriteString(modernCSS(o.Theme))
		if o.CSS != "" {
			b.WriteString("\n")
			b.WriteString(o.CSS)
		}
		b.WriteString("</style></head><body>")
	}
	b.WriteString(`<main class="mdany-shell">`)
	writeHero(&b, d, o)
	if o.TOC && len(d.Headings) > 1 {
		writeTOC(&b, d.Headings)
	}
	b.WriteString(`<article class="mdany-doc">`)
	for _, n := range d.Nodes {
		writeHTML(&b, n, o, d.Refs)
	}
	b.WriteString(`</article></main>`)
	if o.Standalone {
		b.WriteString("</body></html>")
	}
	return b.Bytes(), nil
}

func applyMeta(o *Options, d markdown.Doc) {
	if o.Title == "" {
		o.Title = d.Meta["title"]
	}
	if o.Author == "" {
		o.Author = d.Meta["author"]
	}
	if o.Theme == "" {
		o.Theme = d.Meta["theme"]
	}
}
func defaultTitle(s string) string {
	if s == "" {
		return "Markdown Document"
	}
	return s
}

func writeHero(b *bytes.Buffer, d markdown.Doc, o Options) {
	if o.Title == "" && o.Author == "" {
		return
	}
	b.WriteString(`<header class="mdany-hero">`)
	if o.Title != "" {
		b.WriteString(`<h1>` + inlineHTML(o.Title) + `</h1>`)
	}
	if o.Author != "" {
		b.WriteString(`<p class="mdany-meta">Prepared by ` + html.EscapeString(o.Author) + `</p>`)
	}
	b.WriteString(`</header>`)
}
func writeTOC(b *bytes.Buffer, hs []markdown.HeadingRef) {
	b.WriteString(`<nav class="mdany-toc" aria-label="Table of contents"><strong>Contents</strong><ol>`)
	for _, h := range hs {
		if h.Level > 3 {
			continue
		}
		b.WriteString(`<li class="toc-l` + strconv.Itoa(h.Level) + `"><a href="#` + attrEsc(h.ID) + `">` + inlineHTML(h.Text) + `</a></li>`)
	}
	b.WriteString(`</ol></nav>`)
}
func writeHTML(b *bytes.Buffer, n markdown.Node, o Options, refs map[string]markdown.LinkRef) {
	switch n.Kind {
	case markdown.Heading:
		lvl := n.Level
		if lvl < 1 || lvl > 6 {
			lvl = 1
		}
		b.WriteString("<h" + strconv.Itoa(lvl))
		if n.ID != "" {
			b.WriteString(` id="` + attrEsc(n.ID) + `"`)
		}
		b.WriteByte('>')
		b.WriteString(inlineHTMLRefs(n.Text, refs))
		b.WriteString("</h" + strconv.Itoa(lvl) + ">")
	case markdown.Paragraph:
		b.WriteString("<p>")
		b.WriteString(inlineHTMLRefs(n.Text, refs))
		b.WriteString("</p>")
	case markdown.BlockQuote:
		b.WriteString("<blockquote>")
		if len(n.Children) > 0 {
			for _, c := range n.Children {
				writeHTML(b, c, o, refs)
			}
		} else {
			b.WriteString(inlineHTMLRefs(n.Text, refs))
		}
		b.WriteString("</blockquote>")
	case markdown.Alert:
		kind := n.Info
		if kind == "" {
			kind = "note"
		}
		b.WriteString(`<aside class="mdany-alert ` + attrEsc(kind) + `"><strong>` + html.EscapeString(strings.ToUpper(kind[:1])+kind[1:]) + `</strong><p>` + inlineHTMLRefs(n.Text, refs) + `</p></aside>`)
	case markdown.HorizontalRule:
		if !o.SoftHR {
			b.WriteString(`<div class="mdany-gap" aria-hidden="true"></div>`)
		}
	case markdown.CodeBlock:
		b.WriteString(`<figure class="mdany-code">`)
		if n.Info != "" {
			b.WriteString(`<figcaption>` + html.EscapeString(n.Info) + `</figcaption>`)
		}
		b.WriteString("<pre><code>")
		b.WriteString(html.EscapeString(n.Text))
		b.WriteString("</code></pre></figure>")
	case markdown.BulletList, markdown.OrderedList:
		tag := "ul"
		if n.Kind == markdown.OrderedList {
			tag = "ol"
		}
		classes := ""
		if hasTasks(n) {
			classes = ` class="task-list"`
		}
		b.WriteByte('<')
		b.WriteString(tag)
		b.WriteString(classes)
		b.WriteByte('>')
		for _, c := range n.Children {
			b.WriteString("<li>")
			if c.Checked != nil {
				checked := ""
				if *c.Checked {
					checked = " checked"
				}
				b.WriteString(`<input type="checkbox" disabled` + checked + `> `)
			}
			b.WriteString(inlineHTMLRefs(c.Text, refs))
			if len(c.Children) > 0 {
				for _, child := range c.Children {
					writeHTML(b, child, o, refs)
				}
			}
			b.WriteString("</li>")
		}
		b.WriteString("</" + tag + ">")
	case markdown.Table:
		writeHTMLTable(b, n, refs)
	case markdown.HTMLBlock:
		if o.SoftHR {
			b.WriteString("<pre class=\"mdany-html-escaped\">")
			b.WriteString(html.EscapeString(n.Text))
			b.WriteString("</pre>")
		} else {
			b.WriteString(n.Text)
		}
	case markdown.DefinitionList:
		b.WriteString("<dl>")
		for _, c := range n.Children {
			if c.Kind == markdown.DefinitionTerm {
				b.WriteString("<dt>" + inlineHTMLRefs(c.Text, refs) + "</dt>")
			} else {
				b.WriteString("<dd>" + inlineHTMLRefs(c.Text, refs) + "</dd>")
			}
		}
		b.WriteString("</dl>")
	case markdown.FootnoteList:
		b.WriteString(`<section class="footnotes"><hr><ol>`)
		for _, f := range n.Children {
			b.WriteString(`<li id="fn-` + attrEsc(f.ID) + `">`)
			if len(f.Children) > 0 {
				for _, c := range f.Children {
					writeHTML(b, c, o, refs)
				}
			} else {
				b.WriteString(inlineHTMLRefs(f.Text, refs))
			}
			b.WriteString(` <a href="#fnref-` + attrEsc(f.ID) + `">↩</a></li>`)
		}
		b.WriteString("</ol></section>")
	}
}
func hasTasks(n markdown.Node) bool {
	for _, c := range n.Children {
		if c.Checked != nil {
			return true
		}
	}
	return false
}
func writeHTMLTable(b *bytes.Buffer, n markdown.Node, refs map[string]markdown.LinkRef) {
	if len(n.Rows) == 0 {
		return
	}
	cols := 0
	for _, r := range n.Rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	b.WriteString(`<div class="table-wrap"><table><thead><tr>`)
	for i := 0; i < cols; i++ {
		b.WriteString(`<th`)
		writeAlign(b, n, i)
		b.WriteString(`>`)
		if i < len(n.Rows[0]) {
			b.WriteString(inlineHTMLRefs(n.Rows[0][i], refs))
		}
		b.WriteString(`</th>`)
	}
	b.WriteString(`</tr></thead>`)
	if len(n.Rows) > 1 {
		b.WriteString("<tbody>")
		for _, row := range n.Rows[1:] {
			b.WriteString("<tr>")
			for i := 0; i < cols; i++ {
				b.WriteString(`<td`)
				writeAlign(b, n, i)
				b.WriteString(`>`)
				if i < len(row) {
					b.WriteString(inlineHTMLRefs(row[i], refs))
				}
				b.WriteString(`</td>`)
			}
			b.WriteString("</tr>")
		}
		b.WriteString("</tbody>")
	}
	b.WriteString("</table></div>")
}
func writeAlign(b *bytes.Buffer, n markdown.Node, i int) {
	if i >= len(n.Aligns) {
		return
	}
	switch n.Aligns[i] {
	case markdown.AlignCenter:
		b.WriteString(` style="text-align:center"`)
	case markdown.AlignRight:
		b.WriteString(` style="text-align:right"`)
	}
}
func modernCSS(theme string) string {
	base := `:root{--bg:#f3f6fb;--paper:#fff;--ink:#18212f;--muted:#667085;--border:#d9e2ef;--soft:#f7f9fc;--head:#eef4ff;--accent:#2458d3;--accent2:#0f766e;--shadow:0 12px 38px rgba(15,23,42,.08)}@media(prefers-color-scheme:dark){:root{--bg:#0f172a;--paper:#111827;--ink:#e5e7eb;--muted:#a3afc2;--border:#334155;--soft:#172033;--head:#1d2b47;--accent:#7aa7ff;--accent2:#4bd1bf;--shadow:none}}*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;background:var(--bg);color:var(--ink);font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Arial,sans-serif;font-size:16px;line-height:1.56}.mdany-shell{max-width:980px;margin:28px auto;padding:0 22px}.mdany-hero{background:linear-gradient(135deg,var(--accent),var(--accent2));color:#fff;border-radius:18px;padding:26px 30px;margin:0 0 18px;box-shadow:var(--shadow)}.mdany-hero h1{margin:0;font-size:clamp(1.75rem,3.6vw,2.65rem);line-height:1.12;letter-spacing:-.035em}.mdany-meta{margin:9px 0 0;opacity:.88}.mdany-doc,.mdany-toc{background:var(--paper);border:1px solid var(--border);border-radius:18px;box-shadow:var(--shadow)}.mdany-doc{padding:34px 38px}.mdany-toc{padding:18px 22px;margin:0 0 18px}.mdany-toc strong{display:block;margin-bottom:6px;color:var(--accent)}.mdany-toc ol{columns:2;margin:0;padding-left:1.15rem}.mdany-toc li{break-inside:avoid;margin:.16rem 0}.mdany-toc .toc-l2{margin-left:.6rem}.mdany-toc .toc-l3{margin-left:1.2rem;font-size:.94em}a{color:var(--accent);text-decoration:none}a:hover{text-decoration:underline}h1,h2,h3,h4,h5,h6{line-height:1.24;letter-spacing:-.02em;margin:1.25em 0 .45em;color:var(--ink)}.mdany-doc>h1:first-child,.mdany-doc>h2:first-child,.mdany-doc>h3:first-child{margin-top:0}h1{font-size:2rem}h2{font-size:1.48rem;border-bottom:1px solid var(--border);padding-bottom:.28rem}h3{font-size:1.18rem}h4{font-size:1.05rem}p{margin:.45em 0 .85em;color:var(--ink)}ul,ol{margin:.35em 0 .85em 1.25em;padding-left:1em}li{margin:.22em 0;padding-left:.1em}li::marker{font-weight:700;color:var(--accent)}.task-list{list-style:none;margin-left:0;padding-left:0}.task-list input{transform:translateY(1px);margin-right:.45rem}.table-wrap{overflow-x:auto;margin:.9em 0 1.1em;border-radius:12px;border:1px solid var(--border)}table{border-collapse:separate;border-spacing:0;width:100%;font-size:.94rem;background:var(--paper)}thead th{background:var(--head);font-weight:750;color:var(--ink)}tbody tr:nth-child(even){background:var(--soft)}td,th{border-right:1px solid var(--border);border-bottom:1px solid var(--border);padding:9px 11px;vertical-align:top;text-align:left}td:last-child,th:last-child{border-right:0}tbody tr:last-child td{border-bottom:0}figure{margin:.9em 0 1.05em}.mdany-code{border:1px solid #243148;border-radius:12px;background:#0d1324;color:#e8edf6;overflow:hidden}.mdany-code figcaption{padding:7px 12px;border-bottom:1px solid rgba(255,255,255,.12);color:#bfdbfe;font-size:.82rem}.mdany-code pre{margin:0;padding:13px 14px;overflow:auto;line-height:1.45}.mdany-code code,code{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace}p code,li code,td code{background:var(--soft);border:1px solid var(--border);border-radius:5px;padding:.1em .3em}blockquote{border-left:4px solid var(--accent);margin:.9em 0;padding:.15em 0 .15em .9em;color:var(--muted);background:linear-gradient(90deg,rgba(36,88,211,.07),transparent);border-radius:0 10px 10px 0}.mdany-alert{border:1px solid var(--border);border-left:5px solid var(--accent);border-radius:12px;background:var(--soft);padding:11px 13px;margin:.9em 0}.mdany-alert strong{display:block;margin-bottom:.2rem}.mdany-alert.warning,.mdany-alert.caution{border-left-color:#d97706}.mdany-alert.danger,.mdany-alert.important{border-left-color:#dc2626}.mdany-alert.tip,.mdany-alert.success{border-left-color:#059669}.mdany-alert p{margin:0}.mdany-gap{height:.35rem}img{max-width:100%;border-radius:10px}@page{margin:16mm}@media(max-width:720px){.mdany-shell{margin:0;padding:0}.mdany-hero,.mdany-doc,.mdany-toc{border-radius:0;border-left:0;border-right:0}.mdany-doc{padding:24px 20px}.mdany-hero{padding:24px 20px}.mdany-toc ol{columns:1}}@media print{body{background:#fff;font-size:11pt;line-height:1.42}.mdany-shell{max-width:none;margin:0;padding:0}.mdany-hero,.mdany-doc,.mdany-toc{box-shadow:none}.mdany-doc,.mdany-toc{border:0;padding:0}.mdany-hero{border-radius:0;margin:0 0 12px;padding:18px 22px}.mdany-toc{margin-bottom:12px}.mdany-toc ol{columns:1}h1,h2,h3,h4{page-break-after:avoid}table,pre,blockquote,.mdany-alert{page-break-inside:avoid}a{color:inherit;text-decoration:none}}`
	if strings.EqualFold(theme, "minimal") {
		base += `.mdany-hero{background:var(--paper);color:var(--ink);border:1px solid var(--border)}.mdany-doc,.mdany-toc{box-shadow:none}`
	}
	return base
}
