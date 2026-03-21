package html

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func BenchmarkConvertSimpleHTML(b *testing.B) {
	html := `<html><body><p>Hello, World! This is a simple paragraph for benchmarking.</p></body></html>`
	opts := Options{UseTailwind: false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Convert(html, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertTableHTML(b *testing.B) {
	html := `<html><body><table>`
	for i := 0; i < 10; i++ {
		html += `<tr><td>Name</td><td>Value</td><td>Description</td><td>Amount</td></tr>`
	}
	html += `</table></body></html>`
	opts := Options{UseTailwind: false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Convert(html, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertFlexLayout(b *testing.B) {
	html := `<html><body><div style="display:flex; gap:10px;">`
	for i := 0; i < 8; i++ {
		html += `<div style="flex:1; padding:10px;">Flex child content here</div>`
	}
	html += `</div></body></html>`
	opts := Options{UseTailwind: false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Convert(html, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertTailwindClasses(b *testing.B) {
	html := `<html><body>
		<div class="flex items-center justify-between p-4 m-2 bg-blue-500 text-white rounded-lg shadow-md border border-gray-200 w-full max-w-lg mx-auto text-center font-bold text-xl leading-tight tracking-wide overflow-hidden relative z-10">
			Hello Tailwind
		</div>
	</body></html>`
	opts := Options{UseTailwind: true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Convert(html, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConvertFullInvoice(b *testing.B) {
	// Try to load from examples directory
	_, thisFile, _, _ := runtime.Caller(0)
	invoicePath := filepath.Join(filepath.Dir(thisFile), "..", "examples", "html_invoice", "invoice.html")
	data, err := os.ReadFile(invoicePath)
	var htmlStr string
	if err != nil {
		// Fallback inline invoice
		htmlStr = `<html><body>
			<h1>Invoice #1001</h1>
			<div style="display:flex; justify-content:space-between;">
				<div><strong>From:</strong> Acme Corp<br>123 Main St</div>
				<div><strong>To:</strong> Client Inc<br>456 Oak Ave</div>
			</div>
			<table style="width:100%; border-collapse:collapse; margin-top:20px;">
				<thead><tr><th>Item</th><th>Qty</th><th>Price</th><th>Total</th></tr></thead>
				<tbody>
				<tr><td>Widget A</td><td>10</td><td>$25.00</td><td>$250.00</td></tr>
				<tr><td>Widget B</td><td>5</td><td>$50.00</td><td>$250.00</td></tr>
				<tr><td>Service C</td><td>1</td><td>$100.00</td><td>$100.00</td></tr>
				<tr><td>Widget D</td><td>20</td><td>$12.50</td><td>$250.00</td></tr>
				<tr><td>Support Plan</td><td>1</td><td>$150.00</td><td>$150.00</td></tr>
				</tbody>
				<tfoot><tr><td colspan="3">Total</td><td>$1000.00</td></tr></tfoot>
			</table>
			<p style="margin-top:20px;">Payment due within 30 days. Thank you for your business.</p>
		</body></html>`
	} else {
		htmlStr = string(data)
	}
	opts := Options{UseTailwind: false}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Convert(htmlStr, opts)
		if err != nil {
			b.Fatal(err)
		}
	}
}
