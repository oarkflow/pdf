package document

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
)

func TestPDFA1b_XMPContainsPart1(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetMetadata(Metadata{Title: "Test", Author: "Tester", Producer: "TestLib"})
	doc.SetPDFA(PDFA1b)
	doc.NewPage()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "<pdfaid:part>1</pdfaid:part>") {
		t.Error("expected pdfaid:part=1 in XMP metadata")
	}
	if !strings.Contains(out, "<pdfaid:conformance>B</pdfaid:conformance>") {
		t.Error("expected pdfaid:conformance=B in XMP metadata")
	}
}

func TestPDFA2b_XMPContainsPart2(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetMetadata(Metadata{Title: "Test2"})
	doc.SetPDFA(PDFA2b)
	doc.NewPage()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "<pdfaid:part>2</pdfaid:part>") {
		t.Error("expected pdfaid:part=2 in XMP metadata")
	}
}

func TestPDFA_OutputIntents(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetPDFA(PDFA1b)
	doc.NewPage()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "/OutputIntents") {
		t.Error("expected /OutputIntents in catalog")
	}
	if !strings.Contains(out, "/GTS_PDFA1") {
		t.Error("expected /GTS_PDFA1 output intent subtype")
	}
}

func TestPDFA_XMPWellFormedXML(t *testing.T) {
	xmp := string(buildPDFAXMPMetadata(Metadata{Title: "Hello <World> & \"Friends\"", Author: "A&B"}, 1, "B"))

	decoder := xml.NewDecoder(strings.NewReader(xmp))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("XMP is not well-formed XML: %v", err)
		}
	}
}

func TestPDFA_DocumentID(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetPDFA(PDFA1b)
	doc.NewPage()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "/ID") {
		t.Error("expected /ID in trailer for PDF/A")
	}
}

func TestPDFA_MetadataStream(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetPDFA(PDFA1b)
	doc.NewPage()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "/Metadata") {
		t.Error("expected /Metadata reference in catalog")
	}
	if !strings.Contains(out, "/Subtype /XML") {
		t.Error("expected metadata stream with /Subtype /XML")
	}
}

func TestSRGBICCProfile(t *testing.T) {
	profile := sRGBICCProfile()
	if len(profile) < 128 {
		t.Fatal("ICC profile too short")
	}
	// Check signature at offset 36.
	sig := string(profile[36:40])
	if sig != "acsp" {
		t.Errorf("expected 'acsp' signature, got %q", sig)
	}
	// Check color space is RGB.
	cs := string(profile[16:20])
	if cs != "RGB " {
		t.Errorf("expected 'RGB ' color space, got %q", cs)
	}
}

func TestNoPDFA_NoMetadata(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.NewPage()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Contains(out, "/OutputIntents") {
		t.Error("should not have /OutputIntents without PDF/A")
	}
}
