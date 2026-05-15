package font

import (
	"path/filepath"
	"testing"
)

func TestTrueTypeFontShapesDevanagari(t *testing.T) {
	face, err := LoadTrueTypeFile(filepath.Join("..", "examples", "html_invoice", "devanagari", "Lohit-Devanagari.ttf"))
	if err != nil {
		t.Fatalf("LoadTrueTypeFile() error = %v", err)
	}

	samples := []string{
		"मिति",
		"रूपन्देही",
		"बुटवल उपमहानगरपालिका",
		"पत्र संख्या: वा.का./पा.सि./०८७/२०८२/८३",
		"खत्री",
		"श्रेष्ठ",
		"कार्यालय",
	}

	for _, sample := range samples {
		t.Run(sample, func(t *testing.T) {
			glyphs, ok := face.ShapeText(sample)
			if !ok {
				t.Fatal("ShapeText() returned ok=false")
			}
			if len(glyphs) == 0 {
				t.Fatal("ShapeText() returned no glyphs")
			}
			totalAdvance := 0
			for _, glyph := range glyphs {
				if glyph.GlyphID == 0 {
					t.Fatalf("shaped glyph has .notdef glyph ID: %+v", glyph)
				}
				totalAdvance += glyph.XAdvance
			}
			if totalAdvance <= 0 {
				t.Fatalf("total advance = %d, want positive", totalAdvance)
			}
		})
	}
}
