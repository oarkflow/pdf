package reader

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/document"
)

// buildMinimalPDF creates a minimal valid PDF with one page and correct xref offsets.
func buildMinimalPDF(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	off1 := buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	off2 := buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := buf.Len()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n")

	xrefOff := buf.Len()
	buf.WriteString("xref\n0 4\n")
	fmt.Fprintf(&buf, "0000000000 65535 f \r\n")
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off1)
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off2)
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off3)
	fmt.Fprintf(&buf, "trailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOff)
	return buf.Bytes()
}

func TestOpenMinimalPDF(t *testing.T) {
	data := buildMinimalPDF(t)
	r, err := Open(data)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if r.NumPages() != 1 {
		t.Errorf("NumPages = %d, want 1", r.NumPages())
	}
}

func TestOpenInvalidData(t *testing.T) {
	_, err := Open([]byte("not a pdf"))
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

func TestOpenEmptyData(t *testing.T) {
	_, err := Open([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestPageInfo(t *testing.T) {
	data := buildMinimalPDF(t)
	r, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}
	page, err := r.Page(0)
	if err != nil {
		t.Fatal(err)
	}
	if page.MediaBox != [4]float64{0, 0, 612, 792} {
		t.Errorf("MediaBox = %v, want [0 0 612 792]", page.MediaBox)
	}
}

func TestPageOutOfRange(t *testing.T) {
	data := buildMinimalPDF(t)
	r, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Page(-1)
	if err == nil {
		t.Error("expected error for negative page")
	}
	_, err = r.Page(1)
	if err == nil {
		t.Error("expected error for out of range page")
	}
}

func TestTrailerAndCatalog(t *testing.T) {
	data := buildMinimalPDF(t)
	r, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}
	tr := r.Trailer()
	if tr == nil {
		t.Fatal("trailer is nil")
	}
	cat := r.Catalog()
	if cat == nil {
		t.Fatal("catalog is nil")
	}
	typ, _ := cat["/Type"].(string)
	if typ != "/Catalog" {
		t.Errorf("catalog type = %q, want /Catalog", typ)
	}
}

func TestMetadataEmpty(t *testing.T) {
	data := buildMinimalPDF(t)
	r, _ := Open(data)
	meta := r.Metadata()
	if len(meta) != 0 {
		t.Errorf("expected empty metadata, got %v", meta)
	}
}

func TestGetResolver(t *testing.T) {
	data := buildMinimalPDF(t)
	r, _ := Open(data)
	if r.GetResolver() == nil {
		t.Error("resolver should not be nil")
	}
}

func TestMetadataWithInfo(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	off1 := buf.Len()
	buf.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	off2 := buf.Len()
	buf.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")
	off3 := buf.Len()
	buf.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\n")
	off4 := buf.Len()
	buf.WriteString("4 0 obj\n<< /Title (Test Title) /Author (Test Author) >>\nendobj\n")
	xrefOff := buf.Len()
	buf.WriteString("xref\n0 5\n")
	fmt.Fprintf(&buf, "0000000000 65535 f \r\n")
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off1)
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off2)
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off3)
	fmt.Fprintf(&buf, "%010d 00000 n \r\n", off4)
	fmt.Fprintf(&buf, "trailer\n<< /Size 5 /Root 1 0 R /Info 4 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOff)

	r, err := Open(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	meta := r.Metadata()
	if meta["Title"] != "Test Title" {
		t.Errorf("Title = %q, want %q", meta["Title"], "Test Title")
	}
	if meta["Author"] != "Test Author" {
		t.Errorf("Author = %q, want %q", meta["Author"], "Test Author")
	}
}

func TestOpenWithPassword_AES128(t *testing.T) {
	doc, err := document.NewDocument(document.A4)
	if err != nil {
		t.Fatal(err)
	}
	doc.SetEncryption(core.EncryptionConfig{
		Algorithm:     core.AES_128,
		OwnerPassword: "owner-secret",
		UserPassword:  "user-secret",
		Permissions:   0xFFFFF0C4,
	})
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf 100 700 Td (Hello Secret) Tj ET")

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	r, err := OpenWithPassword(buf.Bytes(), "user-secret")
	if err != nil {
		t.Fatalf("OpenWithPassword: %v", err)
	}
	text, err := r.ExtractText(0)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	if text != "Hello Secret" {
		t.Fatalf("text = %q, want %q", text, "Hello Secret")
	}
}

func TestOpenWithPassword_WrongPassword(t *testing.T) {
	doc, err := document.NewDocument(document.A4)
	if err != nil {
		t.Fatal(err)
	}
	doc.SetEncryption(core.EncryptionConfig{
		Algorithm:     core.AES_128,
		OwnerPassword: "owner-secret",
		UserPassword:  "user-secret",
		Permissions:   0xFFFFF0C4,
	})
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf 100 700 Td (Nope) Tj ET")

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	if _, err := OpenWithPassword(buf.Bytes(), "wrong-password"); err == nil {
		t.Fatal("expected password error")
	}
}
