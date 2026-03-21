package barcode

import "testing"

func TestEncodeCode128Text(t *testing.T) {
	bc, err := EncodeCode128("Hello")
	if err != nil {
		t.Fatal(err)
	}
	if bc.Text != "Hello" {
		t.Errorf("Text = %q", bc.Text)
	}
	if len(bc.Bars) == 0 {
		t.Error("Bars is empty")
	}
}

func TestEncodeCode128Numeric(t *testing.T) {
	bc, err := EncodeCode128("1234")
	if err != nil {
		t.Fatal(err)
	}
	if len(bc.Bars) == 0 {
		t.Error("Bars is empty")
	}
}

func TestEncodeCode128OddDigits(t *testing.T) {
	bc, err := EncodeCode128("123")
	if err != nil {
		t.Fatal(err)
	}
	if bc.Text != "123" {
		t.Errorf("Text = %q", bc.Text)
	}
}

func TestEncodeCode128Empty(t *testing.T) {
	_, err := EncodeCode128("")
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestEncodeCode128Unsupported(t *testing.T) {
	_, err := EncodeCode128("\x01")
	if err == nil {
		t.Error("expected error for unsupported character")
	}
}

func TestCode128Widths(t *testing.T) {
	bc, err := EncodeCode128("AB")
	if err != nil {
		t.Fatal(err)
	}
	widths := bc.Widths()
	if len(widths) == 0 {
		t.Error("Widths returned empty")
	}
	if widths[0] <= 0 {
		t.Errorf("first width = %d", widths[0])
	}
}

func TestCode128SVG(t *testing.T) {
	bc, err := EncodeCode128("Test")
	if err != nil {
		t.Fatal(err)
	}
	svg := bc.SVG(1.0, 50.0)
	if svg == "" {
		t.Error("SVG is empty")
	}
}

func TestCode128Deterministic(t *testing.T) {
	bc1, _ := EncodeCode128("ABC")
	bc2, _ := EncodeCode128("ABC")
	if len(bc1.Bars) != len(bc2.Bars) {
		t.Fatal("non-deterministic bar count")
	}
	for i := range bc1.Bars {
		if bc1.Bars[i] != bc2.Bars[i] {
			t.Fatal("non-deterministic bars")
		}
	}
}
