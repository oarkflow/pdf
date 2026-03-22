package document

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/core"
)

func TestApplyEncryptionRC4(t *testing.T) {
	doc, _ := NewDocument(PageSize{Width: 612, Height: 792})
	doc.SetEncryption(core.EncryptionConfig{
		Algorithm:     core.RC4_128,
		OwnerPassword: "owner",
		UserPassword:  "user",
		Permissions:   0xFFFFF0C4,
	})
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf 100 700 Td (Hello) Tj ET")

	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	pdf := buf.String()
	if !strings.Contains(pdf, "/Encrypt") {
		t.Error("PDF trailer should contain /Encrypt reference")
	}
	if !strings.Contains(pdf, "/Filter /Standard") {
		t.Error("Encrypt dict should have /Filter /Standard")
	}
	if !strings.Contains(pdf, "/ID") {
		t.Error("PDF trailer should contain /ID array")
	}
}

func TestApplyEncryptionAES128(t *testing.T) {
	doc, _ := NewDocument(PageSize{Width: 612, Height: 792})
	doc.SetEncryption(core.EncryptionConfig{
		Algorithm:     core.AES_128,
		OwnerPassword: "secret",
		UserPassword:  "",
		Permissions:   0xFFFFF0C4,
	})
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf 100 700 Td (Test) Tj ET")

	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	pdf := buf.String()
	if !strings.Contains(pdf, "/Encrypt") {
		t.Error("PDF should contain /Encrypt")
	}
	if !strings.Contains(pdf, "/StmF /StdCF") {
		t.Error("AES-128 should have crypt filter /StmF /StdCF")
	}
	if !strings.Contains(pdf, "/P -3900") {
		t.Error("permissions should be written as signed PDF integer")
	}
}

func TestApplyEncryptionAES256Unsupported(t *testing.T) {
	doc, _ := NewDocument(PageSize{Width: 612, Height: 792})
	doc.SetEncryption(core.EncryptionConfig{
		Algorithm:     core.AES_256,
		OwnerPassword: "owner256",
		UserPassword:  "user256",
		Permissions:   0xFFFFF0C4,
	})
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf 100 700 Td (AES256) Tj ET")

	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err == nil {
		t.Fatal("expected AES-256 to be rejected")
	}
}

func TestNoEncryption(t *testing.T) {
	doc, _ := NewDocument(PageSize{Width: 612, Height: 792})
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf 100 700 Td (Plain) Tj ET")

	var buf bytes.Buffer
	_, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	pdf := buf.String()
	if strings.Contains(pdf, "/Encrypt") {
		t.Error("unencrypted PDF should not contain /Encrypt")
	}
}

func TestWriteStreamingToPreservesEncryption(t *testing.T) {
	doc, _ := NewDocument(PageSize{Width: 612, Height: 792})
	doc.SetEncryption(core.EncryptionConfig{
		Algorithm:     core.AES_128,
		OwnerPassword: "owner256",
		UserPassword:  "user256",
		Permissions:   0xFFFFF0C4,
	})
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf 100 700 Td (Encrypted) Tj ET")

	var buf bytes.Buffer
	if err := doc.WriteStreamingTo(&buf); err != nil {
		t.Fatalf("WriteStreamingTo failed: %v", err)
	}

	pdf := buf.String()
	if !strings.Contains(pdf, "/Encrypt") {
		t.Error("streaming encrypted PDF should contain /Encrypt")
	}
	if !strings.Contains(pdf, "/StmF /StdCF") {
		t.Error("streaming encrypted PDF should keep AES-128 dictionary")
	}
}
