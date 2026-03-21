package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestPdfIndirectReferenceType(t *testing.T) {
	r := PdfIndirectReference{ObjectNumber: 1, GenerationNumber: 0}
	if r.Type() != ObjReference {
		t.Fatal("expected ObjReference")
	}
}

func TestPdfIndirectReferenceWriteTo(t *testing.T) {
	tests := []struct {
		obj, gen int
		want     string
	}{
		{1, 0, "1 0 R"},
		{42, 3, "42 3 R"},
		{0, 0, "0 0 R"},
	}
	for _, tt := range tests {
		r := PdfIndirectReference{ObjectNumber: tt.obj, GenerationNumber: tt.gen}
		got := serialize(t, r)
		if got != tt.want {
			t.Errorf("ref(%d,%d) = %q, want %q", tt.obj, tt.gen, got, tt.want)
		}
	}
}

func TestPdfIndirectObjectWriteTo(t *testing.T) {
	obj := &PdfIndirectObject{
		Reference: PdfIndirectReference{ObjectNumber: 5, GenerationNumber: 0},
		Object:    PdfInteger(42),
	}
	var buf bytes.Buffer
	obj.WriteTo(&buf)
	s := buf.String()
	if !strings.Contains(s, "5 0 obj") {
		t.Errorf("missing obj header in %q", s)
	}
	if !strings.Contains(s, "42") {
		t.Errorf("missing object value in %q", s)
	}
	if !strings.Contains(s, "endobj") {
		t.Errorf("missing endobj in %q", s)
	}
}

func TestPdfIndirectObjectNilObject(t *testing.T) {
	obj := &PdfIndirectObject{
		Reference: PdfIndirectReference{ObjectNumber: 1, GenerationNumber: 0},
		Object:    nil,
	}
	if obj.Type() != ObjNull {
		t.Error("nil object should return ObjNull type")
	}
	var buf bytes.Buffer
	obj.WriteTo(&buf)
	s := buf.String()
	if !strings.Contains(s, "1 0 obj") {
		t.Error("missing obj header")
	}
	if !strings.Contains(s, "endobj") {
		t.Error("missing endobj")
	}
}

func TestPdfIndirectObjectTypeDelegate(t *testing.T) {
	obj := &PdfIndirectObject{
		Reference: PdfIndirectReference{ObjectNumber: 1},
		Object:    PdfName("Test"),
	}
	if obj.Type() != ObjName {
		t.Errorf("type = %v, want ObjName", obj.Type())
	}
}

func TestPdfIndirectObjectWithDict(t *testing.T) {
	d := NewDictionary()
	d.Set("Type", PdfName("Catalog"))
	obj := &PdfIndirectObject{
		Reference: PdfIndirectReference{ObjectNumber: 3, GenerationNumber: 0},
		Object:    d,
	}
	var buf bytes.Buffer
	obj.WriteTo(&buf)
	s := buf.String()
	if !strings.Contains(s, "3 0 obj") || !strings.Contains(s, "/Catalog") {
		t.Errorf("unexpected output: %q", s)
	}
}
