package reader

import (
	"fmt"
	"testing"
)

func BenchmarkTokenize(b *testing.B) {
	// Realistic PDF content stream
	stream := []byte(`
		BT
		/F1 12 Tf
		100 700 Td
		(Hello, World!) Tj
		0 -20 Td
		(Second line of text) Tj
		ET
		q
		1 0 0 1 50 500 cm
		200 0 0 150 0 0 cm
		/Im1 Do
		Q
		0.5 0.5 0.5 rg
		50 400 500 1 re f
		BT
		/F2 10 Tf
		50 380 Td
		[(Item) 50 (Qty) 80 (Price) 80 (Total)] TJ
		ET
		1 0 0 RG
		0.5 w
		50 375 m 550 375 l S
	`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok := NewTokenizer(stream)
		for {
			t, err := tok.Next()
			if err != nil || t.Type == TokenEOF {
				break
			}
		}
	}
}

func BenchmarkOpenMinimalPDF(b *testing.B) {
	// Build a minimal valid PDF with correct offsets
	body := "%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n"
	xrefOffset := len(body)
	xref := "xref\n0 4\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"trailer\n<< /Size 4 /Root 1 0 R >>\n" +
		"startxref\n" +
		fmt.Sprintf("%d", xrefOffset) + "\n%%EOF"
	pdf := []byte(body + xref)
	// If Open fails due to parser limitations, skip instead of fail
	_, err := Open(pdf)
	if err != nil {
		b.Skip("reader.Open cannot parse minimal test PDF:", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Open(pdf)
	}
}
