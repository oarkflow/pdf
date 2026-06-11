// Package main demonstrates generating a beautiful, professional invoice PDF
// from HTML+CSS using the github.com/oarkflow/pdf library with {{placeholder}} templates.
//
// Run:
//
//	go run ./examples/html_invoice/
//
// Output: invoice.pdf in the current directory.
package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/core"
	pdffont "github.com/oarkflow/pdf/font"
	"github.com/oarkflow/pdf/html"
	"github.com/oarkflow/pdf/reader"
	"github.com/oarkflow/pdf/template"
)

func main() {
	d := InvoiceData{
		Number:   "INV-2026-0042",
		Date:     time.Now().Format("January 2, 2006"),
		DueDate:  time.Now().AddDate(0, 0, 30).Format("January 2, 2006"),
		Currency: "USD",
		From: Company{
			Name:    "Acme Corporation",
			Address: "123 Innovation Drive",
			City:    "San Francisco",
			State:   "CA",
			Zip:     "94105",
			Country: "United States",
			Phone:   "+1 (415) 555-0198",
			Email:   "billing@acmecorp.com",
			Website: "www.acmecorp.com",
			TaxID:   "US-87-1234567",
		},
		To: Company{
			Name:    "Wayne Enterprises",
			Address: "1007 Mountain Drive",
			City:    "Gotham City",
			State:   "NJ",
			Zip:     "07001",
			Country: "United States",
			Phone:   "+1 (201) 555-0142",
			Email:   "accounts@wayne.com",
		},
		Items: []LineItem{
			{Description: "Enterprise Platform License", Details: "Annual subscription — 500 seats", Qty: 1, UnitPrice: 24999.00},
			{Description: "Custom Integration Module", Details: "REST API + SSO connector", Qty: 2, UnitPrice: 4500.00},
			{Description: "Professional Services", Details: "On-site training — 3 day workshop", Qty: 3, UnitPrice: 1800.00},
			{Description: "24/7 Premium Support", Details: "12-month contract", Qty: 1, UnitPrice: 7200.00},
			{Description: "Data Migration Service", Details: "Legacy system import — up to 5M records", Qty: 1, UnitPrice: 3500.00},
		},
		TaxRate:     8.5,
		Notes:       "Thank you for your business! Payment is due within 30 days. Late payments may incur a 1.5% monthly finance charge.",
		PaymentInfo: "Wire Transfer: First National Bank, Routing #021000021, Account #1234567890\nOr pay online at: https://pay.acmecorp.com/inv/2026-0042",
	}

	// Calculate totals
	subtotal := 0.0
	for _, item := range d.Items {
		subtotal += float64(item.Qty) * item.UnitPrice
	}
	tax := subtotal * d.TaxRate / 100
	total := subtotal + tax

	// Build item rows HTML
	itemRows := ""
	for i, item := range d.Items {
		amount := float64(item.Qty) * item.UnitPrice
		bgColor := "#ffffff"
		if i%2 == 1 {
			bgColor = "#f8f9fb"
		}
		itemRows += fmt.Sprintf(`
        <tr style="background-color: %s;">
          <td style="padding: 10px 12px; border-bottom: 1px solid #eaedf1;">
            <div style="font-weight: 600; color: #1a1f36;">%s</div>
            <div style="font-size: 8.5pt; color: #6b7294; margin-top: 2px;">%s</div>
          </td>
          <td style="padding: 10px 12px; border-bottom: 1px solid #eaedf1; text-align: center; color: #3c4257;">%d</td>
          <td style="padding: 10px 12px; border-bottom: 1px solid #eaedf1; text-align: right; color: #3c4257;">%s</td>
          <td style="padding: 10px 12px; border-bottom: 1px solid #eaedf1; text-align: right; font-weight: 600; color: #1a1f36;">%s</td>
        </tr>`, bgColor, item.Description, item.Details, item.Qty, fmtMoney(item.UnitPrice), fmtMoney(amount))
	}

	// Build logo HTML
	logoHTML := ""
	logoDataURI := loadLogoDataURI()
	if logoDataURI != "" {
		logoHTML = fmt.Sprintf(`<img class="brand-logo" src="%s" alt="%s logo">`, logoDataURI, d.From.Name)
	}

	// Build tax ID line
	taxIDLine := ""
	if d.From.TaxID != "" {
		taxIDLine = "<br>Tax ID: " + d.From.TaxID
	}

	// Build the data map for {{placeholder}} resolution
	data := map[string]any{
		"invoice_number":    d.Number,
		"from_name":         d.From.Name,
		"from_email":        d.From.Email,
		"from_email_href":   "mailto:" + d.From.Email,
		"from_phone":        d.From.Phone,
		"from_phone_href":   phoneHref(d.From.Phone),
		"from_website":      d.From.Website,
		"from_website_href": websiteHref(d.From.Website),
		"from_address":      d.From.Address,
		"from_city":         d.From.City,
		"from_state":        d.From.State,
		"from_zip":          d.From.Zip,
		"from_country":      d.From.Country,
		"from_tax_id":       d.From.TaxID,
		"from_tax_id_line":  taxIDLine,
		"from_full_address": d.From.Address + ", " + d.From.City + ", " + d.From.State + " " + d.From.Zip,
		"to_name":           d.To.Name,
		"to_address":        d.To.Address,
		"to_city":           d.To.City,
		"to_state":          d.To.State,
		"to_zip":            d.To.Zip,
		"to_country":        d.To.Country,
		"to_phone":          d.To.Phone,
		"to_phone_href":     phoneHref(d.To.Phone),
		"to_email":          d.To.Email,
		"to_email_href":     "mailto:" + d.To.Email,
		"date":              d.Date,
		"due_date":          d.DueDate,
		"currency":          d.Currency,
		"logo_html":         logoHTML,
		"item_rows":         itemRows,
		"subtotal":          fmtMoney(subtotal),
		"tax_rate":          fmt.Sprintf("%.1f", d.TaxRate),
		"tax":               fmtMoney(tax),
		"total":             fmtMoney(total),
		"notes":             d.Notes,
		"payment_info":      d.PaymentInfo,
	}

	// Load the HTML template from file
	templateBytes, err := os.ReadFile(filepath.Join("examples", "html_invoice", "template.html"))
	if err != nil {
		templateBytes, err = os.ReadFile("template.html")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading template.html: %v\n", err)
		os.Exit(1)
	}

	// Resolve {{placeholders}} in the HTML template using fasttpl
	invoiceHTML, err := template.RenderHTML(string(templateBytes), data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering template: %v\n", err)
		os.Exit(1)
	}

	baseOpt := html.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Helvetica",
		PageSize:          [2]float64{595.28, 841.89}, // A4
		Margins:           [4]float64{40, 40, 50, 40},
		UseTailwind:       true,
	}
	leanDir := filepath.Join("out", "lean")
	compliantDir := filepath.Join("out", "compliant")
	pdfa4Dir := filepath.Join("out", "pdfa4")
	protectedDir := filepath.Join("out", "protected")
	otherDir := filepath.Join("out", "other")
	ensureOutputDirs(leanDir, compliantDir, pdfa4Dir, protectedDir, otherDir)
	fmt.Println("Default compliant outputs are PDF/A-2B + PDF/UA-1 in out/compliant; PDF/A-4 outputs are in out/pdfa4 and need a PDF/A-4 validation profile.")

	leanPath := filepath.Join(leanDir, "invoice_lean.pdf")
	err = pdf.FromLeanHTML(invoiceHTML, leanPath, baseOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s\n", leanPath)
	validateGeneratedPDF(leanPath, "", "INVOICE", d.Number, d.From.Name, d.To.Name)

	leanWriterPath := filepath.Join(leanDir, "invoice_lean_writer.pdf")
	leanFile, err := os.Create(leanWriterPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating lean writer invoice: %v\n", err)
		os.Exit(1)
	}
	err = pdf.WriteLeanHTMLToPDF(leanFile, invoiceHTML, baseOpt)
	closeErr := leanFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating lean writer invoice: %v\n", err)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "Error closing lean writer invoice: %v\n", closeErr)
		os.Exit(1)
	}
	fmt.Printf("Generated %s\n", leanWriterPath)
	validateGeneratedPDF(leanWriterPath, "", "INVOICE", d.Number, d.From.Name, d.To.Name)

	compliantPath := filepath.Join(compliantDir, "invoice_compliant_pdfa2b_ua1.pdf")
	err = pdf.FromCompliantHTML(invoiceHTML, compliantPath, baseOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating compliant invoice: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s (PDF/A-2B + PDF/UA-1)\n", compliantPath)
	validateGeneratedPDF(compliantPath, "", "INVOICE", d.Number, d.From.Name, d.To.Name)

	compliantWriterPath := filepath.Join(compliantDir, "invoice_compliant_writer_pdfa2b_ua1.pdf")
	compliantFile, err := os.Create(compliantWriterPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating compliant writer invoice: %v\n", err)
		os.Exit(1)
	}
	err = pdf.WriteCompliantHTMLToPDF(compliantFile, invoiceHTML, baseOpt)
	closeErr = compliantFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating compliant writer invoice: %v\n", err)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "Error closing compliant writer invoice: %v\n", closeErr)
		os.Exit(1)
	}
	fmt.Printf("Generated %s (PDF/A-2B + PDF/UA-1)\n", compliantWriterPath)
	validateGeneratedPDF(compliantWriterPath, "", "INVOICE", d.Number, d.From.Name, d.To.Name)

	pdfa4ua2 := pdf.HTMLComplianceOptions{
		PDFA:     pdf.PDFA4,
		PDFUA:    pdf.PDFUA2,
		Language: "en-US",
	}
	pdfa4ua2Path := filepath.Join(pdfa4Dir, "invoice_compliant_pdfa4_ua2.pdf")
	err = pdf.FromCompliantHTMLWithOptions(invoiceHTML, pdfa4ua2Path, pdfa4ua2, baseOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating PDF/A-4 + PDF/UA-2 invoice: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s (PDF/A-4 + PDF/UA-2)\n", pdfa4ua2Path)
	validateGeneratedPDF(pdfa4ua2Path, "", "INVOICE", d.Number, d.From.Name, d.To.Name)

	customCompliance := pdf.HTMLComplianceOptions{
		PDFA:     pdf.PDFA2b,
		PDFUA:    pdf.PDFUA1,
		Language: "en-US",
	}
	customCompliantPath := filepath.Join(compliantDir, "invoice_compliant_custom_pdfa2b_ua1.pdf")
	err = pdf.FromCompliantHTMLWithOptions(invoiceHTML, customCompliantPath, customCompliance, baseOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating custom compliant invoice: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s (PDF/A-2B + PDF/UA-1)\n", customCompliantPath)
	validateGeneratedPDF(customCompliantPath, "", "INVOICE", d.Number, d.From.Name, d.To.Name)

	compiledLean, err := pdf.CompileLeanHTML(invoiceHTML, baseOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compiling lean invoice: %v\n", err)
		os.Exit(1)
	}
	writeCompiledPDF(filepath.Join(leanDir, "invoice_compiled_lean.pdf"), compiledLean, "", "INVOICE", d.Number, d.From.Name, d.To.Name)
	writeFrozenPDF(filepath.Join(leanDir, "invoice_frozen_lean.pdf"), compiledLean, "", "INVOICE", d.Number, d.From.Name, d.To.Name)

	compiledCompliant, err := pdf.CompileCompliantHTML(invoiceHTML, baseOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error compiling compliant invoice: %v\n", err)
		os.Exit(1)
	}
	writeCompiledPDF(filepath.Join(compliantDir, "invoice_compiled_compliant_pdfa2b_ua1.pdf"), compiledCompliant, "", "INVOICE", d.Number, d.From.Name, d.To.Name)
	writeFrozenPDF(filepath.Join(compliantDir, "invoice_frozen_compliant_pdfa2b_ua1.pdf"), compiledCompliant, "", "INVOICE", d.Number, d.From.Name, d.To.Name)

	var compiledBuffer bytes.Buffer
	if err := compiledCompliant.WriteStreamingTo(&compiledBuffer); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing compliant invoice to buffer: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated compliant invoice in-memory (%d bytes)\n", compiledBuffer.Len())

	protectedOpt := baseOpt
	protectedOpt.Encryption = &core.EncryptionConfig{
		Algorithm:     core.AES_128,
		OwnerPassword: "owner-invoice-demo-2026",
		UserPassword:  "open-sesame",
		Permissions:   0xFFFFF0C4,
	}
	protectedPath := filepath.Join(protectedDir, "invoice_protected.pdf")
	err = pdf.FromLeanHTML(invoiceHTML, protectedPath, protectedOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating protected invoice: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s (password: open-sesame)\n", protectedPath)
	validateGeneratedPDF(protectedPath, "open-sesame", "INVOICE", d.Number, d.From.Name, d.To.Name)

	// Also generate PDF from the Tailwind-styled invoice.html file
	htmlBytes, err := os.ReadFile("examples/html_invoice/invoice.html")
	if err != nil {
		// Try from current directory
		htmlBytes, err = os.ReadFile("invoice.html")
	}
	if err == nil {
		tailwindOpt := html.Options{
			DefaultFontSize:   10,
			DefaultFontFamily: "Helvetica",
			PageSize:          [2]float64{595.28, 841.89},
			Margins:           [4]float64{40, 40, 50, 40},
			UseTailwind:       true,
			EnableJavaScript:  true,
		}
		tailwindPath := filepath.Join(leanDir, "invoice_tailwind.pdf")
		err = pdf.FromLeanHTML(string(htmlBytes), tailwindPath, tailwindOpt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating tailwind invoice: %v\n", err)
		} else {
			fmt.Printf("Generated %s\n", tailwindPath)
			validateGeneratedPDF(tailwindPath, "", "Monthly Newsletter", "Latest Updates")
		}

		protectedTailwindOpt := tailwindOpt
		protectedTailwindOpt.Encryption = &core.EncryptionConfig{
			Algorithm:     core.AES_128,
			OwnerPassword: "owner-newsletter-demo-2026",
			UserPassword:  "open-sesame",
			Permissions:   0xFFFFF0C4,
		}
		tailwindProtectedPath := filepath.Join(protectedDir, "invoice_tailwind_protected.pdf")
		err = pdf.FromLeanHTML(string(htmlBytes), tailwindProtectedPath, protectedTailwindOpt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating protected tailwind invoice: %v\n", err)
		} else {
			fmt.Printf("Generated %s (password: open-sesame)\n", tailwindProtectedPath)
			validateGeneratedPDF(tailwindProtectedPath, "open-sesame", "Monthly Newsletter", "Latest Updates")
		}
	}

	// Generate passport sifarish patra (recommendation letter) PDF
	generatePassportSifarish()
}

func writeCompiledPDF(path string, compiled *pdf.CompiledHTML, password string, requiredText ...string) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", path, err)
		os.Exit(1)
	}
	err = compiled.WriteStreamingTo(f)
	closeErr := f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating %s: %v\n", path, err)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "Error closing %s: %v\n", path, closeErr)
		os.Exit(1)
	}
	fmt.Printf("Generated %s\n", path)
	validateGeneratedPDF(path, password, requiredText...)
}

