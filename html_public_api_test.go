package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestPublicLeanAndCompliantHTMLToPDF(t *testing.T) {
	html := `<!doctype html><html><head><title>API Test</title></head><body><h1>Hello</h1><p>World</p></body></html>`

	var lean bytes.Buffer
	if err := WriteLeanHTMLToPDF(&lean, html); err != nil {
		t.Fatalf("WriteLeanHTMLToPDF: %v", err)
	}
	if !strings.HasPrefix(lean.String(), "%PDF-") {
		t.Fatal("lean output is not a PDF")
	}
	if strings.Contains(lean.String(), "pdfuaid:part") {
		t.Fatal("lean output unexpectedly contains PDF/UA metadata")
	}

	var compliant bytes.Buffer
	if err := WriteCompliantHTMLToPDF(&compliant, html); err != nil {
		t.Fatalf("WriteCompliantHTMLToPDF: %v", err)
	}
	out := compliant.String()
	for _, want := range []string{
		"<pdfaid:part>2</pdfaid:part>",
		"<pdfaid:conformance>B</pdfaid:conformance>",
		"<pdfuaid:part>1</pdfuaid:part>",
		"/StructTreeRoot",
		"/MarkInfo",
		"/OutputIntents",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("compliant output missing %q", want)
		}
	}
}

func TestCompileCompliantHTMLPDFUA2PDFA4WithOptions(t *testing.T) {
	html := `<!doctype html><html><body><p>PDF/A-4 + PDF/UA-2</p></body></html>`
	compiled, err := CompileCompliantHTMLWithOptions(html, HTMLComplianceOptions{
		PDFA:     PDFA4,
		PDFUA:    PDFUA2,
		Language: "en-US",
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := compiled.WriteStreamingTo(&out); err != nil {
		t.Fatal(err)
	}
	pdf := out.String()
	for _, want := range []string{
		"<pdfaid:part>4</pdfaid:part>",
		"<pdfaid:rev>2020</pdfaid:rev>",
		"<pdfuaid:part>2</pdfuaid:part>",
	} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("PDF/A-4 + PDF/UA-2 output missing %q", want)
		}
	}
}

func TestCompileCompliantHTMLWithOptions(t *testing.T) {
	html := `<!doctype html><html><body><p>Custom compliance</p></body></html>`
	compiled, err := CompileCompliantHTMLWithOptions(html, HTMLComplianceOptions{
		PDFA:     PDFA2b,
		PDFUA:    PDFUA1,
		Language: "en-GB",
	})
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := compiled.WriteStreamingTo(&out); err != nil {
		t.Fatal(err)
	}
	pdf := out.String()
	for _, want := range []string{
		"<pdfaid:part>2</pdfaid:part>",
		"<pdfuaid:part>1</pdfuaid:part>",
		"/Lang (en-GB)",
	} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("custom compliant output missing %q", want)
		}
	}
}
