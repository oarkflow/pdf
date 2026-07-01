package export

import (
	"html"
	"regexp"
	"strings"

	"github.com/oarkflow/pdf/md/internal/markdown"
)

var (
	imgRe        = regexp.MustCompile(`!\[([^\]]*)\]\(([^\s)]+)(?:\s+"([^"]*)")?\)`)
	linkRe       = regexp.MustCompile(`\[([^\]]+)\]\(([^\s)]+)(?:\s+"([^"]*)")?\)`)
	refLinkRe    = regexp.MustCompile(`\[([^\]]+)\]\[([^\]]*)\]`)
	autoLinkRe   = regexp.MustCompile(`&lt;(https?://[^\s]+|mailto:[^\s]+)&gt;`)
	footRefRe    = regexp.MustCompile(`\[\^([^\]]+)\]`)
	strongRe     = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)
	emphasisRe   = regexp.MustCompile(`\*([^*]+)\*|_([^_]+)_`)
	codeInlineRe = regexp.MustCompile("`([^`]+)`")
	strikeRe     = regexp.MustCompile(`~~([^~]+)~~`)
)

func inlineHTML(s string) string { return inlineHTMLRefs(s, nil) }

func inlineHTMLRefs(s string, refs map[string]markdown.LinkRef) string {
	esc := html.EscapeString(s)
	esc = imgRe.ReplaceAllString(esc, `<img src="$2" alt="$1" loading="lazy">`)
	esc = linkRe.ReplaceAllString(esc, `<a href="$2">$1</a>`)
	if len(refs) > 0 {
		esc = refLinkRe.ReplaceAllStringFunc(esc, func(m string) string {
			sub := refLinkRe.FindStringSubmatch(m)
			if len(sub) != 3 {
				return m
			}
			key := sub[2]
			if key == "" {
				key = sub[1]
			}
			if ref, ok := refs[strings.ToLower(key)]; ok {
				title := ""
				if ref.Title != "" {
					title = ` title="` + attrEsc(ref.Title) + `"`
				}
				return `<a href="` + attrEsc(ref.URL) + `"` + title + `>` + sub[1] + `</a>`
			}
			return m
		})
	}
	esc = autoLinkRe.ReplaceAllString(esc, `<a href="$1">$1</a>`)
	esc = footRefRe.ReplaceAllString(esc, `<sup id="fnref-$1"><a href="#fn-$1">$1</a></sup>`)
	// Apply code before emphasis so markers inside code are not interpreted.
	esc = codeInlineRe.ReplaceAllString(esc, `<code>$1</code>`)
	esc = strongRe.ReplaceAllString(esc, `<strong>$1$2</strong>`)
	esc = emphasisRe.ReplaceAllString(esc, `<em>$1$2</em>`)
	esc = strikeRe.ReplaceAllString(esc, `<del>$1</del>`)
	esc = strings.ReplaceAll(esc, "  \n", "<br>")
	esc = strings.ReplaceAll(esc, "\n", "\n")
	return esc
}

func plainInline(s string) string { return plainInlineRefs(s, nil) }

func plainInlineRefs(s string, refs map[string]markdown.LinkRef) string {
	s = imgRe.ReplaceAllString(s, "$1")
	s = linkRe.ReplaceAllString(s, "$1 ($2)")
	if len(refs) > 0 {
		s = refLinkRe.ReplaceAllStringFunc(s, func(m string) string {
			sub := refLinkRe.FindStringSubmatch(m)
			if len(sub) != 3 {
				return m
			}
			key := sub[2]
			if key == "" {
				key = sub[1]
			}
			if ref, ok := refs[strings.ToLower(key)]; ok {
				return sub[1] + " (" + ref.URL + ")"
			}
			return sub[1]
		})
	}
	s = footRefRe.ReplaceAllString(s, "$1")
	s = strings.NewReplacer("**", "", "__", "", "`", "", "~~", "", "*", "", "_", "").Replace(s)
	return s
}

func attrEsc(s string) string { return html.EscapeString(s) }
