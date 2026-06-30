// Package markdown renders Markdown as self-contained, print-oriented HTML.
package markdown

import (
	"bytes"
	"html"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Options controls the generated HTML document.
type Options struct {
	Title      string
	Theme      string
	Stylesheet string
}

// Render converts Markdown to a complete, print-oriented HTML document. GFM
// tables, task lists, strikethrough, and autolinks are enabled in addition to
// CommonMark. Raw HTML remains disabled so it is not emitted into the document.
func Render(source string, opts ...Options) string {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Theme == "" {
		opt.Theme = "classic"
	}
	engine := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	sourceBytes := []byte(source)
	document := engine.Parser().Parse(text.NewReader(sourceBytes))
	var body bytes.Buffer
	if err := engine.Renderer().Render(&body, sourceBytes, document); err != nil {
		// bytes.Buffer writes cannot fail; retain a useful, safely escaped body if
		// a custom Goldmark renderer ever changes that assumption.
		body.WriteString("<p>" + html.EscapeString(source) + "</p>")
	}
	title := opt.Title
	if title == "" {
		title = documentTitle(document, sourceBytes)
	}
	if title == "" {
		title = "Document"
	}
	css := stylesheet(opt.Theme)
	if opt.Stylesheet != "" {
		css += "\n" + opt.Stylesheet
	}
	return "<!doctype html><html><head><meta charset=\"utf-8\"><title>" + html.EscapeString(title) +
		"</title><style>" + css + "</style></head><body><main class=\"markdown-body\">" + body.String() + "</main></body></html>"
}

func documentTitle(document ast.Node, source []byte) string {
	var title strings.Builder
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, ok := node.(*ast.Heading); ok {
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if text, ok := child.(*ast.Text); ok {
					title.Write(text.Segment.Value(source))
				}
			}
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(title.String())
}

func stylesheet(theme string) string {
	base := `@page { size: A4; margin: 19mm 18mm 21mm; }
body { color: #20242a; font-size: 10.5pt; line-height: 1.52; }
.markdown-body { width: 100%; }
p { margin: 0 0 9pt; }
h1, h2, h3, h4, h5, h6 { color: #16202a; font-family: Helvetica; line-height: 1.18; page-break-after: avoid; }
h1 { font-size: 25pt; margin: 0 0 16pt; padding-bottom: 7pt; border-bottom: 1pt solid #d8dee6; }
h2 { font-size: 17pt; margin: 18pt 0 8pt; padding-bottom: 4pt; border-bottom: 0.5pt solid #e5e9ee; }
h3 { font-size: 13pt; margin: 14pt 0 6pt; }
h4, h5, h6 { font-size: 10.5pt; margin: 11pt 0 5pt; }
a { color: #1769aa; text-decoration: none; }
strong { font-weight: bold; } em { font-style: italic; }
blockquote { margin: 10pt 0 12pt; padding: 7pt 12pt; color: #4d5966; background-color: #f5f7f9; border-left: 3pt solid #7d96ad; }
ul, ol { margin: 3pt 0 10pt; padding-left: 20pt; }
li { margin-bottom: 3pt; }
code { font-family: Courier; font-size: 8.8pt; color: #9b2948; background-color: #f4f5f7; padding: 1pt 2pt; }
pre { margin: 10pt 0 13pt; padding: 10pt 12pt; color: #e8edf2; background-color: #20262e; border-radius: 3pt; white-space: pre-wrap; page-break-inside: avoid; }
pre code { color: #e8edf2; background-color: transparent; padding: 0; }
table { width: 100%; margin: 10pt 0 15pt; border-collapse: collapse; font-size: 9.3pt; }
th { color: #ffffff; background-color: #34495e; font-family: Helvetica; font-weight: bold; text-align: left; padding: 6pt 7pt; border: 0.5pt solid #34495e; }
td { padding: 6pt 7pt; border: 0.5pt solid #d8dee6; }
tr:nth-child(even) { background-color: #f7f9fb; }
hr { margin: 17pt 0; border: 0; border-top: 0.7pt solid #cdd5df; }
img { max-width: 100%; height: auto; margin: 8pt 0 12pt; }`
	if strings.EqualFold(theme, "modern") {
		return strings.Replace(base, "body {", "body { font-family: Helvetica;", 1)
	}
	return strings.Replace(base, "body {", "body { font-family: Times-Roman;", 1)
}
