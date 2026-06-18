// Package main demonstrates a complete invoice workflow using only this
// repository's PDF APIs:
//
//  1. Fill a professional HTML invoice template with {{placeholders}}.
//  2. Generate a PDF from the rendered HTML.
//
// Run:
//
//	go run ./examples/invoice
//
// Output:
//
//	examples/invoice/out/invoice.html
//	examples/invoice/out/invoice.pdf
package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oarkflow/pdf"
	pdfhtml "github.com/oarkflow/pdf/html"
	pdftemplate "github.com/oarkflow/pdf/template"

	_ "embed"
)

//go:embed template.html
var invoiceTemplate string

//go:embed logo.jpg
var logoBytes []byte

type Contact struct {
	Name      string
	Address1  string
	Address2  string
	City      string
	State     string
	ZipCode   string
	Country   string
	Telephone string
	Email     string
	Website   string
	TaxID     string
}

type BankDetail struct {
	AccountName   string
	AccountNumber string
	BankName      string
	BankAddress   string
	SwiftCode     string
}

type Business struct {
	Details      Contact
	BankDetail   BankDetail
	ContactName  string
	ContactEmail string
	ContactPhone string
}

type Customer struct {
	Details Contact
}

type Item struct {
	Description string
	Details     string
	Quantity    float64
	UnitPrice   float64
}

func (i Item) Total() float64 {
	return i.Quantity * i.UnitPrice
}

type Transaction struct {
	Description   string
	PaymentMethod string
	Date          string
	Amount        float64
}

type Invoice struct {
	Number       string
	Status       string
	Currency     string
	PaymentTerms string
	Date         time.Time
	DueDays      int
	TaxRate      float64
	Note         string
	Business     Business
	Customer     Customer
	Items        []Item
	Transactions []Transaction
}

func main() {
	inv := sampleInvoice()

	renderedHTML, err := renderInvoiceHTML(inv)
	if err != nil {
		exitf("render invoice template: %v", err)
	}

	outDir := outputDir()
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		exitf("create output directory: %v", err)
	}

	htmlPath := filepath.Join(outDir, "invoice.html")
	if err := os.WriteFile(htmlPath, []byte(renderedHTML), 0o644); err != nil {
		exitf("write rendered HTML: %v", err)
	}

	pdfPath := filepath.Join(outDir, "invoice.pdf")
	err = pdf.FromLeanHTML(renderedHTML, pdfPath, pdfhtml.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Helvetica",
		PageSize:          [2]float64{595.28, 841.89}, // A4 in points
		Margins:           [4]float64{0, 0, 0, 0},
		MediaType:         "print",
	})
	if err != nil {
		exitf("generate PDF: %v", err)
	}

	fmt.Printf("Generated %s\n", htmlPath)
	fmt.Printf("Generated %s\n", pdfPath)
}

func sampleInvoice() Invoice {
	issueDate := time.Date(2026, time.June, 17, 0, 0, 0, 0, time.Local)

	return Invoice{
		Number:       "INV-2026-0005",
		Status:       "Unpaid",
		Currency:     "$",
		PaymentTerms: "Net 5",
		Date:         issueDate,
		DueDays:      5,
		TaxRate:      0,
		Note:         "Thank you for your business. Please include the invoice number in the transfer remarks.",
		Business: Business{
			Details: Contact{
				Name:      "Orgware Construct Pvt. Ltd.",
				Address1:  "Prachin Marg, Old Baneshwor",
				City:      "Kathmandu",
				ZipCode:   "10",
				Country:   "Nepal",
				Telephone: "+977-1-4497653",
				Email:     "info@orgwareconstruct.com",
				Website:   "orgwareconstruct.com",
				TaxID:     "PAN 609999999",
			},
			BankDetail: BankDetail{
				AccountName:   "ORGWARE CONSTRUCT PVT. LTD.",
				AccountNumber: "08001010007253",
				BankName:      "GLOBAL IME BANK LIMITED",
				BankAddress:   "KAMALADI, 28",
				SwiftCode:     "GLBBNPKA",
			},
			ContactName:  "Sujit Baniya",
			ContactEmail: "s.baniya.np@gmail.com",
			ContactPhone: "+977-9856034616",
		},
		Customer: Customer{
			Details: Contact{
				Name:     "Edelberg + Associates",
				Address1: "1205 Johnson Ferry Rd.",
				Address2: "Suite 136-356",
				City:     "Marietta",
				State:    "GA",
				ZipCode:  "30068",
				Country:  "US",
				Email:    "accounts@edelberg.example",
			},
		},
		Items: []Item{
			{
				Description: "CARE 2.0 Development and Support",
				Details:     "Engineering, maintenance, and release support for May 2026",
				Quantity:    1,
				UnitPrice:   12500,
			},
			{
				Description: "CARE 2.0 Development and Support",
				Details:     "Engineering, maintenance, and release support for June 2026",
				Quantity:    1,
				UnitPrice:   12500,
			},
		},
		Transactions: []Transaction{
			{
				Description:   "Advance payment received for May 2026 support",
				PaymentMethod: "Wire Transfer",
				Date:          "2026-06-10",
				Amount:        7000,
			},
		},
	}
}