func writeFrozenPDF(path string, compiled *pdf.CompiledHTML, password string, requiredText ...string) {
	frozen, err := compiled.Freeze()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error freezing %s: %v\n", path, err)
		os.Exit(1)
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", path, err)
		os.Exit(1)
	}
	_, err = frozen.WriteTo(f)
	closeErr := f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating %s: %v\n", path, err)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "Error closing %s: %v\n", path, closeErr)
		os.Exit(1)
	}
	fmt.Printf("Generated %s\n", path)
	validateGeneratedPDF(path, password, requiredText...)
}

func ensureOutputDirs(dirs ...string) {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}
}

func validateGeneratedPDF(path, password string, requiredText ...string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read %s for validation: %v\n", path, err)
		return
	}
	r, err := reader.OpenWithPassword(data, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open %s for validation: %v\n", path, err)
		return
	}
	if r.NumPages() == 0 {
		fmt.Fprintf(os.Stderr, "Warning: %s has no pages\n", path)
		return
	}
	text, err := r.ExtractText(0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not extract text from %s: %v\n", path, err)
		return
	}
	for _, want := range requiredText {
		if !strings.Contains(text, want) {
			fmt.Fprintf(os.Stderr, "Warning: %s missing expected text %q\n", path, want)
			return
		}
	}
	fmt.Printf("Validated %s (%d page(s))\n", path, r.NumPages())
}

