package md

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

func TestConvertFormats(t *testing.T) {
	in := []byte("# Hello\n\n- A\n- B\n")
	for _, f := range []Format{PDF, DOCX, HTML} {
		b, err := Convert(in, f, Options{Title: "Hello"})
		if err != nil {
			t.Fatal(err)
		}
		if len(b) == 0 {
			t.Fatalf("empty %s", f)
		}
	}
}

func TestHorizontalRuleDoesNotRenderVisibleLine(t *testing.T) {
	input := []byte("Intro\n\n---\n\nBody\n")
	htmlOut, err := Convert(input, HTML, Options{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(htmlOut, []byte("<hr")) || bytes.Contains(htmlOut, []byte("border-top")) {
		t.Fatalf("html rendered a visible horizontal rule: %s", string(htmlOut))
	}
	docxOut, err := Convert(input, DOCX, Options{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	documentXML := readDocxPart(t, docxOut, "word/document.xml")
	if bytes.Contains(documentXML, []byte("<w:pBdr>")) || bytes.Contains(documentXML, []byte("w:bottom w:val=\"single\"")) {
		t.Fatal("docx rendered a visible horizontal rule")
	}
	pdfOut, err := Convert(input, PDF, Options{Title: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(pdfOut, []byte(" m ")) && bytes.Contains(pdfOut, []byte(" l S")) {
		t.Fatal("pdf rendered line drawing operators for a horizontal rule")
	}
}

func readDocxPart(t *testing.T, docx []byte, name string) []byte {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			if err != nil {
				t.Fatal(err)
			}
			return b
		}
	}
	t.Fatalf("missing docx part %s", name)
	return nil
}

func TestPDFRendersCareStyleHeadingAndTOCLines(t *testing.T) {
	input := []byte("# Main\n\n## Section\n\nContent\n\n## Next\n\nMore content\n")
	pdfOut, err := Convert(input, PDF, Options{Title: "x", TOC: true})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(pdfOut, []byte(" l S")) {
		t.Fatal("pdf should render subtle CARE-style heading/TOC separator lines")
	}
}

func TestPDFTOCEntriesAreClickable(t *testing.T) {
	input := []byte("# Main\n\n## Section One\n\nContent\n\n## Section Two\n\nMore content\n")
	pdfOut, err := Convert(input, PDF, Options{Title: "x", TOC: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range [][]byte{[]byte("/Subtype /Link"), []byte("/Annots ["), []byte("/Dest [")} {
		if !bytes.Contains(pdfOut, needle) {
			t.Fatalf("pdf TOC is not clickable; missing %s", needle)
		}
	}
}
