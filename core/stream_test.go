package core

import (
	"bytes"
	"compress/zlib"
	"io"
	"strings"
	"testing"
)

func TestNewStream(t *testing.T) {
	data := []byte("BT /F1 12 Tf (Hello) Tj ET")
	s := NewStream(data)
	if s.Type() != ObjStream {
		t.Fatal("expected ObjStream")
	}
	if !bytes.Equal(s.Data, data) {
		t.Error("stream data mismatch")
	}
	if s.Dictionary == nil {
		t.Fatal("dictionary should not be nil")
	}
}

func TestStreamCompress(t *testing.T) {
	data := []byte("Hello World Hello World Hello World")
	s := NewStream(data)
	if err := s.Compress(); err != nil {
		t.Fatal(err)
	}
	// Data should be different (compressed)
	if bytes.Equal(s.Data, data) {
		t.Error("data unchanged after compression")
	}
	// Should have FlateDecode filter
	f := s.Dictionary.Get("Filter")
	if f == nil || f.(PdfName) != "FlateDecode" {
		t.Errorf("Filter = %v, want FlateDecode", f)
	}
	// Verify we can decompress
	r, err := zlib.NewReader(bytes.NewReader(s.Data))
	if err != nil {
		t.Fatal(err)
	}
	decompressed, _ := io.ReadAll(r)
	r.Close()
	if !bytes.Equal(decompressed, data) {
		t.Error("decompressed data mismatch")
	}
}

func TestStreamCompressIdempotent(t *testing.T) {
	s := NewStream([]byte("test"))
	s.Compress()
	data1 := make([]byte, len(s.Data))
	copy(data1, s.Data)
	s.Compress() // should be no-op
	if !bytes.Equal(s.Data, data1) {
		t.Error("double compress changed data")
	}
}

func TestStreamCompressEmpty(t *testing.T) {
	s := NewStream([]byte{})
	if err := s.Compress(); err != nil {
		t.Fatal(err)
	}
}

func TestStreamWriteTo(t *testing.T) {
	data := []byte("hello")
	s := NewStream(data)
	var buf bytes.Buffer
	_, err := s.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "stream") {
		t.Error("output missing 'stream' keyword")
	}
	if !strings.Contains(out, "endstream") {
		t.Error("output missing 'endstream' keyword")
	}
	if !strings.Contains(out, "hello") {
		t.Error("output missing data")
	}
	// Length should be set
	if !strings.Contains(out, "/Length") {
		t.Error("output missing /Length")
	}
}

func TestStreamWriteToSetsLength(t *testing.T) {
	data := []byte("1234567890")
	s := NewStream(data)
	var buf bytes.Buffer
	s.WriteTo(&buf)
	l := s.Dictionary.Get("Length")
	if l == nil {
		t.Fatal("Length not set")
	}
	if int(l.(PdfInteger)) != 10 {
		t.Errorf("Length = %d, want 10", l.(PdfInteger))
	}
}

func TestStreamCompressedWriteTo(t *testing.T) {
	data := []byte("BT /F1 12 Tf (Hello World) Tj ET")
	s := NewStream(data)
	s.Compress()
	var buf bytes.Buffer
	s.WriteTo(&buf)
	out := buf.String()
	if !strings.Contains(out, "/FlateDecode") {
		t.Error("compressed output missing /FlateDecode")
	}
}
