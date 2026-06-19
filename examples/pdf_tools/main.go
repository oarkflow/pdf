package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdimage "image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/barcode"
	"github.com/oarkflow/pdf/document"
	pdfhtml "github.com/oarkflow/pdf/html"
	"github.com/oarkflow/pdf/layout"
	pdftemplate "github.com/oarkflow/pdf/template"
)

func main() {
	outDir := filepath.Join("examples", "pdf_tools", "out")
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		exitErr(err)
	}

	templateHTML := filepath.Join("examples", "pdf_tools", "agreement.html")
	outTemplateHTML := filepath.Join(outDir, "agreement.html")
	agreementDataJSON := filepath.Join("examples", "pdf_tools", "agreement-data.json")
	filledTemplatePDF := filepath.Join(outDir, "agreement-filled.pdf")
	formPDF := filepath.Join(outDir, "application-form.pdf")
	formJSON := filepath.Join(outDir, "application-data.json")
	filledFormPDF := filepath.Join(outDir, "application-filled.pdf")
	sourcePDF := filepath.Join(outDir, "source.pdf")
	logoPNG := filepath.Join(outDir, "logo.png")
	qrCodePNG := filepath.Join(outDir, "qr-code.png")
	scanPNG := filepath.Join(outDir, "scan-page.png")
	stampedPDF := filepath.Join(outDir, "agreement-signed-visual.pdf")
	scannedPDF := filepath.Join(outDir, "scanned-images.pdf")
	compressedPDF := filepath.Join(outDir, "compressed.pdf")
	redactedPDF := filepath.Join(outDir, "redacted.pdf")
	translationJSON := filepath.Join(outDir, "translations.json")
	translatedPDF := filepath.Join(outDir, "translated.pdf")
	printPDF := filepath.Join(outDir, "print-ready.pdf")
	archivePDF := filepath.Join(outDir, "archive-copy.pdf")
	graphJSON := filepath.Join(outDir, "graph.json")
	agreementJSON := filepath.Join(outDir, "agreement-data.json")

	writeStampLogoPNG(logoPNG)
	writeQRCodePNG(qrCodePNG, "PSA-2026-0042|Oarkflow Labs Pvt. Ltd.|Acme Ltd.|2026-06-19")
	logoDataURI := pngDataURI(logoPNG)
	qrCodeDataURI := pngDataURI(qrCodePNG)
	templateBytes, err := os.ReadFile(templateHTML)
	if err != nil {
		exitErr(err)
	}
	mustWrite(outTemplateHTML, templateBytes)

	agreementData, err := pdf.LoadTemplateJSON(agreementDataJSON)
	if err != nil {
		exitErr(err)
	}
	injectAgreementImageData(agreementData, logoDataURI, qrCodeDataURI)
	agreementJSONBytes, err := json.MarshalIndent(agreementData, "", "  ")
	if err != nil {
		exitErr(err)
	}
	mustWrite(agreementJSON, agreementJSONBytes)
	if err := renderAgreementPDF(outTemplateHTML, agreementData, filledTemplatePDF); err != nil {
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
	writePNG(scanPNG, color.RGBA{R: 230, G: 230, B: 230, A: 255})

	// REPLACE with a simple copy so stampedPDF still exists for downstream uses:
	srcData, err := os.ReadFile(filledTemplatePDF)
	if err != nil {
		exitErr(err)
	}
	if err := os.WriteFile(stampedPDF, srcData, 0644); err != nil {
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

func injectAgreementImageData(agreementData map[string]any, logoDataURI, qrCodeDataURI string) {
	provider, _ := agreementData["provider"].(map[string]any)
	client, _ := agreementData["client"].(map[string]any)
	providerSignatory, _ := provider["signatory"].(map[string]any)
	clientSignatory, _ := client["signatory"].(map[string]any)

	if provider != nil {
		provider["stamp_image"] = logoDataURI
	}
	if providerSignatory != nil {
		providerSignatory["signature_image"] = qrCodeDataURI
	}
	if client != nil {
		client["stamp_image"] = logoDataURI
	}
	if clientSignatory != nil {
		clientSignatory["signature_image"] = qrCodeDataURI
	}
}

func renderAgreementPDF(templatePath string, agreementData map[string]any, outputPath string) error {
	renderedHTML, err := pdftemplate.RenderHTMLFile(templatePath, agreementData)
	if err != nil {
		return fmt.Errorf("rendering agreement template: %w", err)
	}

	result, err := pdfhtml.Convert(renderedHTML, pdfhtml.Options{
		DefaultFontSize:   10.5,
		DefaultFontFamily: "Helvetica",
		MediaType:         "print",
	})
	if err != nil {
		return fmt.Errorf("converting agreement HTML: %w", err)
	}

	doc, err := document.NewDocument(document.PageSize{Width: result.Config.Width, Height: result.Config.Height})
	if err != nil {
		return err
	}
	doc.SetMargins(document.Margins{
		Top:    result.Config.Margins[0],
		Right:  result.Config.Margins[1],
		Bottom: result.Config.Margins[2],
		Left:   result.Config.Margins[3],
	})

	hasHF := len(result.HeaderElements) > 0 || len(result.FooterElements) > 0
	var pages []layout.PageResult
	if hasHF {
		pages = layout.RenderPagesWithHeaderFooter(
			result.Elements, result.HeaderElements, result.FooterElements,
			result.Config.Width, result.Config.Height,
			result.Config.Margins[0], result.Config.Margins[1],
			result.Config.Margins[2], result.Config.Margins[3],
		)
	} else {
		pages = layout.RenderPages(
			result.Elements,
			result.Config.Width, result.Config.Height,
			result.Config.Margins[0], result.Config.Margins[1],
			result.Config.Margins[2], result.Config.Margins[3],
		)
	}

	for _, pr := range pages {
		page := document.NewPage(document.PageSize{Width: pr.Width, Height: pr.Height})
		page.Contents = pr.Content
		for _, fe := range pr.Fonts {
			page.FontEntries[fe.PDFName] = fe
		}
		for name, ie := range pr.Images {
			page.Images[name] = ie
		}
		page.Annotations = pr.Links
		doc.AddPage(page)
	}

	return doc.Save(outputPath)
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

func writeStampLogoPNG(path string) {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, 180, 96))
	for y := 0; y < 96; y++ {
		for x := 0; x < 180; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	blue := color.RGBA{R: 18, G: 86, B: 145, A: 255}
	light := color.RGBA{R: 225, G: 239, B: 252, A: 255}
	for y := 8; y < 88; y++ {
		for x := 18; x < 162; x++ {
			dx := float64(x-90) / 72
			dy := float64(y-48) / 40
			r := dx*dx + dy*dy
			if r < 1.0 {
				img.Set(x, y, light)
			}
			if r > 0.82 && r < 1.0 {
				img.Set(x, y, blue)
			}
			if r > 0.36 && r < 0.42 {
				img.Set(x, y, blue)
			}
		}
	}
	for y := 42; y < 54; y++ {
		for x := 46; x < 134; x++ {
			img.Set(x, y, blue)
		}
	}
	writePNGFile(path, img)
}

func writeQRCodePNG(path, payload string) {
	qr, err := barcode.EncodeQR(payload, barcode.ECMedium)
	if err != nil {
		exitErr(err)
	}
	scale := 5
	quiet := 4
	size := (qr.Size + quiet*2) * scale
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	black := color.RGBA{A: 255}
	for row := 0; row < qr.Size; row++ {
		for col := 0; col < qr.Size; col++ {
			if !qr.Modules[row][col] {
				continue
			}
			for y := 0; y < scale; y++ {
				for x := 0; x < scale; x++ {
					img.Set((col+quiet)*scale+x, (row+quiet)*scale+y, black)
				}
			}
		}
	}
	writePNGFile(path, img)
}

func writePNGFile(path string, img stdimage.Image) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		exitErr(err)
	}
	mustWrite(path, buf.Bytes())
}

func pngDataURI(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		exitErr(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
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
