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
)

func main() {
	outDir := filepath.Join("examples", "pdf_tools", "out")
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		exitErr(err)
	}

	templateHTML := filepath.Join(outDir, "agreement.html")
	templateJSON := filepath.Join(outDir, "agreement-data.json")
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

	writeStampLogoPNG(logoPNG)
	writeQRCodePNG(qrCodePNG, "PSA-2026-0042|Oarkflow Labs Pvt. Ltd.|Acme Ltd.|2026-06-19")
	logoDataURI := pngDataURI(logoPNG)
	qrCodeDataURI := pngDataURI(qrCodePNG)

	mustWrite(templateHTML, []byte(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>{{ agreement.title }}</title>
  <style>
    @page {
      size: A4;
      margin: 8px 10px;
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      padding: 0;
      background: #ffffff;
      color: #1f2933;
      font-family: Helvetica, Arial, sans-serif;
      font-size: 10.5pt;
      line-height: 1.58;
      -webkit-print-color-adjust: exact;
      print-color-adjust: exact;
    }

    .document {
      max-width: 780px;
      margin: 0 auto;
    }

    .top-rule {
      height: 5px;
      background: #102a43;
      border-radius: 99px;
      margin-bottom: 18px;
    }

    .header {
      text-align: center;
      margin-bottom: 22px;
      padding-bottom: 16px;
      border-bottom: 1px solid #d9e2ec;
    }

    h1 {
      margin: 0;
      color: #102a43;
      font-size: 24pt;
      font-weight: 700;
      line-height: 1.18;
      letter-spacing: .02em;
      text-transform: uppercase;
    }

    .subtitle {
      display: inline-block;
      margin-top: 8px;
      padding: 5px 12px;
      border: 1px solid #d9e2ec;
      border-radius: 99px;
      color: #52606d;
      background: #f8fafc;
      font-size: 8.8pt;
      font-weight: 600;
      letter-spacing: .01em;
    }

    .meta {
      width: 100%;
      border-collapse: separate;
      border-spacing: 0;
      margin: 0 0 20px;
      overflow: hidden;
      border: 1px solid #bcccdc;
      border-radius: 8px;
    }

    .meta td {
      padding: 10px 12px;
      vertical-align: top;
      border-bottom: 1px solid #d9e2ec;
    }

    .meta tr:last-child td {
      border-bottom: 0;
    }

    .meta .label {
      width: 28%;
      color: #243b53;
      background: #f0f4f8;
      font-size: 8.6pt;
      font-weight: 700;
      letter-spacing: .04em;
      text-transform: uppercase;
      border-right: 1px solid #d9e2ec;
      white-space: nowrap;
    }

    .party-name {
      color: #102a43;
      font-size: 11pt;
      font-weight: 700;
    }

    .muted {
      color: #52606d;
    }

    .section {
      margin: 0 0 14px;
      padding: 12px 14px 13px;
      border: 1px solid #e6edf3;
      border-radius: 8px;
      page-break-inside: avoid;
    }

    h2 {
      margin: 0 0 8px;
      padding-bottom: 6px;
      border-bottom: 1px solid #d9e2ec;
      color: #102a43;
      font-size: 12.2pt;
      font-weight: 700;
      line-height: 1.25;
    }

    p {
      margin: 0 0 7px;
    }

    p:last-child {
      margin-bottom: 0;
    }

    ol {
      margin: 7px 0 0 19px;
      padding: 0;
    }

    li {
      margin: 0 0 6px;
      padding-left: 2px;
    }

    .signatures {
      display: flex;
      gap: 10px;
      margin-top: 10px;
    }

    .signature-card {
      flex: 1;
      width: 50%;
      padding: 12px;
      vertical-align: top;
      border: 1px solid #bcccdc;
      border-radius: 8px;
      background: #f8fafc;
      page-break-inside: avoid;
    }

    .signature-card strong {
      display: block;
      margin-bottom: 10px;
      color: #102a43;
      font-size: 10.5pt;
    }

    .signature-line {
      min-height: 68px;
      margin: 0 0 8px;
      padding: 5px 0 7px;
      border-bottom: 1.2px solid #243b53;
    }

    .signature-img {
      display: block;
      width: 58px;
      height: 58px;
      object-fit: contain;
    }

    .signature-detail {
      margin-top: 7px;
      font-size: 9.5pt;
    }

    .signature-detail .row {
      margin-bottom: 2px;
    }

    .field-label {
      color: #52606d;
      font-size: 8.4pt;
      font-weight: 700;
      letter-spacing: .03em;
      text-transform: uppercase;
    }

    .stamp-box {
      width: 136px;
      height: 82px;
      margin-top: 11px;
      padding: 5px;
      border: 1px dashed #9fb3c8;
      border-radius: 6px;
      background: #ffffff;
    }

    .stamp-img {
      display: block;
      width: 124px;
      height: 70px;
      object-fit: contain;
    }

    .footer-note {
      margin-top: 16px;
      padding-top: 8px;
      border-top: 1px solid #d9e2ec;
      color: #52606d;
      font-size: 8.4pt;
      text-align: center;
    }

    @media print {
      .document {
        max-width: none;
      }

      .section,
      .signature-card,
      .meta {
        break-inside: avoid;
      }
    }
  </style>
</head>
<body>
  <div class="document">
    <div class="top-rule"></div>

    <header class="header">
      <h1>{{ agreement.title }}</h1>
      <div class="subtitle">Agreement No. {{ agreement.number }} | Effective {{ agreement.effective_date }}</div>
    </header>

    <table class="meta">
      <tr>
        <td class="label">Provider</td>
        <td>
          <span class="party-name">{{ provider.name }}</span><br>
          <span class="muted">{{ provider.address }}</span><br>
          Authorized representative: {{ provider.signatory.name }}, {{ provider.signatory.title }}
        </td>
      </tr>
      <tr>
        <td class="label">Client</td>
        <td>
          <span class="party-name">{{ client.name }}</span><br>
          <span class="muted">{{ client.address }}</span><br>
          Authorized representative: {{ client.signatory.name }}, {{ client.signatory.title }}
        </td>
      </tr>
      <tr>
        <td class="label">Term and value</td>
        <td>{{ agreement.term }} | {{ agreement.value }}</td>
      </tr>
    </table>

    <section class="section">
      <h2>1. Scope of Services</h2>
      <p>{{ provider.name }} will provide the services described below to {{ client.name }}.</p>
      <ol>
        {{ range item in services }}
        <li>{{ $item }}</li>
        {{ end }}
      </ol>
    </section>

    <section class="section">
      <h2>2. Payment</h2>
      <p>{{ client.name }} will pay {{ agreement.value }} according to the following schedule: {{ payment.schedule }}.</p>
      <p>Invoices are payable within {{ payment.net_days }} days from receipt unless disputed in writing.</p>
    </section>

    <section class="section">
      <h2>3. Confidentiality</h2>
      <p>Each party will protect non-public business, financial, technical, and customer information received from the other party and will use it only to perform this agreement.</p>
    </section>

    <section class="section">
      <h2>4. Deliverables and Acceptance</h2>
      <p>Deliverables will be reviewed by {{ client.signatory.name }} or a designated reviewer. Written acceptance, production use, or no rejection within {{ agreement.acceptance_days }} days will constitute acceptance.</p>
    </section>

    <section class="section">
      <h2>5. Signatures and Stamp Images</h2>
      <p>The names, signature QR images, and company stamp images below are populated from JSON data. The same QR image is also used by the post-generation signature-image placement example.</p>

      <div class="signatures">
        <div class="signature-card">
          <strong>For {{ provider.name }}</strong>
          <div class="signature-line">
            <img class="signature-img" src="{{ provider.signatory.signature_image }}" alt="Provider signature QR">
          </div>
          <div class="signature-detail">
            <div class="row"><span class="field-label">Name:</span> {{ provider.signatory.name }}</div>
            <div class="row"><span class="field-label">Title:</span> {{ provider.signatory.title }}</div>
            <div class="row"><span class="field-label">Date:</span> {{ agreement.signature_date }}</div>
          </div>
          <div class="stamp-box">
            <img class="stamp-img" src="{{ provider.stamp_image }}" alt="Provider stamp">
          </div>
        </div>

        <div class="signature-card">
          <strong>For {{ client.name }}</strong>
          <div class="signature-line">
            <img class="signature-img" src="{{ client.signatory.signature_image }}" alt="Client signature QR">
          </div>
          <div class="signature-detail">
            <div class="row"><span class="field-label">Name:</span> {{ client.signatory.name }}</div>
            <div class="row"><span class="field-label">Title:</span> {{ client.signatory.title }}</div>
            <div class="row"><span class="field-label">Date:</span> {{ agreement.signature_date }}</div>
          </div>
          <div class="stamp-box">
            <img class="stamp-img" src="{{ client.stamp_image }}" alt="Client stamp">
          </div>
        </div>
      </div>
    </section>

    <p class="footer-note">Generated by github.com/oarkflow/pdf using HTML placeholders and JSON data.</p>
  </div>
</body>
</html>`))
	agreementData := map[string]any{
		"agreement": map[string]any{
			"title":           "Professional Services Agreement",
			"number":          "PSA-2026-0042",
			"effective_date":  "June 19, 2026",
			"term":            "12 months",
			"value":           "USD 48,000",
			"acceptance_days": 10,
			"signature_date":  "June 19, 2026",
		},
		"provider": map[string]any{
			"name":        "Oarkflow Labs Pvt. Ltd.",
			"address":     "Kathmandu, Nepal",
			"stamp_file":  "logo.png",
			"stamp_image": logoDataURI,
			"stamp_placement": map[string]any{
				"page":  1,
				"x":     82,
				"y":     58,
				"width": 124,
			},
			"signatory": map[string]any{
				"name":            "Sujit Shrestha",
				"title":           "Managing Director",
				"signature_file":  "qr-code.png",
				"signature_image": qrCodeDataURI,
				"signature_placement": map[string]any{
					"page":  1,
					"x":     82,
					"y":     144,
					"width": 54,
				},
			},
		},
		"client": map[string]any{
			"name":        "Acme Ltd.",
			"address":     "Austin, Texas, USA",
			"stamp_file":  "logo.png",
			"stamp_image": logoDataURI,
			"stamp_placement": map[string]any{
				"page":  1,
				"x":     320,
				"y":     58,
				"width": 124,
			},
			"signatory": map[string]any{
				"name":            "Ada Lovelace",
				"title":           "Chief Operating Officer",
				"signature_file":  "qr-code.png",
				"signature_image": qrCodeDataURI,
				"signature_placement": map[string]any{
					"page":  1,
					"x":     320,
					"y":     144,
					"width": 54,
				},
			},
		},
		"services": []string{
			"PDF generation, conversion, and document automation tooling.",
			"Template design with JSON-driven placeholder filling.",
			"Form, signature image, redaction, compression, and validation workflows.",
		},
		"payment": map[string]any{
			"schedule": "50% upon signing and 50% after delivery acceptance",
			"net_days": 15,
		},
	}
	agreementJSON, err := json.MarshalIndent(agreementData, "", "  ")
	if err != nil {
		exitErr(err)
	}
	mustWrite(templateJSON, agreementJSON)
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
	writePNG(scanPNG, color.RGBA{R: 230, G: 230, B: 230, A: 255})

	if err := stampAgreementAssetsFromJSON(filledTemplatePDF, stampedPDF, templateJSON, outDir); err != nil {
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

type agreementAssets struct {
	Provider agreementPartyAssets `json:"provider"`
	Client   agreementPartyAssets `json:"client"`
}

type agreementPartyAssets struct {
	StampFile      string         `json:"stamp_file"`
	StampPlacement imagePlacement `json:"stamp_placement"`
	Signatory      struct {
		SignatureFile      string         `json:"signature_file"`
		SignaturePlacement imagePlacement `json:"signature_placement"`
	} `json:"signatory"`
}

type imagePlacement struct {
	Page   int     `json:"page"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func stampAgreementAssetsFromJSON(inputPDF, outputPDF, jsonPath, baseDir string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}
	var assets agreementAssets
	if err := json.Unmarshal(data, &assets); err != nil {
		return err
	}
	stamps := []struct {
		file      string
		placement imagePlacement
	}{
		{assets.Provider.Signatory.SignatureFile, assets.Provider.Signatory.SignaturePlacement},
		{assets.Provider.StampFile, assets.Provider.StampPlacement},
		{assets.Client.Signatory.SignatureFile, assets.Client.Signatory.SignaturePlacement},
		{assets.Client.StampFile, assets.Client.StampPlacement},
	}

	current := inputPDF
	var temps []string
	defer func() {
		for _, temp := range temps {
			_ = os.Remove(temp)
		}
	}()
	for i, stamp := range stamps {
		if stamp.file == "" {
			continue
		}
		next := outputPDF
		if i < len(stamps)-1 {
			next = filepath.Join(baseDir, fmt.Sprintf(".agreement-stamp-step-%d.pdf", i+1))
			temps = append(temps, next)
		}
		placement := stamp.placement
		if placement.Page <= 0 {
			placement.Page = 1
		}
		if err := pdf.StampImage(current, next, pdf.ImageStampOptions{
			ImagePath: filepath.Join(baseDir, stamp.file),
			Page:      placement.Page,
			X:         placement.X,
			Y:         placement.Y,
			Width:     placement.Width,
			Height:    placement.Height,
		}); err != nil {
			return err
		}
		current = next
	}
	if current != outputPDF {
		return fmt.Errorf("no agreement assets were stamped")
	}
	return nil
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
