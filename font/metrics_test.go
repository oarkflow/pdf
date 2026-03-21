package font

import (
	"testing"
)

func TestStandardFont_TextWidth(t *testing.T) {
	f, _ := NewStandardFont("Helvetica")
	// Measure "Hello" manually: H=722 e=556 l=222 l=222 o=556 = 2278
	text := "Hello"
	var total int
	for _, r := range text {
		idx := f.GlyphIndex(r)
		total += f.GlyphAdvance(idx)
	}
	if total != 2278 {
		t.Errorf("Hello width = %d, want 2278", total)
	}
}

func TestStandardFont_TextWidth_Empty(t *testing.T) {
	f, _ := NewStandardFont("Helvetica")
	var total int
	for _, r := range "" {
		idx := f.GlyphIndex(r)
		total += f.GlyphAdvance(idx)
	}
	if total != 0 {
		t.Error("empty string should have 0 width")
	}
}

func TestStandardFont_BBox(t *testing.T) {
	f, _ := NewStandardFont("Helvetica")
	bbox := f.BBox()
	// BBox should have min < max
	if bbox[0] >= bbox[2] {
		t.Errorf("bbox x: min=%d >= max=%d", bbox[0], bbox[2])
	}
	if bbox[1] >= bbox[3] {
		t.Errorf("bbox y: min=%d >= max=%d", bbox[1], bbox[3])
	}
}

func TestStandardFont_StemV(t *testing.T) {
	tests := []struct {
		name  string
		stemV int
	}{
		{"Helvetica", 88},
		{"Helvetica-Bold", 140},
		{"Times-Roman", 87},
		{"Courier", 51},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := NewStandardFont(tt.name)
			if f.StemV() != tt.stemV {
				t.Errorf("StemV() = %d, want %d", f.StemV(), tt.stemV)
			}
		})
	}
}

func TestStandardFont_Flags(t *testing.T) {
	f, _ := NewStandardFont("Courier")
	// Courier should have fixed-pitch flag (bit 0 = 1)
	if f.Flags()&1 == 0 {
		t.Error("Courier should have fixed-pitch flag")
	}
}

func TestStandardFont_GlyphAdvance_OutOfRange(t *testing.T) {
	f, _ := NewStandardFont("Helvetica")
	if w := f.GlyphAdvance(256); w != 0 {
		t.Errorf("GlyphAdvance(256) = %d, want 0", w)
	}
	if w := f.GlyphAdvance(0); w != 0 {
		t.Errorf("GlyphAdvance(0) = %d, want 0 (control char)", w)
	}
}

func TestFaceInterface(t *testing.T) {
	// Verify StandardFont implements Face
	var _ Face = (*StandardFont)(nil)
}

func TestStandardFont_WidthConsistency(t *testing.T) {
	// Times-Roman 'a' should be 444
	f, _ := NewStandardFont("Times-Roman")
	idx := f.GlyphIndex('a')
	if w := f.GlyphAdvance(idx); w != 444 {
		t.Errorf("Times-Roman 'a' width = %d, want 444", w)
	}
}
