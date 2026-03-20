package template

import (
	"fmt"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
)

// InvoiceData holds all data needed to render an invoice.
type InvoiceData struct {
	Number      string
	Date        string
	DueDate     string
	From        Company
	To          Company
	Items       []InvoiceItem
	Subtotal    string
	Tax         string
	TaxRate     string
	Total       string
	Notes       string
	Currency    string
	PaymentInfo string
	Logo        []byte
}

// Company represents a company or person's contact information.
type Company struct {
	Name    string
	Address string
	City    string
	State   string
	Zip     string
	Country string
	Phone   string
	Email   string
}

func (c Company) fullAddress() string {
	line := c.Address
	if c.City != "" {
		line += ", " + c.City
	}
	if c.State != "" {
		line += ", " + c.State
	}
	if c.Zip != "" {
		line += " " + c.Zip
	}
	if c.Country != "" {
		line += ", " + c.Country
	}
	return line
}

// InvoiceItem represents a line item on an invoice.
type InvoiceItem struct {
	Description string
	Quantity    string
	UnitPrice   string
	Amount      string
}

// NewInvoiceTemplate creates a template configured for invoices.
func NewInvoiceTemplate() *Template {
	return New("invoice").
		SetPageSize(document.A4).
		SetMargins(document.Margins{Top: 50, Right: 50, Bottom: 50, Left: 50})
}

// RenderInvoice renders an invoice from the given data.
func RenderInvoice(data InvoiceData) (*document.Document, error) {
	t := NewInvoiceTemplate()

	currency := data.Currency
	if currency == "" {
		currency = "$"
	}

	var elements []layout.Element

	// Header: INVOICE title
	elements = append(elements, layout.NewHeading(layout.H1, "INVOICE"))
	elements = append(elements, layout.NewSpacer(4))

	// Invoice details line
	details := fmt.Sprintf("Invoice #: %s    Date: %s    Due: %s", data.Number, data.Date, data.DueDate)
	elements = append(elements, layout.NewParagraph(details))
	elements = append(elements, layout.NewSpacer(16))

	// Bill From / Bill To side by side using flex
	fromBlock := layout.NewDiv(
		layout.NewHeading(layout.H4, "Bill From"),
		layout.NewParagraph(data.From.Name),
		layout.NewParagraph(data.From.fullAddress()),
		contactParagraph(data.From.Phone, data.From.Email),
	)
	toBlock := layout.NewDiv(
		layout.NewHeading(layout.H4, "Bill To"),
		layout.NewParagraph(data.To.Name),
		layout.NewParagraph(data.To.fullAddress()),
		contactParagraph(data.To.Phone, data.To.Email),
	)
	elements = append(elements, layout.NewFlex(layout.FlexRow, fromBlock, toBlock))
	elements = append(elements, layout.NewSpacer(20))

	// Items table
	cols := 4
	table := layout.NewTable(cols)
	table.CellPadding = 6
	table.BorderWidth = 0.5
	table.BorderColor = [3]float64{0.8, 0.8, 0.8}
	headerBg := [3]float64{0.93, 0.93, 0.93}
	table.HeaderBg = &headerBg
	table.AddHeader("Description", "Quantity", "Unit Price", "Amount")
	for _, item := range data.Items {
		table.AddRow(item.Description, item.Quantity, item.UnitPrice, item.Amount)
	}
	elements = append(elements, table)
	elements = append(elements, layout.NewSpacer(8))

	// Totals
	totalTable := layout.NewTable(2)
	totalTable.CellPadding = 4
	totalTable.BorderWidth = 0
	totalTable.SetColumnWidths(370, 125)
	totalTable.AddRow("", "Subtotal: "+currency+data.Subtotal)
	if data.Tax != "" {
		taxLabel := "Tax"
		if data.TaxRate != "" {
			taxLabel += " (" + data.TaxRate + ")"
		}
		totalTable.AddRow("", taxLabel+": "+currency+data.Tax)
	}
	totalTable.AddRow("", "Total: "+currency+data.Total)
	elements = append(elements, totalTable)
	elements = append(elements, layout.NewSpacer(20))

	// Notes
	if data.Notes != "" {
		elements = append(elements, layout.NewHeading(layout.H4, "Notes"))
		elements = append(elements, layout.NewParagraph(data.Notes))
		elements = append(elements, layout.NewSpacer(12))
	}

	// Payment info
	if data.PaymentInfo != "" {
		elements = append(elements, layout.NewHeading(layout.H4, "Payment Information"))
		elements = append(elements, layout.NewParagraph(data.PaymentInfo))
	}

	t.AddSection("invoice", elements...)
	return t.Render(nil)
}

func contactParagraph(phone, email string) *layout.Paragraph {
	parts := ""
	if phone != "" {
		parts += phone
	}
	if email != "" {
		if parts != "" {
			parts += " | "
		}
		parts += email
	}
	if parts == "" {
		parts = " "
	}
	return layout.NewParagraph(parts)
}
