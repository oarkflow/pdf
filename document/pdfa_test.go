package document

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/layout"
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

func TestPDFA4_XMPContainsPart4Revision(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetMetadata(Metadata{Title: "PDF/A-4 Test"})
	doc.SetPDFA(PDFA4)
	doc.NewPage()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "<pdfaid:part>4</pdfaid:part>") {
		t.Error("expected pdfaid:part=4 in XMP metadata")
	}
	if !strings.Contains(out, "<pdfaid:rev>2020</pdfaid:rev>") {
		t.Error("expected pdfaid:rev=2020 in XMP metadata")
	}
}

func TestPDFUA2_XMPAndCatalogMarkers(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetMetadata(Metadata{Title: "PDF/UA-2 Test"})
	doc.SetPDFUA(PDFUA2)
	doc.NewPage()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	for _, want := range []string{
		"<pdfuaid:part>2</pdfuaid:part>",
		"/MarkInfo",
		"/StructTreeRoot",
		"/Lang (en-US)",
		"/ViewerPreferences",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q", want)
		}
	}
}

func TestPDFAWithPDFUAIncludesExtensionSchema(t *testing.T) {
	pdfa := PDFA2b
	pdfua := PDFUA1
	xmp := string(BuildComplianceXMPMetadata(Metadata{Title: "Extension Test"}, &pdfa, &pdfua))
	for _, want := range []string{
		"pdfaExtension:schemas",
		"PDF/UA identification schema",
		"pdfaSchema:namespaceURI>http://www.aiim.org/pdfua/ns/id/",
		"pdfaProperty:name>part",
		`xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/"`,
		`xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/"`,
	} {
		if !strings.Contains(xmp, want) {
			t.Errorf("expected XMP extension schema to contain %q", want)
		}
	}
}

func TestPDFAWithPDFUAXMPWellFormedXML(t *testing.T) {
	pdfa := PDFA2b
	pdfua := PDFUA1
	xmp := string(BuildComplianceXMPMetadata(Metadata{Title: "PDF/A + PDF/UA"}, &pdfa, &pdfua))

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

func TestLinkAnnotationsHavePrintFlag(t *testing.T) {
	doc, _ := NewDocument(A4)
	page := NewPage(A4)
	page.Contents = []byte("BT ET")
	page.Annotations = append(page.Annotations, layout.LinkAnnotation{X1: 10, Y1: 10, X2: 20, Y2: 20, URI: "https://example.com"})
	doc.AddPage(page)

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "/F 4") {
		t.Fatal("expected link annotation to include /F 4")
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
