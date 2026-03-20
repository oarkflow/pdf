// Package main demonstrates generating a beautiful, professional invoice PDF
// from HTML+CSS using the github.com/oarkflow/pdf library.
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
	"github.com/oarkflow/pdf/html"
)

func main() {
	invoiceHTML := buildInvoiceHTML(InvoiceData{
		Number:      "INV-2026-0042",
		Date:        time.Now().Format("January 2, 2006"),
		DueDate:     time.Now().AddDate(0, 0, 30).Format("January 2, 2006"),
		Currency:    "USD",
		LogoDataURI: loadLogoDataURI(),
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
	})

	err := pdf.FromHTML(invoiceHTML, "invoice.pdf", html.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Helvetica",
		PageSize:          [2]float64{595.28, 841.89}, // A4
		Margins:           [4]float64{40, 40, 50, 40},
		UseTailwind:       true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Generated invoice.pdf")

	// Also generate PDF from the Tailwind-styled invoice.html file
	htmlBytes, err := os.ReadFile("examples/html_invoice/invoice.html")
	if err != nil {
		// Try from current directory
		htmlBytes, err = os.ReadFile("invoice.html")
	}
	if err == nil {
		err = pdf.FromHTML(string(htmlBytes), "invoice_tailwind.pdf", html.Options{
			DefaultFontSize:   10,
			DefaultFontFamily: "Helvetica",
			PageSize:          [2]float64{595.28, 841.89},
			Margins:           [4]float64{40, 40, 50, 40},
			UseTailwind:       true,
			EnableJavaScript:  true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating tailwind invoice: %v\n", err)
		} else {
			fmt.Println("Generated invoice_tailwind.pdf")
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
	LogoDataURI                     string
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

// ---------------------------------------------------------------------------
// HTML builder
// ---------------------------------------------------------------------------

func buildInvoiceHTML(d InvoiceData) string {
	// Calculate totals
	subtotal := 0.0
	for _, item := range d.Items {
		subtotal += float64(item.Qty) * item.UnitPrice
	}
	tax := subtotal * d.TaxRate / 100
	total := subtotal + tax

	fmtMoney := func(v float64) string {
		// Format with commas as thousands separator
		s := fmt.Sprintf("%.2f", v)
		parts := strings.SplitN(s, ".", 2)
		intPart := parts[0]
		decPart := parts[1]
		// Insert commas
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

	// Build item rows
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

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>Invoice %s</title>
  <meta name="author" content="%s">
  <style>
    @page {
      size: A4;
      margin: 40px;
    }

    body {
      font-family: Helvetica, Arial, sans-serif;
      font-size: 10pt;
      color: #3c4257;
      line-height: 1.5;
      margin: 0;
      padding: 0;
    }

    /* ---------- Header bar ---------- */
    .header {
      display: flex;
      padding: 24px 0 20px 0;
      border-bottom: 3px solid #635bff;
      margin-bottom: 24px;
    }
    .header .brand-wrap {
      max-width: 300px;
    }
    .header .brand-logo {
      width: 180px;
      height: auto;
      margin-bottom: 10px;
    }
    .header .brand {
      font-size: 22pt;
      font-weight: 700;
      color: #1a1f36;
      letter-spacing: -0.5px;
    }
    .header .brand-sub {
      font-size: 8.5pt;
      color: #6b7294;
      margin-top: 2px;
    }
    .header .invoice-title {
      text-align: right;
      margin-left: auto;
    }
    .header .invoice-label {
      font-size: 28pt;
      font-weight: 700;
      color: #635bff;
      letter-spacing: 1px;
    }
    .header .invoice-number {
      font-size: 10pt;
      color: #6b7294;
      margin-top: 2px;
    }

    /* ---------- Info grid ---------- */
    .info-grid {
      display: flex;
      margin-bottom: 28px;
      gap: 20px;
    }
    .info-block {
      flex: 1;
      padding: 16px;
      background-color: #f8f9fb;
      border-radius: 6px;
      border: 1px solid #eaedf1;
    }
    .info-block .label {
      font-size: 7.5pt;
      font-weight: 700;
      color: #635bff;
      text-transform: uppercase;
      letter-spacing: 0.8px;
      margin-bottom: 8px;
    }
    .info-block .name {
      font-size: 11pt;
      font-weight: 700;
      color: #1a1f36;
      margin-bottom: 4px;
    }
    .info-block .detail {
      font-size: 9pt;
      color: #6b7294;
      line-height: 1.6;
    }

    /* ---------- Date strip ---------- */
    .date-strip {
      display: flex;
      background-color: #f0f0ff;
      border-radius: 6px;
      padding: 12px 16px;
      margin-bottom: 24px;
      gap: 40px;
    }
    .date-strip .date-item .date-label {
      font-size: 7.5pt;
      font-weight: 700;
      color: #635bff;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
    .date-strip .date-item .date-value {
      font-size: 10pt;
      font-weight: 600;
      color: #1a1f36;
      margin-top: 2px;
    }

    /* ---------- Table ---------- */
    .items-table {
      width: 100%%;
      border-collapse: collapse;
      margin-bottom: 0;
    }
    .items-table thead th {
      padding: 10px 12px;
      font-size: 7.5pt;
      font-weight: 700;
      color: #ffffff;
      background-color: #635bff;
      text-transform: uppercase;
      letter-spacing: 0.8px;
    }
    .items-table thead th:first-child {
      border-radius: 6px 0 0 0;
      text-align: left;
    }
    .items-table thead th:last-child {
      border-radius: 0 6px 0 0;
      text-align: right;
    }

    /* ---------- Totals ---------- */
    .totals-wrapper {
      display: flex;
      margin-top: 0;
      margin-bottom: 24px;
    }
    .totals-spacer {
      flex: 2;
    }
    .totals-box {
      flex: 1;
      border: 1px solid #eaedf1;
      border-top: none;
      border-radius: 0 0 6px 6px;
      overflow: hidden;
    }
    .totals-row {
      display: flex;
      padding: 8px 12px;
      border-bottom: 1px solid #eaedf1;
    }
    .totals-row .totals-label {
      flex: 1;
      font-size: 9pt;
      color: #6b7294;
    }
    .totals-row .totals-value {
      text-align: right;
      font-size: 9pt;
      font-weight: 600;
      color: #1a1f36;
    }
    .totals-row.grand-total {
      background-color: #635bff;
      padding: 12px;
    }
    .totals-row.grand-total .totals-label {
      color: #ffffff;
      font-weight: 700;
      font-size: 10pt;
    }
    .totals-row.grand-total .totals-value {
      color: #ffffff;
      font-weight: 700;
      font-size: 13pt;
    }

    /* ---------- Footer sections ---------- */
    .footer-grid {
      display: flex;
      gap: 20px;
      margin-top: 12px;
    }
    .footer-block {
      flex: 1;
      padding: 14px;
      background-color: #f8f9fb;
      border-radius: 6px;
      border: 1px solid #eaedf1;
    }
    .footer-block .block-title {
      font-size: 8pt;
      font-weight: 700;
      color: #635bff;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-bottom: 6px;
    }
    .footer-block .block-body {
      font-size: 8.5pt;
      color: #6b7294;
      line-height: 1.6;
    }

    /* ---------- Bottom bar ---------- */
    .bottom-bar {
      margin-top: 24px;
      padding-top: 12px;
      border-top: 2px solid #635bff;
      text-align: center;
      font-size: 7.5pt;
      color: #a3acb9;
    }
  </style>
</head>
<body>

  <!-- ===== HEADER ===== -->
  <div class="header">
    <div class="brand-wrap">
      %s
      <div class="brand">%s</div>
      <div class="brand-sub">%s &bull; %s &bull; %s</div>
    </div>
    <div class="invoice-title">
      <div class="invoice-label">INVOICE</div>
      <div class="invoice-number">%s</div>
    </div>
  </div>

  <!-- ===== DATE STRIP ===== -->
  <div class="date-strip">
    <div class="date-item">
      <div class="date-label">Invoice Date</div>
      <div class="date-value">%s</div>
    </div>
    <div class="date-item">
      <div class="date-label">Due Date</div>
      <div class="date-value">%s</div>
    </div>
    <div class="date-item">
      <div class="date-label">Currency</div>
      <div class="date-value">%s</div>
    </div>
  </div>

  <!-- ===== BILL FROM / TO ===== -->
  <div class="info-grid">
    <div class="info-block">
      <div class="label">Bill From</div>
      <div class="name">%s</div>
      <div class="detail">
        %s<br>
        %s, %s %s<br>
        %s<br>
        %s<br>
        %s
        %s
      </div>
    </div>
    <div class="info-block">
      <div class="label">Bill To</div>
      <div class="name">%s</div>
      <div class="detail">
        %s<br>
        %s, %s %s<br>
        %s<br>
        %s<br>
        %s
      </div>
    </div>
  </div>

  <!-- ===== ITEMS TABLE ===== -->
  <table class="items-table">
    <thead>
      <tr>
        <th style="text-align: left;">Description</th>
        <th style="text-align: center;">Qty</th>
        <th style="text-align: right;">Unit Price</th>
        <th style="text-align: right;">Amount</th>
      </tr>
    </thead>
    <tbody>
      %s
    </tbody>
  </table>

  <!-- ===== TOTALS ===== -->
  <div class="totals-wrapper">
    <div class="totals-spacer"></div>
    <div class="totals-box">
      <div class="totals-row">
        <div class="totals-label">Subtotal</div>
        <div class="totals-value">%s</div>
      </div>
      <div class="totals-row">
        <div class="totals-label">Tax (%.1f%%)</div>
        <div class="totals-value">%s</div>
      </div>
      <div class="totals-row grand-total">
        <div class="totals-label">Total Due</div>
        <div class="totals-value">%s</div>
      </div>
    </div>
  </div>

  <!-- ===== NOTES & PAYMENT ===== -->
  <div class="footer-grid">
    <div class="footer-block">
      <div class="block-title">Notes</div>
      <div class="block-body">%s</div>
    </div>
    <div class="footer-block">
      <div class="block-title">Payment Information</div>
      <div class="block-body">%s</div>
    </div>
  </div>

  <!-- ===== BOTTOM BAR ===== -->
  <div class="bottom-bar">
    %s &bull; %s &bull; %s &bull; Tax ID: %s
  </div>

</body>
</html>`,
		// title, author
		d.Number, d.From.Name,
		// header
		func() string {
			if d.LogoDataURI == "" {
				return ""
			}
			return fmt.Sprintf(`<img class="brand-logo" src="%s" alt="%s logo">`, d.LogoDataURI, d.From.Name)
		}(),
		d.From.Name, d.From.Email, d.From.Phone, d.From.Website,
		// invoice number
		d.Number,
		// dates
		d.Date, d.DueDate, d.Currency,
		// from
		d.From.Name, d.From.Address,
		d.From.City, d.From.State, d.From.Zip, d.From.Country,
		d.From.Phone, d.From.Email,
		func() string {
			if d.From.TaxID != "" {
				return fmt.Sprintf("<br>Tax ID: %s", d.From.TaxID)
			}
			return ""
		}(),
		// to
		d.To.Name, d.To.Address,
		d.To.City, d.To.State, d.To.Zip, d.To.Country,
		d.To.Phone, d.To.Email,
		// items
		itemRows,
		// totals
		fmtMoney(subtotal), d.TaxRate, fmtMoney(tax), fmtMoney(total),
		// notes + payment
		d.Notes, d.PaymentInfo,
		// bottom bar
		d.From.Name, d.From.Address+", "+d.From.City+", "+d.From.State+" "+d.From.Zip,
		d.From.Email, d.From.TaxID,
	)
}
