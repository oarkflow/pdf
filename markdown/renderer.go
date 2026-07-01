package markdown

import (
	"html"
	"strings"

	"github.com/oarkflow/pdf/md"
)

type Options struct {
	Title      string
	Theme      string
	Stylesheet string
}

func Render(source string, opts ...Options) string {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	title := opt.Title
	if title == "" {
		title = extractTitle(source)
	}
	if title == "" {
		title = "Document"
	}

	htmlBytes, err := md.Convert([]byte(source), md.HTML, md.Options{
		Title: title,
		Theme: opt.Theme,
		CSS:   opt.Stylesheet,
	})
	if err != nil {
		return "<p>" + html.EscapeString(source) + "</p>"
	}

	return string(htmlBytes)
}

func extractTitle(source string) string {
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		n := 0
		for n < len(trimmed) && trimmed[n] == '#' && n < 6 {
			n++
		}
		if n > 0 && n < len(trimmed) && (trimmed[n] == ' ' || trimmed[n] == '\t') {
			return strings.TrimSpace(trimmed[n+1:])
		}
	}
	return ""
}
