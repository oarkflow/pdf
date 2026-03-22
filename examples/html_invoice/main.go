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
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/html"
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
	err = pdf.FromHTML(invoiceHTML, "invoice.pdf", baseOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated invoice.pdf")

	protectedOpt := baseOpt
	protectedOpt.Encryption = &core.EncryptionConfig{
		Algorithm:     core.AES_128,
		OwnerPassword: "owner-invoice-demo-2026",
		UserPassword:  "open-sesame",
		Permissions:   0xFFFFF0C4,
	}
	err = pdf.FromHTML(invoiceHTML, "invoice_protected.pdf", protectedOpt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating protected invoice: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated invoice_protected.pdf (password: open-sesame)")

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
		err = pdf.FromHTML(string(htmlBytes), "invoice_tailwind.pdf", tailwindOpt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating tailwind invoice: %v\n", err)
		} else {
			fmt.Println("Generated invoice_tailwind.pdf")
		}

		protectedTailwindOpt := tailwindOpt
		protectedTailwindOpt.Encryption = &core.EncryptionConfig{
			Algorithm:     core.AES_128,
			OwnerPassword: "owner-newsletter-demo-2026",
			UserPassword:  "open-sesame",
			Permissions:   0xFFFFF0C4,
		}
		err = pdf.FromHTML(string(htmlBytes), "invoice_tailwind_protected.pdf", protectedTailwindOpt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating protected tailwind invoice: %v\n", err)
		} else {
			fmt.Println("Generated invoice_tailwind_protected.pdf (password: open-sesame)")
		}
	}
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