func generatePassportSifarish() {
	// Example data for passport sifarish patra
	passportData := map[string]any{
		// Layout controls. Defaults use the full PDF page with no outer
		// margin/padding. Set PASSPORT_PAGE_PADDING=18mm 20mm 18mm 20mm,
		// PASSPORT_PAGE_BORDER_TOP=6px solid #1a3a6b, etc. for a padded
		// government-letter layout.
		"page_margin":                 envString("PASSPORT_PAGE_MARGIN", "0"),
		"page_padding":                envString("PASSPORT_PAGE_PADDING", "0"),
		"print_page_padding":          envString("PASSPORT_PRINT_PAGE_PADDING", envString("PASSPORT_PAGE_PADDING", "0")),
		"page_width":                  envString("PASSPORT_PAGE_WIDTH", "100vw"),
		"page_height":                 envString("PASSPORT_PAGE_HEIGHT", "100vh"),
		"page_border_top":             envString("PASSPORT_PAGE_BORDER_TOP", "0"),
		"letterhead_margin_bottom":    envString("PASSPORT_LETTERHEAD_MARGIN_BOTTOM", "0"),
		"meta_margin":                 envString("PASSPORT_META_MARGIN", "0"),
		"subject_margin":              envString("PASSPORT_SUBJECT_MARGIN", "0"),
		"block_margin_bottom":         envString("PASSPORT_BLOCK_MARGIN_BOTTOM", "0"),
		"details_heading_margin":      envString("PASSPORT_DETAILS_HEADING_MARGIN", "0"),
		"details_table_margin_bottom": envString("PASSPORT_DETAILS_TABLE_MARGIN_BOTTOM", "0"),
		"details_cell_padding":        envString("PASSPORT_DETAILS_CELL_PADDING", "0"),
		"signature_margin_top":        envString("PASSPORT_SIGNATURE_MARGIN_TOP", "0"),
		"signature_gap":               envString("PASSPORT_SIGNATURE_GAP", "0"),
		"seal_padding":                envString("PASSPORT_SEAL_PADDING", "0"),
		"footer_margin_top":           envString("PASSPORT_FOOTER_MARGIN_TOP", "0"),
		"footer_padding_top":          envString("PASSPORT_FOOTER_PADDING_TOP", "0"),

		// Letterhead / Office details
		"municipality_name": "बुटवल उपमहानगरपालिका",
		"ward_number":       "१२",
		"district":          "रूपन्देही",
		"province":          "लुम्बिनी प्रदेश",
		"office_phone":      "०७१-५४०१२३",
		"office_email":      "ward12@butwalmunicipality.gov.np",

		// Letter metadata
		"letter_serial_no": "०८७",
		"fiscal_year":      "२०८२/८३",
		"date_bs":          "२०८२/१२/१७",
		"date_ad":          "2026-03-31",

		// Applicant personal details (Nepali)
		"applicant_full_name_nepali":  "राम बहादुर खत्री",
		"applicant_full_name_english": "Ram Bahadur Khatri",
		"dob_bs":                      "२०४५/०५/२०",
		"dob_ad":                      "1988-09-05",
		"birth_place":                 "बुटवल, रूपन्देही",
		"gender":                      "पुरुष",

		// Citizenship details
		"citizenship_no":              "१२३-४५६-७८९",
		"citizenship_issued_district": "रूपन्देही",
		"citizenship_issued_date":     "२०६३/०१/१५",

		// Family details
		"father_name":      "कृष्ण बहादुर खत्री",
		"mother_name":      "सीता देवी खत्री",
		"grandfather_name": "धन बहादुर खत्री",
		"spouse_name":      "गीता खत्री",

		// Permanent address
		"permanent_ward_no":      "१२",
		"permanent_municipality": "बुटवल उपमहानगरपालिका",
		"permanent_district":     "रूपन्देही",

		// Current address
		"current_ward_no":      "१२",
		"current_municipality": "बुटवल उपमहानगरपालिका",
		"current_district":     "रूपन्देही",

		// Contact & passport type
		"applicant_phone": "९८४७०१२३४५",
		"passport_type":   "साधारण (Ordinary)",

		// Signature section
		"authorized_officer_name": "सुरेश कुमार श्रेष्ठ",
	}

	// Load the HTML template
	templateBytes, err := os.ReadFile(filepath.Join("examples", "html_invoice", "passport_sifarish_patra.html"))
	if err != nil {
		templateBytes, err = os.ReadFile("passport_sifarish_patra.html")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading passport_sifarish_patra.html: %v\n", err)
		return
	}

	// Render the template with data
	renderedHTML, err := template.RenderHTML(string(templateBytes), passportData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering passport sifarish template: %v\n", err)
		return
	}

	devanagariFace, err := loadTrueTypeFace(
		filepath.Join("examples", "html_invoice", "devanagari", "Lohit-Devanagari.ttf"),
		"./devanagari/Lohit-Devanagari.ttf",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading Devanagari font: %v\n", err)
		return
	}

	opt := html.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Lohit Devanagari",
		PageSize:          [2]float64{595.28, 841.89}, // A4
		Margins:           [4]float64{0, 0, 0, 0},
		FontFaces: map[string]pdffont.Face{
			"Lohit Devanagari":      devanagariFace,
			"Noto Sans Devanagari":  devanagariFace,
			"Noto Serif Devanagari": devanagariFace,
			"Mangal":                devanagariFace,
		},
	}

	ensureOutputDirs(filepath.Join("out", "other"))
	outputPath := envString("PASSPORT_OUTPUT", filepath.Join("out", "other", "passport_sifarish_patra.pdf"))
	if err = pdf.FromLeanHTML(renderedHTML, outputPath, opt); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating passport sifarish PDF: %v\n", err)
		return
	}
	fmt.Printf("Generated %s\n", outputPath)
}

