package core

import (
	"bytes"
	"testing"
)

func serialize(t *testing.T, obj PdfObject) string {
	t.Helper()
	var buf bytes.Buffer
	_, err := obj.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo error: %v", err)
	}
	return buf.String()
}

func TestPdfBooleanType(t *testing.T) {
	if PdfBoolean(true).Type() != ObjBoolean {
		t.Fatal("expected ObjBoolean")
	}
}

func TestPdfBooleanWriteTo(t *testing.T) {
	tests := []struct {
		val  PdfBoolean
		want string
	}{
		{true, "true"},
		{false, "false"},
	}
	for _, tt := range tests {
		got := serialize(t, tt.val)
		if got != tt.want {
			t.Errorf("PdfBoolean(%v) = %q, want %q", bool(tt.val), got, tt.want)
		}
	}
}

func TestPdfIntegerWriteTo(t *testing.T) {
	tests := []struct {
		val  PdfInteger
		want string
	}{
		{0, "0"},
		{42, "42"},
		{-100, "-100"},
		{9999999, "9999999"},
	}
	for _, tt := range tests {
		got := serialize(t, tt.val)
		if got != tt.want {
			t.Errorf("PdfInteger(%d) = %q, want %q", int64(tt.val), got, tt.want)
		}
	}
}

func TestPdfIntegerType(t *testing.T) {
	if PdfInteger(1).Type() != ObjInteger {
		t.Fatal("expected ObjInteger")
	}
}

func TestPdfNumberWriteTo(t *testing.T) {
	tests := []struct {
		val  PdfNumber
		want string
	}{
		{0, "0.0"},
		{3.14, "3.14"},
		{-1.5, "-1.5"},
		{100, "100.0"},
	}
	for _, tt := range tests {
		got := serialize(t, tt.val)
		if got != tt.want {
			t.Errorf("PdfNumber(%v) = %q, want %q", float64(tt.val), got, tt.want)
		}
	}
}

func TestPdfNumberType(t *testing.T) {
	if PdfNumber(1.0).Type() != ObjNumber {
		t.Fatal("expected ObjNumber")
	}
}

func TestPdfStringWriteTo(t *testing.T) {
	tests := []struct {
		val  PdfString
		want string
	}{
		{"Hello", "(Hello)"},
		{"", "()"},
		{"a(b)c", "(a\\(b\\)c)"},
		{"back\\slash", "(back\\\\slash)"},
		{"line\nfeed", "(line\\nfeed)"},
		{"\t\r", "(\\t\\r)"},
	}
	for _, tt := range tests {
		got := serialize(t, tt.val)
		if got != tt.want {
			t.Errorf("PdfString(%q) = %q, want %q", string(tt.val), got, tt.want)
		}
	}
}

func TestPdfStringType(t *testing.T) {
	if PdfString("x").Type() != ObjString {
		t.Fatal("expected ObjString")
	}
}

func TestPdfStringSpecialChars(t *testing.T) {
	// Control characters below 0x20 should be octal-escaped
	s := PdfString(string([]byte{0x01, 0x7f}))
	got := serialize(t, s)
	if got != "(\\001\\177)" {
		t.Errorf("special chars = %q, want %q", got, "(\\001\\177)")
	}
}

func TestPdfHexStringWriteTo(t *testing.T) {
	tests := []struct {
		val  PdfHexString
		want string
	}{
		{[]byte{}, "<>"},
		{[]byte{0xDE, 0xAD}, "<DEAD>"},
		{[]byte{0x00, 0xFF}, "<00FF>"},
	}
	for _, tt := range tests {
		got := serialize(t, tt.val)
		if got != tt.want {
			t.Errorf("PdfHexString = %q, want %q", got, tt.want)
		}
	}
}

func TestPdfHexStringType(t *testing.T) {
	if PdfHexString(nil).Type() != ObjHexString {
		t.Fatal("expected ObjHexString")
	}
}

func TestPdfNameWriteTo(t *testing.T) {
	tests := []struct {
		val  PdfName
		want string
	}{
		{"Type", "/Type"},
		{"", "/"},
		{"A B", "/A#20B"},
		{"A#B", "/A#23B"},
		{"A/B", "/A#2FB"},
	}
	for _, tt := range tests {
		got := serialize(t, tt.val)
		if got != tt.want {
			t.Errorf("PdfName(%q) = %q, want %q", string(tt.val), got, tt.want)
		}
	}
}

func TestPdfNameType(t *testing.T) {
	if PdfName("X").Type() != ObjName {
		t.Fatal("expected ObjName")
	}
}

func TestPdfArrayWriteTo(t *testing.T) {
	arr := PdfArray{PdfInteger(1), PdfInteger(2), PdfInteger(3)}
	got := serialize(t, arr)
	if got != "[1 2 3]" {
		t.Errorf("array = %q, want %q", got, "[1 2 3]")
	}
}

func TestPdfArrayEmpty(t *testing.T) {
	arr := PdfArray{}
	got := serialize(t, arr)
	if got != "[]" {
		t.Errorf("empty array = %q, want %q", got, "[]")
	}
}

func TestPdfArrayType(t *testing.T) {
	if PdfArray(nil).Type() != ObjArray {
		t.Fatal("expected ObjArray")
	}
}

func TestPdfArrayNested(t *testing.T) {
	inner := PdfArray{PdfInteger(1), PdfInteger(2)}
	outer := PdfArray{inner, PdfName("X")}
	got := serialize(t, outer)
	if got != "[[1 2] /X]" {
		t.Errorf("nested array = %q, want %q", got, "[[1 2] /X]")
	}
}

func TestPdfNullWriteTo(t *testing.T) {
	got := serialize(t, PdfNull{})
	if got != "null" {
		t.Errorf("null = %q, want %q", got, "null")
	}
}

func TestPdfNullType(t *testing.T) {
	if (PdfNull{}).Type() != ObjNull {
		t.Fatal("expected ObjNull")
	}
}
