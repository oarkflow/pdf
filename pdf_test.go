package pdf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/core"
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

func TestFromHTMLStreamingEmbedsTailwindShadowAsImage(t *testing.T) {
	var buf bytes.Buffer
	err := FromHTMLStreaming(`<html><body><div style="width:200px;height:80px;background:#fff;box-shadow:0 10px 15px -3px rgb(0 0 0 / 0.1)">Card</div></body></html>`, &buf)
	if err != nil {
		t.Fatalf("FromHTMLStreaming() error = %v", err)
	}

	pdfData := buf.String()
	if !strings.Contains(pdfData, "/Subtype /Image") {
		t.Fatal("expected shadow image XObject to be embedded")
	}
	if !strings.Contains(pdfData, "/SMask") {
		t.Fatal("expected shadow image soft mask for alpha")
	}
}
