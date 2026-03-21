package barcode

import "testing"

func TestEncodePDF417Basic(t *testing.T) {
	p, err := EncodePDF417("HELLO", 2)
	if err != nil {
		t.Fatal(err)
	}
	if p.Rows < 3 {
		t.Errorf("Rows = %d, want >= 3", p.Rows)
	}
	if p.Cols < 1 {
		t.Errorf("Cols = %d, want >= 1", p.Cols)
	}
	if len(p.Matrix) != p.Rows {
		t.Errorf("Matrix rows = %d, want %d", len(p.Matrix), p.Rows)
	}
}

func TestEncodePDF417ECLevels(t *testing.T) {
	for ec := 0; ec <= 8; ec++ {
		_, err := EncodePDF417("TEST", ec)
		if err != nil {
			t.Errorf("ecLevel %d: %v", ec, err)
		}
	}
}

func TestEncodePDF417InvalidECLevel(t *testing.T) {
	_, err := EncodePDF417("X", 9)
	if err == nil {
		t.Error("expected error for ecLevel 9")
	}
	_, err = EncodePDF417("X", -1)
	if err == nil {
		t.Error("expected error for ecLevel -1")
	}
}

func TestEncodePDF417Empty(t *testing.T) {
	_, err := EncodePDF417("", 0)
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestEncodePDF417SVG(t *testing.T) {
	p, err := EncodePDF417("SVG TEST", 1)
	if err != nil {
		t.Fatal(err)
	}
	svg := p.SVG(1.0, 3.0)
	if svg == "" {
		t.Error("SVG is empty")
	}
}

func TestEncodePDF417Codewords(t *testing.T) {
	p, err := EncodePDF417("ABC", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Codewords) == 0 {
		t.Error("Codewords is empty")
	}
	// First codeword is the length descriptor
	if p.Codewords[0] <= 0 {
		t.Error("length descriptor should be > 0")
	}
}

func TestPDF417MatrixWidth(t *testing.T) {
	p, err := EncodePDF417("WIDTH", 1)
	if err != nil {
		t.Fatal(err)
	}
	expectedWidth := 17 + 17 + p.Cols*17 + 17 + 18
	for r, row := range p.Matrix {
		if len(row) != expectedWidth {
			t.Errorf("row %d width = %d, want %d", r, len(row), expectedWidth)
		}
	}
}
