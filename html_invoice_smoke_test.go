package pdf

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/core"
	pdffont "github.com/oarkflow/pdf/font"
	"github.com/oarkflow/pdf/html"
	"github.com/oarkflow/pdf/reader"
	pdftemplate "github.com/oarkflow/pdf/template"
)

func TestHTMLInvoiceTemplateSmoke(t *testing.T) {
	templateBytes, err := os.ReadFile("examples/html_invoice/template.html")
	if err != nil {
		t.Fatalf("read invoice template: %v", err)
	}

	rendered, err := pdftemplate.RenderHTML(string(templateBytes), invoiceSmokeData())
	if err != nil {
		t.Fatalf("render invoice template: %v", err)
	}
	if strings.Contains(rendered, "{{") {
		t.Fatal("invoice template still contains unresolved placeholders")
	}

	var out bytes.Buffer
	err = FromHTMLStreaming(rendered, &out, html.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Helvetica",
		PageSize:          [2]float64{595.28, 841.89},
		Margins:           [4]float64{40, 40, 50, 40},
		UseTailwind:       true,
		Encryption: &core.EncryptionConfig{
			Algorithm:     core.AES_128,
			OwnerPassword: "owner-secret",
			UserPassword:  "user-secret",
			Permissions:   0xFFFFF0C4,
		},
	})
	if err != nil {
		t.Fatalf("FromHTMLStreaming invoice: %v", err)
	}

	pdfData := out.Bytes()
	if !bytes.HasPrefix(pdfData, []byte("%PDF-")) {
		t.Fatal("generated invoice is missing PDF header")
	}
	if !bytes.Contains(pdfData, []byte("/Encrypt")) {
		t.Fatal("generated invoice is missing encryption dictionary")
	}

	r, err := reader.OpenWithPassword(pdfData, "user-secret")
	if err != nil {
		t.Fatalf("open generated invoice: %v", err)
	}
	if r.NumPages() == 0 {
		t.Fatal("generated invoice has no pages")
	}
	text, err := r.ExtractText(0)
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}
	for _, want := range []string{"INVOICE", "INV-SMOKE-001", "Acme Corporation", "Wayne Enterprises"} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated invoice text missing %q in %q", want, text)
		}
	}
}

func TestPassportSifarishDevanagariSmoke(t *testing.T) {
	templateBytes, err := os.ReadFile("examples/html_invoice/passport_sifarish_patra.html")
	if err != nil {
		t.Fatalf("read passport template: %v", err)
	}
	rendered, err := pdftemplate.RenderHTML(string(templateBytes), passportSifarishSmokeData())
	if err != nil {
		t.Fatalf("render passport template: %v", err)
	}

	devanagariFace, err := pdffont.LoadTrueTypeFile("examples/html_invoice/devanagari/Lohit-Devanagari.ttf")
	if err != nil {
		t.Fatalf("load Devanagari font: %v", err)
	}

	var out bytes.Buffer
	err = FromHTMLStreaming(rendered, &out, html.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Lohit Devanagari",
		PageSize:          [2]float64{595.28, 841.89},
		Margins:           [4]float64{0, 0, 0, 0},
		FontFaces: map[string]pdffont.Face{
			"Lohit Devanagari":      devanagariFace,
			"Noto Sans Devanagari":  devanagariFace,
			"Noto Serif Devanagari": devanagariFace,
			"Mangal":                devanagariFace,
		},
	})
	if err != nil {
		t.Fatalf("FromHTMLStreaming passport: %v", err)
	}
	if bytes.Contains(out.Bytes(), []byte("????")) {
		t.Fatal("generated passport PDF contains ???? placeholder bytes")
	}

	r, err := reader.Open(out.Bytes())
	if err != nil {
		t.Fatalf("open generated passport PDF: %v", err)
	}
	if r.NumPages() == 0 {
		t.Fatal("generated passport PDF has no pages")
	}
	text, err := r.ExtractText(0)
	if err != nil {
		t.Fatalf("extract passport text: %v", err)
	}
	if strings.Contains(text, "????") {
		t.Fatalf("extracted passport text contains ????: %q", text)
	}
	for _, want := range []string{"बुटवल", "मिति", "रूपन्देही", "२०८२/१२/१७"} {
		if !strings.Contains(text, want) {
			t.Fatalf("passport text missing %q in %q", want, text)
		}
	}
}

