package document

import (
	"bytes"
	"testing"

	"github.com/oarkflow/pdf/core"
)

func TestNewDocument(t *testing.T) {
	doc, _ := NewDocument(A4)
	if doc.PageSize() != A4 {
		t.Errorf("page size = %v, want A4", doc.PageSize())
	}
	m := doc.Margins()
	if m.Top != 72 || m.Right != 72 || m.Bottom != 72 || m.Left != 72 {
		t.Errorf("margins = %v, want default 72pt", m)
	}
	if len(doc.Pages()) != 0 {
		t.Error("new doc should have 0 pages")
	}
}

func TestNewDocumentPageSizes(t *testing.T) {
	tests := []struct {
		name string
		size PageSize
	}{
		{"A4", A4},
		{"A3", A3},
		{"A5", A5},
		{"Letter", Letter},
		{"Legal", Legal},
	}
	for _, tt := range tests {
		doc, _ := NewDocument(tt.size)
		if doc.PageSize() != tt.size {
			t.Errorf("%s: page size mismatch", tt.name)
		}
	}
}

func TestAddPage(t *testing.T) {
	doc, _ := NewDocument(A4)
	p := NewPage(A4)
	doc.AddPage(p)
	if len(doc.Pages()) != 1 {
		t.Errorf("pages = %d, want 1", len(doc.Pages()))
	}
}

func TestNewPage(t *testing.T) {
	doc, _ := NewDocument(Letter)
	p := doc.NewPage()
	if len(doc.Pages()) != 1 {
		t.Error("NewPage should add page")
	}
	if p.Size != Letter {
		t.Error("page size mismatch")
	}
}

func TestSetMargins(t *testing.T) {
	doc, _ := NewDocument(A4)
	m := Margins{10, 20, 30, 40}
	doc.SetMargins(m)
	if doc.Margins() != m {
		t.Error("margins not set")
	}
}

func TestSetMetadata(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetMetadata(Metadata{
		Title:  "Test",
		Author: "Tester",
	})
	doc.NewPage()
	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Test")) {
		t.Error("metadata title not in output")
	}
}

func TestSetEncryption(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetEncryption(core.EncryptionConfig{
		Algorithm:     core.AES_256,
		OwnerPassword: "owner",
	})
	// Should not panic; encryption is stub
	doc.NewPage()
	var buf bytes.Buffer
	doc.WriteTo(&buf)
}

func TestSetPDFA(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetPDFA(PDFA2b)
	doc.NewPage()
	var buf bytes.Buffer
	doc.WriteTo(&buf)
	// Just verify it doesn't panic
}

func TestMultiplePages(t *testing.T) {
	doc, _ := NewDocument(A4)
	for i := 0; i < 5; i++ {
		doc.NewPage()
	}
	if len(doc.Pages()) != 5 {
		t.Errorf("pages = %d, want 5", len(doc.Pages()))
	}
}

func TestSetHeader(t *testing.T) {
	doc, _ := NewDocument(A4)
	called := false
	doc.SetHeader(func(info PageInfo) []byte {
		called = true
		return []byte("BT /F1 10 Tf (Header) Tj ET\n")
	})
	doc.NewPage()
	var buf bytes.Buffer
	doc.WriteTo(&buf)
	if !called {
		t.Error("header decorator not called")
	}
}

func TestSetFooter(t *testing.T) {
	doc, _ := NewDocument(A4)
	doc.SetFooter(func(info PageInfo) []byte {
		return []byte("BT /F1 10 Tf (Footer) Tj ET\n")
	})
	doc.NewPage()
	var buf bytes.Buffer
	doc.WriteTo(&buf)
}

func TestDefaultMargins(t *testing.T) {
	m := DefaultMargins()
	if m.Top != 72 || m.Right != 72 || m.Bottom != 72 || m.Left != 72 {
		t.Errorf("DefaultMargins = %v", m)
	}
}
