package document

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStreamingWriter_SinglePage(t *testing.T) {
	var buf bytes.Buffer
	sw, err := NewStreamingWriter(&buf)
	if err != nil {
		t.Fatalf("NewStreamingWriter: %v", err)
	}

	page := NewPage(A4)
	page.Contents = []byte("BT /F1 12 Tf 100 700 Td (Hello World) Tj ET")
	page.Fonts["F1"] = 0 // will create standard font

	if _, err := sw.AddPage(page); err != nil {
		t.Fatalf("AddPage: %v", err)
	}

	if err := sw.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	output := buf.String()
	assertValidPDF(t, output)
}

func TestStreamingWriter_MultiPage(t *testing.T) {
	var buf bytes.Buffer
	sw, err := NewStreamingWriter(&buf)
	if err != nil {
		t.Fatalf("NewStreamingWriter: %v", err)
	}

	for i := 0; i < 10; i++ {
		page := NewPage(A4)
		page.Contents = []byte(fmt.Sprintf("BT /F1 12 Tf 100 700 Td (Page %d) Tj ET", i+1))
		page.Fonts["F1"] = 0
		if _, err := sw.AddPage(page); err != nil {
			t.Fatalf("AddPage %d: %v", i, err)
		}
	}

	if err := sw.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	output := buf.String()
	assertValidPDF(t, output)

	// Check page count.
	if !strings.Contains(output, "/Count 10") {
		t.Error("expected /Count 10 in pages dict")
	}
}

func TestStreamingWriter_WithInfo(t *testing.T) {
	var buf bytes.Buffer
	sw, err := NewStreamingWriter(&buf)
	if err != nil {
		t.Fatalf("NewStreamingWriter: %v", err)
	}

	sw.SetInfo(map[string]string{
		"Title":  "Test Document",
		"Author": "Test Suite",
	})

	page := NewPage(A4)
	page.Contents = []byte("BT /F1 12 Tf 100 700 Td (Test) Tj ET")
	page.Fonts["F1"] = 0
	if _, err := sw.AddPage(page); err != nil {
		t.Fatalf("AddPage: %v", err)
	}

	if err := sw.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	output := buf.String()
	assertValidPDF(t, output)

	if !strings.Contains(output, "Test Document") {
		t.Error("expected document title in output")
	}
}

func TestStreamingWriter_ToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}

	sw, err := NewStreamingWriter(f)
	if err != nil {
		f.Close()
		t.Fatalf("NewStreamingWriter: %v", err)
	}

	page := NewPage(A4)
	page.Contents = []byte("BT /F1 12 Tf 100 700 Td (File Test) Tj ET")
	page.Fonts["F1"] = 0
	if _, err := sw.AddPage(page); err != nil {
		f.Close()
		t.Fatalf("AddPage: %v", err)
	}

	if err := sw.Finish(); err != nil {
		f.Close()
		t.Fatalf("Finish: %v", err)
	}
	f.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	assertValidPDF(t, string(data))
}

func TestStreamingWriter_FinishTwice(t *testing.T) {
	var buf bytes.Buffer
	sw, err := NewStreamingWriter(&buf)
	if err != nil {
		t.Fatalf("NewStreamingWriter: %v", err)
	}

	page := NewPage(A4)
	page.Contents = []byte("BT ET")
	if _, err := sw.AddPage(page); err != nil {
		t.Fatalf("AddPage: %v", err)
	}

	if err := sw.Finish(); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	if err := sw.Finish(); err == nil {
		t.Error("expected error on second Finish call")
	}
}

func TestDocument_WriteStreamingTo(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetMetadata(Metadata{Title: "Streaming Test"})

	page := NewPage(A4)
	page.Contents = []byte("BT /F1 12 Tf 100 700 Td (Streaming) Tj ET")
	page.Fonts["F1"] = 0
	doc.AddPage(page)

	var buf bytes.Buffer
	if err := doc.WriteStreamingTo(&buf); err != nil {
		t.Fatalf("WriteStreamingTo: %v", err)
	}

	assertValidPDF(t, buf.String())
}

func assertValidPDF(t *testing.T, output string) {
	t.Helper()

	if !strings.HasPrefix(output, "%PDF-1.7") {
		t.Error("missing PDF header")
	}
	if !strings.Contains(output, "xref") {
		t.Error("missing xref table")
	}
	if !strings.Contains(output, "trailer") {
		t.Error("missing trailer")
	}
	if !strings.HasSuffix(output, "%%EOF\n") {
		t.Error("missing EOF marker")
	}
	if !strings.Contains(output, "startxref") {
		t.Error("missing startxref")
	}
}
