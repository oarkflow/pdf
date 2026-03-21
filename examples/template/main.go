package main

import (
	"fmt"
	"os"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/html"
)

func main() {
	// HTML template using fasttpl syntax with conditions, ranges, and filters
	htmlTemplate := `
<html>
<head>
  <title>{{ title | upper }}</title>
  <style>
    body { font-family: Helvetica, sans-serif; color: #333; }
    h1 { color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 8px; }
    .info { margin: 12px 0; }
    .admin-badge { background: #e74c3c; color: white; padding: 2px 8px; border-radius: 4px; font-size: 10px; }
    table { width: 100%; border-collapse: collapse; margin: 16px 0; }
    th { background: #3498db; color: white; padding: 8px 12px; text-align: left; }
    td { padding: 8px 12px; border-bottom: 1px solid #eee; }
    tr:nth-child(even) { background: #f8f9fa; }
    .total { font-size: 14pt; font-weight: bold; color: #2c3e50; margin-top: 16px; }
    .notes { background: #fef9e7; border-left: 4px solid #f1c40f; padding: 12px; margin-top: 20px; }
  </style>
</head>
<body>
  <h1>{{ title }}</h1>

  <div class="info">
    <strong>Invoice:</strong> {{ invoice.number }}<br>
    <strong>Date:</strong> {{ invoice.date }}<br>
    <strong>Due:</strong> {{ invoice.due_date }}
  </div>

  {{ if customer.is_vip }}
  <div class="admin-badge">VIP Customer</div>
  {{ end }}

  <div class="info">
    <strong>Bill To:</strong> {{ customer.name }}<br>
    {{ customer.address }}<br>
    {{ customer.city }}, {{ customer.state }} {{ customer.zip }}
  </div>

  <table>
    <tr>
      <th>Item</th>
      <th>Qty</th>
      <th>Price</th>
      <th>Total</th>
    </tr>
    {{ range item in items }}
    <tr>
      <td>{{ $item.name }}</td>
      <td>{{ $item.qty }}</td>
      <td>${{ $item.price }}</td>
      <td>${{ $item.total }}</td>
    </tr>
    {{ end }}
  </table>

  <div class="total">Total: ${{ grand_total }}</div>

  {{ if notes }}
  <div class="notes">
    <strong>Notes:</strong> {{ notes }}
  </div>
  {{ end }}
</body>
</html>`

	data := map[string]any{
		"title": "Invoice",
		"invoice": map[string]any{
			"number":   "INV-2026-0042",
			"date":     "March 21, 2026",
			"due_date": "April 20, 2026",
		},
		"customer": map[string]any{
			"name":    "Acme Corporation",
			"address": "123 Business Ave, Suite 100",
			"city":    "New York",
			"state":   "NY",
			"zip":     "10001",
			"is_vip":  true,
		},
		"items": []map[string]any{
			{"name": "Enterprise License", "qty": 1, "price": "24,999.00", "total": "24,999.00"},
			{"name": "Custom Integration", "qty": 2, "price": "4,500.00", "total": "9,000.00"},
			{"name": "Premium Support", "qty": 1, "price": "7,200.00", "total": "7,200.00"},
		},
		"grand_total": "41,199.00",
		"notes":       "Payment is due within 30 days. Thank you for your business!",
	}

	output := "template_invoice.pdf"
	if len(os.Args) > 1 {
		output = os.Args[1]
	}

	err := pdf.FromHTMLTemplate(htmlTemplate, data, output, html.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Helvetica",
		PageSize:          [2]float64{595.28, 841.89},
		Margins:           [4]float64{40, 40, 50, 40},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s with fasttpl template (conditions, ranges, filters)\n", output)
}
