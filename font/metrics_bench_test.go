package font

import "testing"

func BenchmarkTextWidth(b *testing.B) {
	f, err := NewStandardFont("Helvetica")
	if err != nil {
		b.Fatal(err)
	}
	texts := []string{
		"Hello, World!",
		"The quick brown fox jumps over the lazy dog",
		"Invoice #12345 — Total: $1,234.56",
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, text := range texts {
			var width int
			for _, r := range text {
				gid := f.GlyphIndex(r)
				width += f.GlyphAdvance(gid)
			}
		}
	}
}

func BenchmarkFontLookup(b *testing.B) {
	names := []string{
		"Helvetica", "Helvetica-Bold", "Times-Roman", "Times-Bold",
		"Courier", "Courier-Bold", "Symbol", "ZapfDingbats",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, name := range names {
			_, err := NewStandardFont(name)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
