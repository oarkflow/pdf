package reader

import (
	"os"
	"path/filepath"
	"testing"
)

// minimalPDF returns a valid single-page PDF with the given text.
func minimalPDF(text string) []byte {
	// A minimal valid PDF with one page containing the given text.
	content := "BT /F1 12 Tf 72 720 Td (" + text + ") Tj ET"
	// Build PDF by hand.
	pdf := "%PDF-1.4\n"

	// Object 1: Catalog
	pdf += "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n"

	// Object 2: Pages
	pdf += "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n"

	// Object 3: Page
	pdf += "3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n"

	// Object 4: Content stream
	pdf += "4 0 obj\n<< /Length " + itoa(len(content)) + " >>\nstream\n" + content + "\nendstream\nendobj\n"

	// Object 5: Font
	pdf += "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n"

	// xref (simplified - offsets don't need to be perfect for our parser)
	xrefOff := len(pdf)
	pdf += "xref\n0 6\n0000000000 65535 f \r\n"
	pdf += "0000000009 00000 n \r\n"
	pdf += "0000000058 00000 n \r\n"
	pdf += "0000000115 00000 n \r\n"
	// Approximate offsets - recalculate properly
	pdf += "0000000300 00000 n \r\n"
	pdf += "0000000400 00000 n \r\n"

	pdf += "trailer\n<< /Size 6 /Root 1 0 R >>\n"
	pdf += "startxref\n" + itoa(xrefOff) + "\n%%EOF\n"

	return []byte(pdf)
}

func itoa(n int) string {
	s := ""
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func TestMergeTwoPDFs(t *testing.T) {
	pdf1 := minimalPDF("Hello from PDF 1")
	pdf2 := minimalPDF("Hello from PDF 2")

	merged, err := Merge([][]byte{pdf1, pdf2})
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Verify we can read the merged PDF and it has 2 pages.
	r, err := Open(merged)
	if err != nil {
		t.Fatalf("Opening merged PDF failed: %v", err)
	}
	if r.NumPages() != 2 {
		t.Fatalf("expected 2 pages, got %d", r.NumPages())
	}
}

func TestMergeSinglePDF(t *testing.T) {
	pdf1 := minimalPDF("Single PDF")

	merged, err := Merge([][]byte{pdf1})
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	r, err := Open(merged)
	if err != nil {
		t.Fatalf("Opening merged PDF failed: %v", err)
	}
	if r.NumPages() != 1 {
		t.Fatalf("expected 1 page, got %d", r.NumPages())
	}
}

func TestMergeNoPDFs(t *testing.T) {
	_, err := Merge(nil)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestMergeFilesNonexistent(t *testing.T) {
	err := MergeFiles([]string{"/nonexistent/file.pdf"}, "/tmp/out.pdf")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestMergeFiles(t *testing.T) {
	dir := t.TempDir()

	// Write two test PDFs
	p1 := filepath.Join(dir, "a.pdf")
	p2 := filepath.Join(dir, "b.pdf")
	out := filepath.Join(dir, "merged.pdf")

	os.WriteFile(p1, minimalPDF("File A"), 0644)
	os.WriteFile(p2, minimalPDF("File B"), 0644)

	err := MergeFiles([]string{p1, p2}, out)
	if err != nil {
		t.Fatalf("MergeFiles failed: %v", err)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	r, err := Open(data)
	if err != nil {
		t.Fatalf("opening merged: %v", err)
	}
	if r.NumPages() != 2 {
		t.Fatalf("expected 2 pages, got %d", r.NumPages())
	}
}
