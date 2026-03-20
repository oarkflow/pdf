package main

import (
	"fmt"
	"os"

	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/template"
)

func main() {
	output := "template_example.pdf"
	if len(os.Args) > 1 {
		output = os.Args[1]
	}

	// Create a template with {{key}} placeholders in text
	t := template.New("invoice")
	t.AddSection("header",
		layout.NewHeading(layout.H1, "{{title}}"),
		layout.NewSpacer(8),
		layout.NewParagraph("Invoice #: {{invoice_number}}    Date: {{date}}"),
		layout.NewSpacer(16),
	)
	t.AddSection("body",
		layout.NewHeading(layout.H3, "Bill To: {{customer_name}}"),
		layout.NewParagraph("{{customer_address}}"),
		layout.NewSpacer(12),
		layout.NewParagraph("Amount Due: {{currency}}{{total}}"),
		layout.NewSpacer(8),
		layout.NewParagraph("Payment Terms: {{payment_terms}}"),
		layout.NewSpacer(20),
		layout.NewParagraph("Thank you for your business, {{customer_name}}!"),
	)

	// Data map — keys match the {{key}} placeholders above
	data := map[string]string{
		"title":            "INVOICE",
		"invoice_number":   "INV-2024-001",
		"date":             "March 20, 2026",
		"customer_name":    "Acme Corporation",
		"customer_address": "123 Business Ave, Suite 100, New York, NY 10001",
		"currency":         "$",
		"total":            "1,250.00",
		"payment_terms":    "Net 30",
	}

	if err := t.Execute(data, output); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s\n", output)
}
