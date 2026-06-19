package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	stdimage "image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/document"
)

func main() {
	outDir := filepath.Join("examples", "pdf_tools", "out")
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		exitErr(err)
	}

	templateHTML := filepath.Join(outDir, "offer.html")
	templateJSON := filepath.Join(outDir, "offer.json")
	filledTemplatePDF := filepath.Join(outDir, "offer-filled.pdf")
	formPDF := filepath.Join(outDir, "application-form.pdf")
	formJSON := filepath.Join(outDir, "application-data.json")
	filledFormPDF := filepath.Join(outDir, "application-filled.pdf")
	sourcePDF := filepath.Join(outDir, "source.pdf")
	signaturePNG := filepath.Join(outDir, "signature.png")
	scanPNG := filepath.Join(outDir, "scan-page.png")
	stampedPDF := filepath.Join(outDir, "signature-image.pdf")
	scannedPDF := filepath.Join(outDir, "scanned-images.pdf")
	compressedPDF := filepath.Join(outDir, "compressed.pdf")
	redactedPDF := filepath.Join(outDir, "redacted.pdf")
	translationJSON := filepath.Join(outDir, "translations.json")
	translatedPDF := filepath.Join(outDir, "translated.pdf")
	printPDF := filepath.Join(outDir, "print-ready.pdf")
	archivePDF := filepath.Join(outDir, "archive-copy.pdf")
	graphJSON := filepath.Join(outDir, "graph.json")

	mustWrite(templateHTML, []byte(`<!doctype html>
<html>
<body style="font-family: Helvetica; margin: 36px">
  <h1>{{ title }}</h1>
  <p>Customer: {{ customer.name }}</p>
  <p>Offer: {{ offer }}</p>
</body>
</html>`))
	mustWrite(templateJSON, []byte(`{
  "title": "Template JSON Example",
  "customer": { "name": "Acme Ltd." },
  "offer": "Annual support package"
}`))
	if err := pdf.FromHTMLTemplateJSONFile(templateHTML, templateJSON, filledTemplatePDF); err != nil {
		exitErr(err)
	}

	writeMinimalFormPDF(formPDF)
	mustWrite(formJSON, []byte(`{"full_name":"Ada Lovelace","approved":true}`))
	fields, err := pdf.ListFormFields(formPDF, "")
	if err != nil {
		exitErr(err)
	}
	if err := pdf.FillFormJSONFile(formPDF, formJSON, filledFormPDF); err != nil {
		exitErr(err)
	}

	writeTextPDF(sourcePDF, "Secret contract for Hello Corp")
	writePNG(signaturePNG, color.RGBA{R: 25, G: 80, B: 180, A: 255})
	writePNG(scanPNG, color.RGBA{R: 230, G: 230, B: 230, A: 255})

	if err := pdf.SignatureImage(sourcePDF, stampedPDF, pdf.ImageStampOptions{
		ImagePath: signaturePNG,
		Page:      1,
		X:         72,
		Y:         96,
		Width:     120,
	}); err != nil {
		exitErr(err)
	}
	if err := pdf.ScanImagesToPDF(pdf.ImagePDFOptions{
		Output: scannedPDF,
		Inputs: []pdf.MergeInput{{Path: scanPNG, Type: "image"}},
	}); err != nil {
		exitErr(err)
	}
	if err := pdf.CompressPDF(sourcePDF, compressedPDF, ""); err != nil {
		exitErr(err)
	}
	if err := pdf.Redact(sourcePDF, redactedPDF, pdf.RedactOptions{
		Texts: []string{"Secret"},
		Regions: []pdf.RedactionRegion{{
			Page: 1, X: 72, Y: 700, Width: 100, Height: 16,
		}},
	}); err != nil {
		exitErr(err)
	}
	mustWrite(translationJSON, []byte(`{"Hello":"Namaste","contract":"agreement"}`))
	if err := pdf.TranslateWithDictionary(sourcePDF, translationJSON, translatedPDF, ""); err != nil {
		exitErr(err)
	}

	comparison, err := pdf.ComparePDF(sourcePDF, compressedPDF)
	if err != nil {
		exitErr(err)
	}
	if err := pdf.PrepareForPrint(sourcePDF, printPDF, map[string]string{
		"Title":  "Print Prepared Example",
		"Author": "github.com/oarkflow/pdf",
	}, true, ""); err != nil {
		exitErr(err)
	}
	archiveReport, err := pdf.ArchivePDF(sourcePDF, archivePDF, "")
	if err != nil {
		exitErr(err)
	}
	graph, err := pdf.BuildPDFGraph([]string{sourcePDF, stampedPDF, filledTemplatePDF}, "")
	if err != nil {
		exitErr(err)
	}
	graphBytes, _ := json.MarshalIndent(graph, "", "  ")
	mustWrite(graphJSON, graphBytes)

	fmt.Printf("Wrote examples to %s\n", outDir)
	fmt.Printf("Form fields discovered: %d\n", len(fields))
	fmt.Printf("Compressed comparison differences: %d\n", len(comparison.Differences))
	fmt.Printf("Archive validation valid: %t\n", archiveReport.Valid)
	fmt.Printf("Graph nodes: %d\n", len(graph.Nodes))
}

func writeTextPDF(path, text string) {
	doc, err := document.NewDocument(document.A4)
	if err != nil {
		exitErr(err)
	}
	page := doc.NewPage()
	page.Contents = []byte("BT /F1 12 Tf 72 760 Td (" + escapePDFLiteral(text) + ") Tj ET")
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		exitErr(err)
	}
	mustWrite(path, buf.Bytes())
}

func writeMinimalFormPDF(path string) {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R /AcroForm << /Fields [5 0 R 6 0 R] >> >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> >> >> /Contents 4 0 R /Annots [5 0 R 6 0 R] >>",
		"<< /Length 0 >>\nstream\n\nendstream",
		"<< /Type /Annot /Subtype /Widget /FT /Tx /T (full_name) /Rect [72 700 260 725] /P 3 0 R >>",
		"<< /Type /Annot /Subtype /Widget /FT /Btn /T (approved) /Rect [72 660 92 680] /P 3 0 R >>",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	mustWrite(path, buf.Bytes())
}

func writePNG(path string, c color.RGBA) {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, 160, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 160; x++ {
			if y > 20 && y < 35 && x > 12 && x < 148 {
				img.Set(x, y, c)
			} else {
				img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		exitErr(err)
	}
	mustWrite(path, buf.Bytes())
}

func mustWrite(path string, data []byte) {
	if err := os.WriteFile(path, data, 0644); err != nil {
		exitErr(err)
	}
}

func escapePDFLiteral(s string) string {
	var out bytes.Buffer
	for _, r := range s {
		switch r {
		case '\\', '(', ')':
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	return out.String()
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
