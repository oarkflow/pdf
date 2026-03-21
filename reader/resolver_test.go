package reader

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"testing"
)

func buildPDFWithXRef(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	off1 := buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	off2 := buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := buf.Len()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] >>\nendobj\n")

	xrefOff := buf.Len()
	buf.WriteString("xref\n0 4\n")
	fmt.Fprintf(&buf, "0000000000 65535 f \r\n")
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off1)
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off2)
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off3)
	fmt.Fprintf(&buf, "trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOff)
	return buf.Bytes()
}

func TestNewResolver(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, err := NewResolver(data)
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal("resolver is nil")
	}
}

func TestNewResolverInvalid(t *testing.T) {
	_, err := NewResolver([]byte("garbage"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

func TestResolverTrailer(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, _ := NewResolver(data)
	tr, err := r.Trailer()
	if err != nil {
		t.Fatal(err)
	}
	if tr == nil {
		t.Fatal("trailer is nil")
	}
	size, ok := getInt(tr, "/Size")
	if !ok || size != 4 {
		t.Errorf("trailer size = %d, want 4", size)
	}
}

func TestResolveObject(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, _ := NewResolver(data)
	obj, err := r.ResolveObject(1)
	if err != nil {
		t.Fatal(err)
	}
	dict, ok := obj.(map[string]interface{})
	if !ok {
		t.Fatal("object 1 is not a dict")
	}
	if dict["/Type"] != "/Catalog" {
		t.Errorf("type = %v, want /Catalog", dict["/Type"])
	}
}

func TestResolveObjectNotInXref(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, _ := NewResolver(data)
	_, err := r.ResolveObject(999)
	if err == nil {
		t.Error("expected error for missing object")
	}
}

func TestResolveObjectCaching(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, _ := NewResolver(data)
	obj1, _ := r.ResolveObject(1)
	obj2, _ := r.ResolveObject(1)
	d1, _ := obj1.(map[string]interface{})
	d2, _ := obj2.(map[string]interface{})
	if d1["/Type"] != d2["/Type"] {
		t.Error("cached results should match")
	}
}

func TestResolveReference(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, _ := NewResolver(data)
	// Non-ref passes through
	val, err := r.ResolveReference("hello")
	if err != nil {
		t.Fatal(err)
	}
	if val != "hello" {
		t.Error("non-ref should pass through")
	}
	// IndirectRef resolves
	ref := IndirectRef{ObjNum: 1, GenNum: 0}
	val, err = r.ResolveReference(ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := val.(map[string]interface{}); !ok {
		t.Error("expected dict from resolved ref")
	}
}

func TestDecompressStreamNoFilter(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, _ := NewResolver(data)
	raw := []byte("hello")
	out, err := r.DecompressStream(nil, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, raw) {
		t.Error("nil dict should return raw data")
	}
}

func TestDecompressStreamFlate(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, _ := NewResolver(data)

	var cbuf bytes.Buffer
	w := zlib.NewWriter(&cbuf)
	w.Write([]byte("Hello PDF"))
	w.Close()

	dict := map[string]interface{}{
		"/Filter": "/FlateDecode",
	}
	out, err := r.DecompressStream(dict, cbuf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "Hello PDF" {
		t.Errorf("decompressed = %q, want %q", out, "Hello PDF")
	}
}

func TestDecompressStreamNoFilterKey(t *testing.T) {
	data := buildPDFWithXRef(t)
	r, _ := NewResolver(data)
	dict := map[string]interface{}{}
	raw := []byte("raw")
	out, _ := r.DecompressStream(dict, raw)
	if !bytes.Equal(out, raw) {
		t.Error("no filter should return raw")
	}
}
