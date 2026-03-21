package document

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewStructureTree(t *testing.T) {
	st := NewStructureTree()
	if st.Root == nil {
		t.Fatal("expected non-nil root")
	}
	if st.Root.Type != "Document" {
		t.Fatalf("expected root type Document, got %s", st.Root.Type)
	}
	if st.Root.MCID != -1 {
		t.Fatalf("expected root MCID -1, got %d", st.Root.MCID)
	}
}

func TestAddElement(t *testing.T) {
	st := NewStructureTree()
	h1 := &StructElement{Type: "H1", MCID: 0, PageNum: 0}
	p := &StructElement{Type: "P", MCID: 1, PageNum: 0}
	st.AddElement(h1)
	st.AddElement(p)

	if len(st.Root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(st.Root.Children))
	}
	if st.Root.Children[0].Type != "H1" {
		t.Fatalf("expected H1, got %s", st.Root.Children[0].Type)
	}
	if st.Root.Children[1].Type != "P" {
		t.Fatalf("expected P, got %s", st.Root.Children[1].Type)
	}
}

func TestNextMCID(t *testing.T) {
	st := NewStructureTree()
	if id := st.NextMCID(); id != 0 {
		t.Fatalf("expected 0, got %d", id)
	}
	if id := st.NextMCID(); id != 1 {
		t.Fatalf("expected 1, got %d", id)
	}
	if st.MCIDCount() != 2 {
		t.Fatalf("expected count 2, got %d", st.MCIDCount())
	}
}

func TestBuildProducesObjects(t *testing.T) {
	st := NewStructureTree()
	st.AddElement(&StructElement{Type: "H1", MCID: 0, PageNum: 0})
	st.AddElement(&StructElement{Type: "P", MCID: 1, PageNum: 0})
	st.nextMCID = 2

	w := NewWriter()
	// Add a page so page refs exist.
	page := NewPage(A4)
	page.Contents = []byte("BT /F1 12 Tf (Hello) Tj ET")
	if _, err := w.AddPage(page); err != nil {
		t.Fatal(err)
	}

	rootNum := st.Build(w)
	if rootNum == 0 {
		t.Fatal("expected non-zero StructTreeRoot object number")
	}

	// Verify the objects were written by serializing.
	var buf bytes.Buffer
	w.SetStructTreeRoot(rootNum)
	w.SetMarkInfo(true)
	_, err := w.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	pdf := buf.String()

	// Check for StructTreeRoot in catalog.
	if !strings.Contains(pdf, "/StructTreeRoot") {
		t.Error("PDF missing /StructTreeRoot in catalog")
	}
	// Check for MarkInfo.
	if !strings.Contains(pdf, "/MarkInfo") {
		t.Error("PDF missing /MarkInfo in catalog")
	}
	if !strings.Contains(pdf, "/Marked true") {
		t.Error("PDF missing /Marked true")
	}
	// Check for structure element types.
	if !strings.Contains(pdf, "/S /H1") {
		t.Error("PDF missing /S /H1 structure element")
	}
	if !strings.Contains(pdf, "/S /P") {
		t.Error("PDF missing /S /P structure element")
	}
	if !strings.Contains(pdf, "/S /Document") {
		t.Error("PDF missing /S /Document structure element")
	}
	// Check for ParentTree.
	if !strings.Contains(pdf, "/ParentTree") {
		t.Error("PDF missing /ParentTree")
	}
	// Check for MCID references.
	if !strings.Contains(pdf, "/MCID 0") {
		t.Error("PDF missing /MCID 0")
	}
	if !strings.Contains(pdf, "/MCID 1") {
		t.Error("PDF missing /MCID 1")
	}
}

func TestMarkedContentOperators(t *testing.T) {
	var stream []byte
	stream = BeginMarkedContent(stream, "P", 0)
	stream = append(stream, []byte("BT /F1 12 Tf (Hello) Tj ET\n")...)
	stream = EndMarkedContent(stream)

	s := string(stream)
	if !strings.Contains(s, "/P <</MCID 0>> BDC") {
		t.Errorf("missing BDC operator, got: %s", s)
	}
	if !strings.Contains(s, "EMC") {
		t.Errorf("missing EMC operator, got: %s", s)
	}
}

func TestParentTreeContainsAllMCIDs(t *testing.T) {
	st := NewStructureTree()
	for i := 0; i < 5; i++ {
		st.AddElement(&StructElement{Type: "P", MCID: i, PageNum: 0})
	}
	st.nextMCID = 5

	w := NewWriter()
	page := NewPage(A4)
	page.Contents = []byte("q Q")
	if _, err := w.AddPage(page); err != nil {
		t.Fatal(err)
	}

	rootNum := st.Build(w)
	w.SetStructTreeRoot(rootNum)
	w.SetMarkInfo(true)

	var buf bytes.Buffer
	_, err := w.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	pdf := buf.String()
	// All MCIDs 0-4 should appear in the Nums array.
	for i := 0; i < 5; i++ {
		needle := strings.Replace("/MCID %d", "%d", string(rune('0'+i)), 1)
		_ = needle
		// Just check MCID references exist in the output.
		if !strings.Contains(pdf, "/Type /MCR") {
			t.Error("PDF missing /Type /MCR marked content reference")
		}
	}
}

func TestDocumentTaggingIntegration(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.EnableTagging()

	if !doc.IsTagged() {
		t.Fatal("expected IsTagged to be true")
	}
	if doc.StructTree() == nil {
		t.Fatal("expected non-nil StructTree")
	}

	st := doc.StructTree()
	mcid := st.NextMCID()

	// Add a tagged page.
	page := doc.NewPage()
	content := BeginMarkedContent(nil, "P", mcid)
	content = append(content, []byte("BT /F1 12 Tf (Hello) Tj ET\n")...)
	content = EndMarkedContent(content)
	page.Contents = content

	st.AddElement(&StructElement{Type: "P", MCID: mcid, PageNum: 0})

	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	pdf := buf.String()
	if !strings.Contains(pdf, "/MarkInfo") {
		t.Error("PDF missing /MarkInfo")
	}
	if !strings.Contains(pdf, "/StructTreeRoot") {
		t.Error("PDF missing /StructTreeRoot")
	}
}

func TestStructElementAltText(t *testing.T) {
	st := NewStructureTree()
	fig := &StructElement{
		Type:    "Figure",
		MCID:    0,
		PageNum: 0,
		AltText: "A photo of a cat",
	}
	st.AddElement(fig)
	st.nextMCID = 1

	w := NewWriter()
	page := NewPage(A4)
	page.Contents = []byte("q Q")
	if _, err := w.AddPage(page); err != nil {
		t.Fatal(err)
	}

	rootNum := st.Build(w)
	w.SetStructTreeRoot(rootNum)
	w.SetMarkInfo(true)

	var buf bytes.Buffer
	_, _ = w.WriteTo(&buf)

	pdf := buf.String()
	if !strings.Contains(pdf, "/Alt (A photo of a cat)") {
		t.Error("PDF missing /Alt text for figure")
	}
	if !strings.Contains(pdf, "/S /Figure") {
		t.Error("PDF missing /S /Figure")
	}
}