func invoiceSmokeData() map[string]any {
	return map[string]any{
		"invoice_number":    "INV-SMOKE-001",
		"from_name":         "Acme Corporation",
		"from_email":        "billing@example.com",
		"from_email_href":   "mailto:billing@example.com",
		"from_phone":        "+1 555 0100",
		"from_phone_href":   "tel:+15550100",
		"from_website":      "example.com",
		"from_website_href": "https://example.com",
		"from_address":      "123 Innovation Drive",
		"from_city":         "San Francisco",
		"from_state":        "CA",
		"from_zip":          "94105",
		"from_country":      "United States",
		"from_tax_id":       "US-87-1234567",
		"from_tax_id_line":  "<br>Tax ID: US-87-1234567",
		"from_full_address": "123 Innovation Drive, San Francisco, CA 94105",
		"to_name":           "Wayne Enterprises",
		"to_address":        "1007 Mountain Drive",
		"to_city":           "Gotham City",
		"to_state":          "NJ",
		"to_zip":            "07001",
		"to_country":        "United States",
		"to_phone":          "+1 555 0142",
		"to_phone_href":     "tel:+15550142",
		"to_email":          "accounts@example.com",
		"to_email_href":     "mailto:accounts@example.com",
		"date":              "May 15, 2026",
		"due_date":          "June 14, 2026",
		"currency":          "USD",
		"logo_html":         "",
		"item_rows": `<tr>
<td style="padding: 10px 12px; border-bottom: 1px solid #eaedf1;">Enterprise Platform License</td>
<td style="padding: 10px 12px; text-align: center;">1</td>
<td style="padding: 10px 12px; text-align: right;">$1,000.00</td>
<td style="padding: 10px 12px; text-align: right;">$1,000.00</td>
</tr>`,
		"subtotal":     "$1,000.00",
		"tax_rate":     "8.5",
		"tax":          "$85.00",
		"total":        "$1,085.00",
		"notes":        "Thank you for your business.",
		"payment_info": "Wire transfer accepted.",
	}
}

func passportSifarishSmokeData() map[string]any {
	return map[string]any{
		"page_margin":                 "0",
		"page_padding":                "0",
		"print_page_padding":          "0",
		"page_width":                  "100vw",
		"page_height":                 "100vh",
		"page_border_top":             "0",
		"letterhead_margin_bottom":    "0",
		"meta_margin":                 "0",
		"subject_margin":              "0",
		"block_margin_bottom":         "0",
		"details_heading_margin":      "0",
		"details_table_margin_bottom": "0",
		"details_cell_padding":        "0",
		"signature_margin_top":        "0",
		"signature_gap":               "0",
		"seal_padding":                "0",
		"footer_margin_top":           "0",
		"footer_padding_top":          "0",
		"municipality_name":           "बुटवल उपमहानगरपालिका",
		"ward_number":                 "१२",
		"district":                    "रूपन्देही",
		"province":                    "लुम्बिनी प्रदेश",
		"office_phone":                "०७१-५४०१२३",
		"office_email":                "ward12@butwalmunicipality.gov.np",
		"letter_serial_no":            "०८७",
		"fiscal_year":                 "२०८२/८३",
		"date_bs":                     "२०८२/१२/१७",
		"date_ad":                     "2026-03-31",
		"applicant_full_name_nepali":  "राम बहादुर खत्री",
		"applicant_full_name_english": "Ram Bahadur Khatri",
		"dob_bs":                      "२०४५/०५/२०",
		"dob_ad":                      "1988-09-05",
		"birth_place":                 "बुटवल, रूपन्देही",
		"gender":                      "पुरुष",
		"citizenship_no":              "१२३-४५६-७८९",
		"citizenship_issued_district": "रूपन्देही",
		"citizenship_issued_date":     "२०६३/०१/१५",
		"father_name":                 "कृष्ण बहादुर खत्री",
		"mother_name":                 "सीता देवी खत्री",
		"grandfather_name":            "धन बहादुर खत्री",
		"spouse_name":                 "गीता खत्री",
		"permanent_ward_no":           "१२",
		"permanent_municipality":      "बुटवल उपमहानगरपालिका",
		"permanent_district":          "रूपन्देही",
		"current_ward_no":             "१२",
		"current_municipality":        "बुटवल उपमहानगरपालिका",
		"current_district":            "रूपन्देही",
		"applicant_phone":             "९८४७०१२३४५",
		"passport_type":               "साधारण (Ordinary)",
		"authorized_officer_name":     "सुरेश कुमार श्रेष्ठ",
	}
}
