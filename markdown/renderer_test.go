package markdown

import (
	"strings"
	"testing"
)

func TestRenderDocumentFeatures(t *testing.T) {
	source := "# A useful document\n\nAn **important** paragraph with [a link](https://example.com) and `code`.\n\n> A short quotation.\n\n- First\n- Second\n\n| Name | Value |\n| --- | --- |\n| Alpha | 42 |\n\n~~~go\nfmt.Println(\"hello\")\n~~~"
	got := Render(source)
	for _, want := range []string{
		"<title>A useful document</title>", `>A useful document</h1>`,
		"<strong>important</strong>", `<a href="https://example.com">a link</a>`,
		"<blockquote>", "<ul>", "<table>",
		`<code class="language-go">fmt.Println(&quot;hello&quot;)`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered HTML missing %q\n%s", want, got)
		}
	}
}

func TestRenderEscapesHTML(t *testing.T) {
	got := Render("# <script>alert(1)</script>\n\n![portrait](face.png)")
	if strings.Contains(got, "<script>") {
		t.Fatal("Markdown HTML was not escaped")
	}
	if !strings.Contains(got, `src="face.png"`) {
		t.Fatal("image was not rendered")
	}
}

func TestRenderEmptyFenceLanguage(t *testing.T) {
	got := Render("```\nplain\n```")
	if !strings.Contains(got, "<pre><code>plain\n</code></pre>") {
		t.Fatalf("unexpected fenced code output: %s", got)
	}
}
