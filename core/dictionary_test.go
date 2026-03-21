package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewDictionary(t *testing.T) {
	d := NewDictionary()
	if d.Len() != 0 {
		t.Fatalf("new dict len = %d, want 0", d.Len())
	}
	if d.Type() != ObjDictionary {
		t.Fatal("expected ObjDictionary")
	}
}

func TestDictionarySetAndGet(t *testing.T) {
	d := NewDictionary()
	d.Set("Type", PdfName("Catalog"))
	d.Set("Pages", PdfInteger(2))

	v := d.Get("Type")
	if v == nil {
		t.Fatal("Get Type returned nil")
	}
	if name, ok := v.(PdfName); !ok || name != "Catalog" {
		t.Errorf("Get Type = %v, want PdfName(Catalog)", v)
	}
	if d.Len() != 2 {
		t.Errorf("Len = %d, want 2", d.Len())
	}
}

func TestDictionarySetOverwrite(t *testing.T) {
	d := NewDictionary()
	d.Set("Key", PdfInteger(1))
	d.Set("Key", PdfInteger(2))
	if d.Len() != 1 {
		t.Errorf("Len = %d after overwrite, want 1", d.Len())
	}
	v := d.Get("Key").(PdfInteger)
	if v != 2 {
		t.Errorf("overwritten value = %d, want 2", v)
	}
}

func TestDictionaryGetMissing(t *testing.T) {
	d := NewDictionary()
	if d.Get("missing") != nil {
		t.Error("expected nil for missing key")
	}
}

func TestDictionaryHas(t *testing.T) {
	d := NewDictionary()
	d.Set("A", PdfInteger(1))
	if !d.Has("A") {
		t.Error("Has(A) should be true")
	}
	if d.Has("B") {
		t.Error("Has(B) should be false")
	}
}

func TestDictionaryRemove(t *testing.T) {
	d := NewDictionary()
	d.Set("A", PdfInteger(1))
	d.Set("B", PdfInteger(2))
	d.Remove("A")
	if d.Has("A") {
		t.Error("A should be removed")
	}
	if d.Len() != 1 {
		t.Errorf("Len = %d after remove, want 1", d.Len())
	}
}

func TestDictionaryRemoveMissing(t *testing.T) {
	d := NewDictionary()
	d.Remove("nonexistent") // should not panic
}

func TestDictionaryKeys(t *testing.T) {
	d := NewDictionary()
	d.Set("C", PdfInteger(3))
	d.Set("A", PdfInteger(1))
	d.Set("B", PdfInteger(2))
	keys := d.Keys()
	if len(keys) != 3 || keys[0] != "C" || keys[1] != "A" || keys[2] != "B" {
		t.Errorf("Keys = %v, want [C A B]", keys)
	}
}

func TestDictionaryWriteTo(t *testing.T) {
	d := NewDictionary()
	d.Set("Type", PdfName("Catalog"))
	var buf bytes.Buffer
	_, err := d.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.HasPrefix(s, "<<") || !strings.HasSuffix(s, " >>") {
		t.Errorf("dict output = %q, expected << ... >>", s)
	}
	if !strings.Contains(s, "/Type") || !strings.Contains(s, "/Catalog") {
		t.Errorf("dict output missing expected content: %q", s)
	}
}

func TestDictionaryWriteToEmpty(t *testing.T) {
	d := NewDictionary()
	var buf bytes.Buffer
	d.WriteTo(&buf)
	if buf.String() != "<< >>" {
		t.Errorf("empty dict = %q, want %q", buf.String(), "<< >>")
	}
}

func TestDictionaryInsertionOrder(t *testing.T) {
	d := NewDictionary()
	d.Set("Z", PdfInteger(1))
	d.Set("A", PdfInteger(2))
	d.Set("M", PdfInteger(3))
	var buf bytes.Buffer
	d.WriteTo(&buf)
	s := buf.String()
	zi := strings.Index(s, "/Z")
	ai := strings.Index(s, "/A")
	mi := strings.Index(s, "/M")
	if zi >= ai || ai >= mi {
		t.Errorf("insertion order not preserved in output: %q", s)
	}
}
