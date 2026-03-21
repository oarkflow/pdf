package barcode

import "testing"

func TestEAN13CheckDigit(t *testing.T) {
	tests := []struct {
		digits string
		check  int
	}{
		{"590123412345", 7},
		{"400638133393", 1},
	}
	for _, tt := range tests {
		got := ean13CheckDigit(tt.digits)
		if got != tt.check {
			t.Errorf("ean13CheckDigit(%q) = %d, want %d", tt.digits, got, tt.check)
		}
	}
}

func TestEncodeEAN13With12Digits(t *testing.T) {
	e, err := EncodeEAN13("590123412345")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Digits) != 13 {
		t.Errorf("got %d digits", len(e.Digits))
	}
	check := ean13CheckDigit("590123412345")
	if int(e.Digits[12]-'0') != check {
		t.Error("check digit mismatch")
	}
}

func TestEncodeEAN13With13Digits(t *testing.T) {
	digits12 := "590123412345"
	check := ean13CheckDigit(digits12)
	digits13 := digits12 + string(rune('0'+check))
	e, err := EncodeEAN13(digits13)
	if err != nil {
		t.Fatal(err)
	}
	if e.Digits != digits13 {
		t.Errorf("Digits = %q", e.Digits)
	}
}

func TestEncodeEAN13InvalidCheckDigit(t *testing.T) {
	_, err := EncodeEAN13("5901234123450")
	if err == nil {
		t.Error("expected error for invalid check digit")
	}
}

func TestEncodeEAN13NonDigit(t *testing.T) {
	_, err := EncodeEAN13("59012341234A")
	if err == nil {
		t.Error("expected error")
	}
}

func TestEncodeEAN13WrongLength(t *testing.T) {
	_, err := EncodeEAN13("12345")
	if err == nil {
		t.Error("expected error")
	}
}

func TestEAN13BarsLength(t *testing.T) {
	e, err := EncodeEAN13("590123412345")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Bars) != 95 {
		t.Errorf("got %d bars, want 95", len(e.Bars))
	}
}

func TestEAN13SVG(t *testing.T) {
	e, err := EncodeEAN13("590123412345")
	if err != nil {
		t.Fatal(err)
	}
	svg := e.SVG(1.0, 50.0)
	if svg == "" {
		t.Error("SVG is empty")
	}
}
