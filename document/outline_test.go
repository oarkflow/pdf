package document

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/core"
)

func TestOutlineBuildEmpty(t *testing.T) {
	o := &Outline{}
	w := NewWriter()
	if _, err := w.AddPage(&Page{Size: A4}); err != nil {
		t.Fatal(err)
	}
	num := o.Build(w)
	if num != 0 {
		t.Fatalf("expected 0 for empty outline, got %d", num)
	}

	// Write and check no /Outlines in catalog.
	var buf bytes.Buffer
	w.WriteTo(&buf)
	if strings.Contains(buf.String(), "/Outlines") {
		t.Fatal("empty outline should not produce /Outlines in catalog")
	}
}

func TestOutlineBuildSingleLevel(t *testing.T) {
	w := NewWriter()
	if _, err := w.AddPage(&Page{Size: A4}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.AddPage(&Page{Size: A4}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.AddPage(&Page{Size: A4}); err != nil {
		t.Fatal(err)
	}

	o := &Outline{
		Items: []OutlineItem{
			{Title: "Chapter 1", Page: 0},
			{Title: "Chapter 2", Page: 1},
			{Title: "Chapter 3", Page: 2},
		},
	}
	rootNum := o.Build(w)
	if rootNum == 0 {
		t.Fatal("expected non-zero outline root")
	}
	w.SetOutlines(rootNum)

	var buf bytes.Buffer
	w.WriteTo(&buf)
	pdf := buf.String()

	// Verify outline root has /Type /Outlines
	if !strings.Contains(pdf, "/Type /Outlines") {
		t.Error("missing /Type /Outlines")
	}
	// Verify titles are present
	for _, title := range []string{"Chapter 1", "Chapter 2", "Chapter 3"} {
		if !strings.Contains(pdf, "("+title+")") {
			t.Errorf("missing title %q", title)
		}
	}
	// Verify catalog has /Outlines
	if !strings.Contains(pdf, "/Outlines") {
		t.Error("catalog missing /Outlines")
	}
}

func TestOutlineBuildNested(t *testing.T) {
	w := NewWriter()
	if _, err := w.AddPage(&Page{Size: A4}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.AddPage(&Page{Size: A4}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.AddPage(&Page{Size: A4}); err != nil {
		t.Fatal(err)
	}

	o := &Outline{
		Items: []OutlineItem{
			{
				Title: "Part 1", Page: 0,
				Children: []OutlineItem{
					{Title: "Section 1.1", Page: 1},
					{Title: "Section 1.2", Page: 2},
				},
			},
		},
	}
	rootNum := o.Build(w)
	if rootNum == 0 {
		t.Fatal("expected non-zero outline root")
	}
	w.SetOutlines(rootNum)

	var buf bytes.Buffer
	w.WriteTo(&buf)
	pdf := buf.String()

	// Check all titles present
	for _, title := range []string{"Part 1", "Section 1.1", "Section 1.2"} {
		if !strings.Contains(pdf, "("+title+")") {
			t.Errorf("missing title %q", title)
		}
	}
	// Root count should be 3 (Part 1 + Section 1.1 + Section 1.2)
	if !strings.Contains(pdf, "/Count 3") {
		t.Error("expected /Count 3 on root")
	}
}

func TestOutlineLinkedListStructure(t *testing.T) {
	w := NewWriter()
	if _, err := w.AddPage(&Page{Size: A4}); err != nil {
		t.Fatal(err)
	}

	o := &Outline{
		Items: []OutlineItem{
			{Title: "A", Page: 0},
			{Title: "B", Page: 0},
			{Title: "C", Page: 0},
		},
	}
	rootNum := o.Build(w)
	w.SetOutlines(rootNum)

	// Find the outline item objects (not root, not page/content objects).
	// Items should have /Prev and /Next appropriately.
	var itemDicts []*core.PdfDictionary
	for _, obj := range w.objects {
		d, ok := obj.Object.(*core.PdfDictionary)
		if !ok {
			continue
		}
		if d.Get("Title") != nil {
			itemDicts = append(itemDicts, d)
		}
	}

	if len(itemDicts) != 3 {
		t.Fatalf("expected 3 outline items, got %d", len(itemDicts))
	}

	// First item: no Prev, has Next
	if itemDicts[0].Get("Prev") != nil {
		t.Error("first item should not have /Prev")
	}
	if itemDicts[0].Get("Next") == nil {
		t.Error("first item should have /Next")
	}

	// Middle item: has both
	if itemDicts[1].Get("Prev") == nil {
		t.Error("middle item should have /Prev")
	}
	if itemDicts[1].Get("Next") == nil {
		t.Error("middle item should have /Next")
	}

	// Last item: has Prev, no Next
	if itemDicts[2].Get("Prev") == nil {
		t.Error("last item should have /Prev")
	}
	if itemDicts[2].Get("Next") != nil {
		t.Error("last item should not have /Next")
	}
}

func TestDocumentAddBookmark(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.NewPage()
	doc.NewPage()
	doc.AddBookmark("Page 1", 0)
	doc.AddBookmark("Page 2", 1)

	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	pdf := buf.String()
	if !strings.Contains(pdf, "/Outlines") {
		t.Error("PDF should contain /Outlines")
	}
	if !strings.Contains(pdf, "(Page 1)") || !strings.Contains(pdf, "(Page 2)") {
		t.Error("PDF should contain bookmark titles")
	}
}

func TestDocumentAddBookmarkWithChildren(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.NewPage()
	doc.NewPage()
	doc.NewPage()
	doc.AddBookmarkWithChildren("Part 1", 0, []OutlineItem{
		{Title: "Chapter 1", Page: 1},
		{Title: "Chapter 2", Page: 2},
	})

	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	pdf := buf.String()
	for _, title := range []string{"Part 1", "Chapter 1", "Chapter 2"} {
		if !strings.Contains(pdf, "("+title+")") {
			t.Errorf("missing bookmark %q", title)
		}
	}
}
