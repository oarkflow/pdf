package barcode

import (
	"strings"
	"testing"
)

func BenchmarkQREncodeSmall(b *testing.B) {
	data := "Hello"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncodeQR(data, ECMedium)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQREncodeMedium(b *testing.B) {
	data := "https://example.com/invoice/12345?token=abc123def456"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncodeQR(data, ECMedium)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQREncodeLarge(b *testing.B) {
	data := strings.Repeat("A", 500)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncodeQR(data, ECLow)
		if err != nil {
			b.Fatal(err)
		}
	}
}
