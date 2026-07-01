package markdown

import (
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	d, err := NewParser(Options{}).Parse(strings.NewReader("# Title\n\nText\n\n- A\n- B\n\n| A | B |\n|---|---|\n| 1 | 2 |\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Nodes) < 4 {
		t.Fatalf("nodes=%d", len(d.Nodes))
	}
	if d.Nodes[0].Kind != Heading {
		t.Fatalf("first node not heading: %#v", d.Nodes[0])
	}
	foundList, foundTable := false, false
	for _, n := range d.Nodes {
		if n.Kind == BulletList {
			foundList = true
		}
		if n.Kind == Table {
			foundTable = true
			if len(n.Rows) != 2 {
				t.Fatalf("table separator leaked into rows: %#v", n.Rows)
			}
		}
	}
	if !foundList || !foundTable {
		t.Fatalf("unexpected nodes: %#v", d.Nodes)
	}
}

func BenchmarkParser(b *testing.B) {
	md := strings.Repeat("# Heading\n\nSome paragraph with text and more text.\n\n- one\n- two\n\n", 1000)
	p := NewParser(Options{})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := p.Parse(strings.NewReader(md)); err != nil {
			b.Fatal(err)
		}
	}
}

func TestParseModernFeatures(t *testing.T) {
	input := strings.Join([]string{
		"---",
		"title: Demo",
		"author: Team",
		"---",
		"# Hello World {#hello}",
		"",
		"> [!WARNING] Pay attention",
		"> second line",
		"",
		"- [x] Done",
		"- [ ] Todo",
		"",
		"| A \\| B | C |",
		"| :--- | ---: |",
		"| 1 | 2 |",
	}, "\n")
	d, err := NewParser(Options{}).Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if d.Meta["title"] != "Demo" || d.Meta["author"] != "Team" {
		t.Fatalf("front matter not parsed: %#v", d.Meta)
	}
	if len(d.Headings) != 1 || d.Headings[0].ID != "hello" {
		t.Fatalf("heading ref not parsed: %#v", d.Headings)
	}
	foundAlert, foundTasks := false, false
	for _, n := range d.Nodes {
		if n.Kind == Alert && n.Info == "warning" && strings.Contains(n.Text, "second line") {
			foundAlert = true
		}
		if n.Kind == BulletList && len(n.Children) == 2 && n.Children[0].Checked != nil && *n.Children[0].Checked {
			foundTasks = true
		}
		if n.Kind == Table && n.Rows[0][0] != "A | B" {
			t.Fatalf("escaped pipe not handled: %#v", n.Rows)
		}
	}
	if !foundAlert || !foundTasks {
		t.Fatalf("missing alert=%v tasks=%v nodes=%#v", foundAlert, foundTasks, d.Nodes)
	}
}

func TestTableSeparatorIsNotRenderedAsData(t *testing.T) {
	d, err := NewParser(Options{}).Parse(strings.NewReader("| A | B |\n| --- | ---: |\n| one | two |\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Nodes) != 1 || d.Nodes[0].Kind != Table {
		t.Fatalf("expected one table node, got %#v", d.Nodes)
	}
	rows := d.Nodes[0].Rows
	if len(rows) != 2 {
		t.Fatalf("expected header and one body row, got %d rows: %#v", len(rows), rows)
	}
	if rows[1][0] != "one" || rows[1][1] != "two" {
		t.Fatalf("unexpected body row: %#v", rows[1])
	}
	if len(d.Nodes[0].Aligns) != 2 || d.Nodes[0].Aligns[1] != AlignRight {
		t.Fatalf("expected right alignment in second column, got %#v", d.Nodes[0].Aligns)
	}
}

func TestProductionMarkdownCoverage(t *testing.T) {
	input := strings.Join([]string{
		"Setext Title",
		"============",
		"",
		"Paragraph with [reference][ref], <https://example.com>, footnote[^a], ~~strike~~ and `code`.",
		"",
		"> Quote intro",
		">",
		"> - nested bullet",
		">   1. nested ordered",
		"",
		"    indented code",
		"",
		"- parent",
		"  continuation paragraph",
		"  - child",
		"",
		"Term",
		": Definition text",
		"",
		"<div>raw</div>",
		"",
		"[ref]: https://example.com \"Example\"",
		"[^a]: Footnote body",
	}, "\n")
	d, err := NewParser(Options{}).Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Headings) != 1 || d.Headings[0].Level != 1 || d.Headings[0].ID != "setext-title" {
		t.Fatalf("setext heading not parsed: %#v", d.Headings)
	}
	if ref := d.Refs["ref"]; ref.URL != "https://example.com" || ref.Title != "Example" {
		t.Fatalf("reference definition missing: %#v", d.Refs)
	}
	var haveBQ, haveNestedList, haveCode, haveDef, haveHTML, haveFoot bool
	var walk func([]Node)
	walk = func(nodes []Node) {
		for _, n := range nodes {
			switch n.Kind {
			case BlockQuote:
				haveBQ = len(n.Children) > 0
			case BulletList:
				if len(n.Children) > 0 && len(n.Children[0].Children) > 0 {
					haveNestedList = true
				}
			case CodeBlock:
				haveCode = haveCode || strings.Contains(n.Text, "indented code")
			case DefinitionList:
				haveDef = true
			case HTMLBlock:
				haveHTML = true
			case FootnoteList:
				haveFoot = true
			}
			if len(n.Children) > 0 {
				walk(n.Children)
			}
		}
	}
	walk(d.Nodes)
	if !haveBQ || !haveNestedList || !haveCode || !haveDef || !haveHTML || !haveFoot {
		t.Fatalf("coverage missing bq=%v nested=%v code=%v def=%v html=%v foot=%v nodes=%#v", haveBQ, haveNestedList, haveCode, haveDef, haveHTML, haveFoot, d.Nodes)
	}
}