func renderInvoiceHTML(inv Invoice) (string, error) {
	subtotal := 0.0
	for _, item := range inv.Items {
		subtotal += item.Total()
	}
	tax := subtotal * inv.TaxRate / 100
	paid := 0.0
	for _, transaction := range inv.Transactions {
		paid += transaction.Amount
	}
	total := subtotal + tax
	balanceDue := total - paid
	dueDate := inv.Date.AddDate(0, 0, inv.DueDays)

	data := map[string]any{
		"invoice_number":        inv.Number,
		"invoice_status":        invoiceStatus(inv, balanceDue),
		"invoice_status_upper":  strings.ToUpper(invoiceStatus(inv, balanceDue)),
		"invoice_status_class":  invoiceStatusClass(inv, balanceDue),
		"invoice_status_color":  invoiceStatusColor(inv, balanceDue),
		"invoice_status_bg":     invoiceStatusBackground(inv, balanceDue),
		"invoice_status_svg":    invoiceStatusSVG(inv, balanceDue),
		"invoice_date":          inv.Date.Format("January 2, 2006"),
		"invoice_due_date":      dueDate.Format("January 2, 2006"),
		"payment_terms":         inv.PaymentTerms,
		"currency":              currencyName(inv.Currency),
		"business_name":         inv.Business.Details.Name,
		"business_logo_src":     logoSrc(),
		"business_address1":     inv.Business.Details.Address1,
		"business_address2":     inv.Business.Details.Address2,
		"business_city_line":    cityLine(inv.Business.Details),
		"business_country":      inv.Business.Details.Country,
		"business_phone":        inv.Business.Details.Telephone,
		"business_phone_href":   phoneHref(inv.Business.Details.Telephone),
		"business_email":        inv.Business.Details.Email,
		"business_email_href":   mailHref(inv.Business.Details.Email),
		"business_website":      inv.Business.Details.Website,
		"business_website_href": websiteHref(inv.Business.Details.Website),
		"business_tax_id":       inv.Business.Details.TaxID,
		"customer_name":         inv.Customer.Details.Name,
		"customer_address1":     inv.Customer.Details.Address1,
		"customer_address2":     inv.Customer.Details.Address2,
		"customer_city_line":    cityLine(inv.Customer.Details),
		"customer_country":      inv.Customer.Details.Country,
		"customer_phone":        inv.Customer.Details.Telephone,
		"customer_phone_href":   phoneHref(inv.Customer.Details.Telephone),
		"customer_email":        inv.Customer.Details.Email,
		"customer_email_href":   mailHref(inv.Customer.Details.Email),
		"items":                 itemRows(inv.Currency, inv.Items),
		"transactions":          transactionRows(inv.Currency, inv.Transactions),
		"has_transactions":      len(inv.Transactions) > 0,
		"no_transactions":       len(inv.Transactions) == 0,
		"subtotal":              money(inv.Currency, subtotal),
		"tax_rate":              fmt.Sprintf("%.2f", inv.TaxRate),
		"tax":                   money(inv.Currency, tax),
		"total":                 money(inv.Currency, total),
		"paid":                  money(inv.Currency, paid),
		"balance_due":           money(inv.Currency, balanceDue),
		"note":                  inv.Note,
		"bank_account_name":     inv.Business.BankDetail.AccountName,
		"bank_account_number":   inv.Business.BankDetail.AccountNumber,
		"bank_name":             inv.Business.BankDetail.BankName,
		"bank_address":          inv.Business.BankDetail.BankAddress,
		"bank_swift_code":       inv.Business.BankDetail.SwiftCode,
		"contact_name":          inv.Business.ContactName,
		"contact_email":         inv.Business.ContactEmail,
		"contact_email_href":    mailHref(inv.Business.ContactEmail),
		"contact_phone":         inv.Business.ContactPhone,
		"contact_phone_href":    phoneHref(inv.Business.ContactPhone),
	}

	rendered, err := pdftemplate.RenderHTML(invoiceTemplate, data)
	if err != nil {
		return "", err
	}
	if strings.Contains(rendered, "{{") {
		return "", fmt.Errorf("template contains unresolved placeholders")
	}
	return rendered, nil
}