// ---------------------------------------------------------------------------
// Data types
// ---------------------------------------------------------------------------

type Company struct {
	Name, Address, City, State, Zip, Country string
	Phone, Email, Website, TaxID             string
}

type LineItem struct {
	Description string
	Details     string
	Qty         int
	UnitPrice   float64
}

type InvoiceData struct {
	Number, Date, DueDate, Currency string
	From, To                        Company
	Items                           []LineItem
	TaxRate                         float64
	Notes, PaymentInfo              string
}

func loadLogoDataURI() string {
	candidates := []string{
		filepath.Join("examples", "html_invoice", "logo.png"),
		"logo.png",
	}

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	}

	return ""
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func fmtMoney(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	decPart := parts[1]
	n := len(intPart)
	if n <= 3 {
		return "$" + intPart + "." + decPart
	}
	var result []byte
	for i, c := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return "$" + string(result) + "." + decPart
}

func phoneHref(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if (r >= '0' && r <= '9') || r == '+' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return "tel:" + b.String()
}

func websiteHref(site string) string {
	if site == "" {
		return ""
	}
	if strings.HasPrefix(site, "http://") || strings.HasPrefix(site, "https://") {
		return site
	}
	return "https://" + site
}

func loadTrueTypeFace(paths ...string) (pdffont.Face, error) {
	for _, path := range paths {
		face, err := pdffont.LoadTrueTypeFile(path)
		if err == nil {
			return face, nil
		}
	}
	return nil, fmt.Errorf("no usable TrueType font found in %v", paths)
}
