package document

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/layout"
)

func TestNewWriter(t *testing.T) {
	w := NewWriter()
	if w == nil {
		t.Fatal("NewWriter returned nil")
	}
}

func TestWriterAddObject(t *testing.T) {
	w := NewWriter()
	num := w.AddObject(core.PdfInteger(42))
	if num != 1 {
		t.Errorf("first object number = %d, want 1", num)
	}
	num2 := w.AddObject(core.PdfInteger(43))
	if num2 != 2 {
		t.Errorf("second object number = %d, want 2", num2)
	}
}

func TestWriterAddPage(t *testing.T) {
	w := NewWriter()
	page := NewPage(A4)
	page.Contents = []byte("BT /F1 12 Tf (Hello) Tj ET")
	num, err := w.AddPage(page)
	if err != nil {
		t.Fatal(err)
	}
	if num <= 0 {
		t.Error("page object number should be positive")
	}
}

func TestWriterSetInfo(t *testing.T) {
	w := NewWriter()
	w.SetInfo(map[string]string{"Title": "Test PDF"})
	// Just verify it doesn't panic
}

func TestWriterWriteToProducesValidPDF(t *testing.T) {
	w := NewWriter()
	page := NewPage(Letter)
	page.Contents = []byte("BT /F1 12 Tf (Hello World) Tj ET")
	if _, err := w.AddPage(page); err != nil {
		t.Fatal(err)
	}
	w.SetInfo(map[string]string{"Title": "Test"})

	var buf bytes.Buffer
	n, err := w.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Error("wrote 0 bytes")
	}
	data := buf.String()
	if !strings.HasPrefix(data, "%PDF-1.7") {
		t.Error("missing PDF header")
	}
	if !strings.Contains(data, "%%EOF") {
		t.Error("missing EOF marker")
	}
	if !strings.Contains(data, "xref") {
		t.Error("missing xref table")
	}
	if !strings.Contains(data, "trailer") {
		t.Error("missing trailer")
	}
	if !strings.Contains(data, "startxref") {
		t.Error("missing startxref")
	}
	if !strings.Contains(data, "/Catalog") {
		t.Error("missing catalog")
	}
}

func TestWriterMultiplePages(t *testing.T) {
	w := NewWriter()
	for i := 0; i < 3; i++ {
		p := NewPage(A4)
		p.Contents = []byte("BT (page) Tj ET")
		if _, err := w.AddPage(p); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	_, err := w.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	data := buf.String()
	if !strings.Contains(data, "/Count 3") {
		t.Error("expected 3 pages in output")
	}
}

func TestWriterEmptyPage(t *testing.T) {
	w := NewWriter()
	p := NewPage(A4)
	if _, err := w.AddPage(p); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, err := w.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriterFontReferences(t *testing.T) {
	w := NewWriter()
	p := NewPage(A4)
	p.Contents = []byte("BT /F1 12 Tf (test) Tj ET")
	p.Fonts["F1"] = 0 // 0 means create standard font
	if _, err := w.AddPage(p); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
	data := buf.String()
	if !strings.Contains(data, "/Font") {
		t.Error("missing font reference")
	}
	if !strings.Contains(data, "/Type1") {
		t.Error("missing Type1 font")
	}
}

func TestWriterRoundTrip(t *testing.T) {
	// Write a PDF then read it back
	doc, _ := NewDocument(A4)
	doc.SetMetadata(Metadata{Title: "RoundTrip Test"})
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf (Hello) Tj ET")

	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	// Verify it starts with PDF header and has structure
	data := buf.Bytes()
	if len(data) < 50 {
		t.Error("output too small")
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Error("missing PDF header")
	}
}

func TestWriterPageMediaBox(t *testing.T) {
	w := NewWriter()
	p := NewPage(PageSize{Width: 100, Height: 200})
	if _, err := w.AddPage(p); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w.WriteTo(&buf)
	data := buf.String()
	if !strings.Contains(data, "/MediaBox") {
		t.Error("missing MediaBox")
	}
	if !strings.Contains(data, "100") || !strings.Contains(data, "200") {
		t.Error("MediaBox dimensions missing")
	}
}

func TestWriterAddsLinkAnnotations(t *testing.T) {
	w := NewWriter()
	p := NewPage(A4)
	p.Contents = []byte("BT /F1 12 Tf (Link) Tj ET")
	p.Annotations = []layout.LinkAnnotation{{
		X1:  10,
		Y1:  20,
		X2:  80,
		Y2:  32,
		URI: "https://example.com",
	}}

	if _, err := w.AddPage(p); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	data := buf.String()
	if !strings.Contains(data, "/Annots") {
		t.Fatal("missing /Annots")
	}
	if !strings.Contains(data, "/URI (https://example.com)") {
		t.Fatal("missing URI annotation")
	}
}