func invoiceStatus(inv Invoice, balanceDue float64) string {
	if strings.TrimSpace(inv.Status) != "" {
		return inv.Status
	}
	if balanceDue <= 0 {
		return "Paid"
	}
	return "Unpaid"
}

func invoiceStatusClass(inv Invoice, balanceDue float64) string {
	status := strings.ToLower(invoiceStatus(inv, balanceDue))
	status = strings.ReplaceAll(status, " ", "-")
	switch status {
	case "paid":
		return "paid"
	case "unpaid":
		return "unpaid"
	default:
		return "neutral"
	}
}

func invoiceStatusColor(inv Invoice, balanceDue float64) string {
	switch invoiceStatusClass(inv, balanceDue) {
	case "paid":
		return "#2e7d32"
	case "unpaid":
		return "#b3261e"
	default:
		return "#607d87"
	}
}

func invoiceStatusBackground(inv Invoice, balanceDue float64) string {
	switch invoiceStatusClass(inv, balanceDue) {
	case "paid":
		return "#e8f5e9"
	case "unpaid":
		return "#fdecec"
	default:
		return "#eef3f5"
	}
}

func invoiceStatusSVG(inv Invoice, balanceDue float64) string {
	status := strings.ToUpper(invoiceStatus(inv, balanceDue))
	color := invoiceStatusColor(inv, balanceDue)
	bg := invoiceStatusBackground(inv, balanceDue)
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="116" height="42" viewBox="0 0 116 42">
<path d="M14 7 H116 L102 35 H0 Z" fill="%s" stroke="%s" stroke-width="2"/>
<text x="58" y="25" text-anchor="middle" font-family="Helvetica, Arial, sans-serif" font-size="12" font-weight="700" fill="%s">%s</text>
</svg>`, bg, color, color, xmlEscape(status))
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}

func itemRows(currency string, items []Item) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for index, item := range items {
		rows = append(rows, map[string]any{
			"sn":          index + 1,
			"description": item.Description,
			"details":     item.Details,
			"quantity":    quantity(item.Quantity),
			"unit_price":  money(currency, item.UnitPrice),
			"amount":      money(currency, item.Total()),
		})
	}
	return rows
}

func transactionRows(currency string, transactions []Transaction) []map[string]any {
	rows := make([]map[string]any, 0, len(transactions))
	for _, transaction := range transactions {
		rows = append(rows, map[string]any{
			"date":           transaction.Date,
			"payment_method": transaction.PaymentMethod,
			"description":    transaction.Description,
			"amount":         money(currency, transaction.Amount),
		})
	}
	return rows
}

func cityLine(c Contact) string {
	return strings.TrimSpace(strings.Join(nonEmpty(c.City, c.State, c.ZipCode), " "))
}

func logoSrc() string {
	if len(logoBytes) == 0 {
		return ""
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(logoBytes)
}

func outputDir() string {
	if _, err := os.Stat(filepath.Join("examples", "invoice")); err == nil {
		return filepath.Join("examples", "invoice", "out")
	}
	return "out"
}

func money(currency string, amount float64) string {
	return fmt.Sprintf("%s%s", currency, comma(fmt.Sprintf("%.2f", amount)))
}

func comma(s string) string {
	parts := strings.SplitN(s, ".", 2)
	whole := parts[0]
	sign := ""
	if strings.HasPrefix(whole, "-") {
		sign = "-"
		whole = strings.TrimPrefix(whole, "-")
	}
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if len(parts) == 2 {
		return sign + whole + "." + parts[1]
	}
	return sign + whole
}

func quantity(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%.0f", q)
	}
	return fmt.Sprintf("%.2f", q)
}

func currencyName(symbol string) string {
	switch symbol {
	case "$":
		return "USD"
	case "Rs.", "Rs", "NPR":
		return "NPR"
	default:
		return symbol
	}
}

func websiteHref(site string) string {
	if site == "" {
		return "#"
	}
	if strings.HasPrefix(site, "http://") || strings.HasPrefix(site, "https://") {
		return site
	}
	return "https://" + site
}

func mailHref(email string) string {
	if email == "" {
		return "#"
	}
	return "mailto:" + email
}

func phoneHref(phone string) string {
	if phone == "" {
		return "#"
	}
	replacer := strings.NewReplacer(" ", "", "(", "", ")", "", "-", "")
	return "tel:" + replacer.Replace(phone)
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
