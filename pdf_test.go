package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/converter"
	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/html"
)

func TestFromHTMLStreamingAppliesEncryption(t *testing.T) {
	var buf bytes.Buffer
	err := FromHTMLStreaming(`<html><body><p>Hello</p></body></html>`, &buf, html.Options{
		Encryption: &core.EncryptionConfig{
			Algorithm:     core.AES_128,
			OwnerPassword: "owner-secret",
			UserPassword:  "user-secret",
			Permissions:   0xFFFFF0C4,
		},
	})
	if err != nil {
		t.Fatalf("FromHTMLStreaming() error = %v", err)
	}

	pdfData := buf.String()
	if !strings.Contains(pdfData, "/Encrypt") {
		t.Fatal("expected encrypted PDF trailer")
	}
	if !strings.Contains(pdfData, "/StmF /StdCF") {
		t.Fatal("expected AES-128 encryption dictionary")
	}
}

func TestFromHTMLStreamingSkipsUnsupportedTailwindShadow(t *testing.T) {
	var buf bytes.Buffer
	err := FromHTMLStreaming(`<html><body><div style="width:200px;height:80px;background:#fff;box-shadow:0 10px 15px -3px rgb(0 0 0 / 0.1)">Card</div></body></html>`, &buf)
	if err != nil {
		t.Fatalf("FromHTMLStreaming() error = %v", err)
	}

	pdfData := buf.String()
	if strings.Contains(pdfData, "/ExtGState") {
		t.Fatal("expected unsupported blurred shadow to be omitted")
	}
}

func TestToTextAndToHTML(t *testing.T) {
	path := writeSimpleReadablePDF(t)

	text, err := ToText(path)
	if err != nil {
		t.Fatalf("ToText() error = %v", err)
	}
	if !strings.Contains(text, "Hello PDF") || !strings.Contains(text, "Second line") {
		t.Fatalf("text = %q, want extracted lines", text)
	}

	htmlOut, err := ToHTML(path, converter.ConvertOptions{Mode: "positioned"})
	if err != nil {
		t.Fatalf("ToHTML() error = %v", err)
	}
	if !strings.Contains(htmlOut, "<!DOCTYPE html>") {
		t.Fatal("expected HTML doctype")
	}
	if !strings.Contains(htmlOut, "Hello PDF") {
		t.Fatal("expected converted text in HTML")
	}
}

func writeSimpleReadablePDF(t *testing.T) string {
	t.Helper()
	doc, err := document.NewDocument(document.A4)
	if err != nil {
		t.Fatal(err)
	}
	p := doc.NewPage()
	p.Contents = []byte("BT /F1 12 Tf 72 760 Td (Hello PDF) Tj 0 -18 Td (Second line) Tj ET")

	path := filepath.Join(t.TempDir(), "sample.pdf")
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}
