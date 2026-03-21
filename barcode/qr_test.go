package barcode

import "testing"

func TestEncodeQRNumeric(t *testing.T) {
	qr, err := EncodeQR("01234567", ECMedium)
	if err != nil {
		t.Fatal(err)
	}
	if qr.Version < 1 {
		t.Error("version should be >= 1")
	}
	if qr.Size != qr.Version*4+17 {
		t.Errorf("size %d != version*4+17 (%d)", qr.Size, qr.Version*4+17)
	}
}

func TestEncodeQRAlphanumeric(t *testing.T) {
	qr, err := EncodeQR("HELLO WORLD", ECLow)
	if err != nil {
		t.Fatal(err)
	}
	if qr.Size <= 0 {
		t.Error("invalid size")
	}
	if len(qr.Modules) != qr.Size {
		t.Error("modules row count mismatch")
	}
}

func TestEncodeQRByte(t *testing.T) {
	qr, err := EncodeQR("hello world!", ECQuartile)
	if err != nil {
		t.Fatal(err)
	}
	if qr.ECLevel != ECQuartile {
		t.Errorf("ECLevel = %d, want %d", qr.ECLevel, ECQuartile)
	}
}

func TestEncodeQRErrorCorrectionLevels(t *testing.T) {
	for _, ec := range []ECLevel{ECLow, ECMedium, ECQuartile, ECHigh} {
		qr, err := EncodeQR("TEST", ec)
		if err != nil {
			t.Errorf("ECLevel %d: %v", ec, err)
			continue
		}
		if qr.ECLevel != ec {
			t.Errorf("ECLevel = %d, want %d", qr.ECLevel, ec)
		}
	}
}

func TestEncodeQRSingleChar(t *testing.T) {
	qr, err := EncodeQR("A", ECLow)
	if err != nil {
		t.Fatal(err)
	}
	if qr.Version != 1 {
		t.Errorf("Version = %d, want 1", qr.Version)
	}
}

func TestQRPickMode(t *testing.T) {
	if m := qrPickMode("12345"); m != qrModeNumeric {
		t.Errorf("got %d, want numeric", m)
	}
	if m := qrPickMode("HELLO"); m != qrModeAlphanumeric {
		t.Errorf("got %d, want alphanumeric", m)
	}
	if m := qrPickMode("hello"); m != qrModeByte {
		t.Errorf("got %d, want byte", m)
	}
}

func TestQRRender(t *testing.T) {
	qr, err := EncodeQR("test", ECLow)
	if err != nil {
		t.Fatal(err)
	}
	modules := qr.Render()
	if len(modules) != qr.Size {
		t.Error("Render returned wrong size")
	}
}

func TestQRSVG(t *testing.T) {
	qr, err := EncodeQR("SVG", ECLow)
	if err != nil {
		t.Fatal(err)
	}
	svg := qr.SVG(4)
	if svg == "" {
		t.Error("SVG output is empty")
	}
	if len(svg) < 50 {
		t.Error("SVG output seems too short")
	}
}

func TestQRModulesSquare(t *testing.T) {
	qr, err := EncodeQR("square", ECMedium)
	if err != nil {
		t.Fatal(err)
	}
	for i, row := range qr.Modules {
		if len(row) != qr.Size {
			t.Errorf("row %d has %d cols, want %d", i, len(row), qr.Size)
		}
	}
}
