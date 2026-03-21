package font

import (
	"testing"
)

func TestIsStandardFont(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Helvetica", true},
		{"Helvetica-Bold", true},
		{"Times-Roman", true},
		{"Courier", true},
		{"Symbol", true},
		{"ZapfDingbats", true},
		{"Arial", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStandardFont(tt.name); got != tt.want {
				t.Errorf("IsStandardFont(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestNewStandardFont(t *testing.T) {
	f, err := NewStandardFont("Helvetica")
	if err != nil {
		t.Fatalf("NewStandardFont(Helvetica) error = %v", err)
	}
	if f.PostScriptName() != "Helvetica" {
		t.Errorf("name = %q", f.PostScriptName())
	}
	if f.UnitsPerEm() != 1000 {
		t.Errorf("upem = %d", f.UnitsPerEm())
	}
}

func TestNewStandardFont_Invalid(t *testing.T) {
	_, err := NewStandardFont("NotAFont")
	if err == nil {
		t.Error("expected error for invalid font")
	}
}

func TestStandardFont_Metrics(t *testing.T) {
	tests := []struct {
		name      string
		ascent    int
		descent   int
		capHeight int
	}{
		{"Helvetica", 718, -207, 718},
		{"Times-Roman", 683, -217, 662},
		{"Courier", 629, -157, 562},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _ := NewStandardFont(tt.name)
			if f.Ascent() != tt.ascent {
				t.Errorf("Ascent() = %d, want %d", f.Ascent(), tt.ascent)
			}
			if f.Descent() != tt.descent {
				t.Errorf("Descent() = %d, want %d", f.Descent(), tt.descent)
			}
			if f.CapHeight() != tt.capHeight {
				t.Errorf("CapHeight() = %d, want %d", f.CapHeight(), tt.capHeight)
			}
		})
	}
}

func TestStandardFont_CharWidths(t *testing.T) {
	f, _ := NewStandardFont("Helvetica")
	// Space is 278
	spaceIdx := f.GlyphIndex(' ')
	if w := f.GlyphAdvance(spaceIdx); w != 278 {
		t.Errorf("space width = %d, want 278", w)
	}
	// 'A' is 667
	aIdx := f.GlyphIndex('A')
	if w := f.GlyphAdvance(aIdx); w != 667 {
		t.Errorf("A width = %d, want 667", w)
	}
}

func TestCourier_Monospaced(t *testing.T) {
	f, _ := NewStandardFont("Courier")
	for _, r := range "ABCxyz123" {
		idx := f.GlyphIndex(r)
		if w := f.GlyphAdvance(idx); w != 600 {
			t.Errorf("Courier %c width = %d, want 600", r, w)
		}
	}
}

func TestStandardFont_GlyphIndex(t *testing.T) {
	f, _ := NewStandardFont("Helvetica")
	// ASCII chars map directly
	if idx := f.GlyphIndex('A'); idx != 65 {
		t.Errorf("GlyphIndex(A) = %d, want 65", idx)
	}
	// Euro sign maps to 128 in WinAnsi
	if idx := f.GlyphIndex('\u20AC'); idx != 128 {
		t.Errorf("GlyphIndex(Euro) = %d, want 128", idx)
	}
}

func TestStandardFont_Properties(t *testing.T) {
	f, _ := NewStandardFont("Times-Italic")
	if f.ItalicAngle() != -15.5 {
		t.Errorf("ItalicAngle() = %v, want -15.5", f.ItalicAngle())
	}
	if f.RawData() != nil {
		t.Error("RawData should be nil for standard fonts")
	}
	if f.NumGlyphs() != 256 {
		t.Errorf("NumGlyphs() = %d, want 256", f.NumGlyphs())
	}
	if f.Kern(65, 66) != 0 {
		t.Error("standard fonts have no kerning")
	}
}

func TestAll14StandardFonts(t *testing.T) {
	for _, name := range standardFontNames {
		t.Run(name, func(t *testing.T) {
			f, err := NewStandardFont(name)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if f.PostScriptName() != name {
				t.Errorf("name = %q, want %q", f.PostScriptName(), name)
			}
			if f.Ascent() == 0 {
				t.Error("ascent should not be 0")
			}
		})
	}
}
