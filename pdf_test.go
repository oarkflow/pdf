package pdf

import (
	"bytes"
	"encoding/json"
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

	markdown, err := ToMarkdown(path)
	if err != nil {
		t.Fatalf("ToMarkdown() error = %v", err)
	}
	if !strings.Contains(markdown, "Hello PDF") {
		t.Fatal("expected converted text in Markdown")
	}

	jsonOut, err := ToJSON(path)
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	var decoded struct {
		Pages []struct {
			Text string `json:"text"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(jsonOut, &decoded); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}
	if len(decoded.Pages) != 1 || !strings.Contains(decoded.Pages[0].Text, "Hello PDF") {
		t.Fatalf("decoded JSON = %#v, want extracted text", decoded)
	}
}

func TestFromMarkdownRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "markdown.pdf")
	err := FromMarkdown("# Quarterly report\n\nA **clear** summary.\n\n| Metric | Value |\n| --- | --- |\n| Revenue | 42 |", path)
	if err != nil {
		t.Fatalf("FromMarkdown() error = %v", err)
	}
	text, err := ToText(path)
	if err != nil {
		t.Fatalf("ToText() error = %v", err)
	}
	for _, want := range []string{"Quarterly report", "clear", "Revenue", "42"} {
		if !strings.Contains(text, want) {
			t.Errorf("PDF text %q does not contain %q", text, want)
		}
	}
}

func TestFromMarkdownListMarkerUsesWinAnsi(t *testing.T) {
	path := filepath.Join(t.TempDir(), "list.pdf")
	markdown := "Current CARE 2.0 status:\n\n- Core development is completed.\n- Acceptance testing is pending."
	if err := FromMarkdown(markdown, path); err != nil {
		t.Fatalf("FromMarkdown() error = %v", err)
	}
	text, err := ToText(path)
	if err != nil {
		t.Fatalf("ToText() error = %v", err)
	}
	if strings.Contains(text, "â€¢") {
		t.Fatalf("list marker was UTF-8 mojibake: %q", text)
	}
	if !strings.Contains(text, "•") {
		t.Fatalf("list marker missing from extracted text: %q", text)
	}
}

func TestInfoSplitAndExtractImages(t *testing.T) {
	path := writeSimpleReadablePDF(t)

	info, err := Info(path)
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	if info.Pages != 1 {
		t.Fatalf("pages = %d, want 1", info.Pages)
	}
	if len(info.PageSizes) != 1 || info.PageSizes[0].Width == 0 {
		t.Fatalf("page sizes = %#v, want one non-empty size", info.PageSizes)
	}

	out := filepath.Join(t.TempDir(), "split.pdf")
	if err := Split(path, out, []int{0}); err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	splitInfo, err := Info(out)
	if err != nil {
		t.Fatalf("Info(split) error = %v", err)
	}
	if splitInfo.Pages != 1 {
		t.Fatalf("split pages = %d, want 1", splitInfo.Pages)
	}

	images, err := ExtractImages(path)
	if err != nil {
		t.Fatalf("ExtractImages() error = %v", err)
	}
	if len(images) != 0 {
		t.Fatalf("images = %d, want 0", len(images))
	}
}

func TestPageOperationsAndProtection(t *testing.T) {
	path := writeTwoPageReadablePDF(t)
	dir := t.TempDir()

	rotated := filepath.Join(dir, "rotated.pdf")
	if err := Rotate(path, rotated, []int{0}, 90); err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	rotatedInfo, err := Info(rotated)
	if err != nil {
		t.Fatalf("Info(rotated) error = %v", err)
	}
	if rotatedInfo.PageSizes[0].Rotation != 90 {
		t.Fatalf("rotation = %d, want 90", rotatedInfo.PageSizes[0].Rotation)
	}

	deleted := filepath.Join(dir, "deleted.pdf")
	if err := DeletePages(path, deleted, []int{0}); err != nil {
		t.Fatalf("DeletePages() error = %v", err)
	}
	text, err := ToText(deleted)
	if err != nil {
		t.Fatalf("ToText(deleted) error = %v", err)
	}
	if strings.Contains(text, "First page") || !strings.Contains(text, "Second page") {
		t.Fatalf("deleted text = %q, want only second page", text)
	}

	reordered := filepath.Join(dir, "reordered.pdf")
	if err := Reorder(path, reordered, []int{1, 0}); err != nil {
		t.Fatalf("Reorder() error = %v", err)
	}
	reorderedText, err := ToText(reordered)
	if err != nil {
		t.Fatalf("ToText(reordered) error = %v", err)
	}
	if strings.Index(reorderedText, "Second page") > strings.Index(reorderedText, "First page") {
		t.Fatalf("reordered text = %q, want second page before first page", reorderedText)
	}

	numbered := filepath.Join(dir, "numbered.pdf")
	if err := AddPageNumbers(path, numbered, PageNumberOptions{}); err != nil {
		t.Fatalf("AddPageNumbers() error = %v", err)
	}
	numberedText, err := ToText(numbered)
	if err != nil {
		t.Fatalf("ToText(numbered) error = %v", err)
	}
	if !strings.Contains(numberedText, "Page 1 of 2") {
		t.Fatalf("numbered text = %q, want page number", numberedText)
	}

	watermarked := filepath.Join(dir, "watermarked.pdf")
	if err := Watermark(path, watermarked, WatermarkOptions{Text: "DRAFT"}); err != nil {
		t.Fatalf("Watermark() error = %v", err)
	}
	watermarkedText, err := ToText(watermarked)
	if err != nil {
		t.Fatalf("ToText(watermarked) error = %v", err)
	}
	if !strings.Contains(watermarkedText, "DRAFT") {
		t.Fatalf("watermarked text = %q, want watermark text", watermarkedText)
	}

	protected := filepath.Join(dir, "protected.pdf")
	if err := Protect(path, protected, core.EncryptionConfig{
		Algorithm:     core.AES_128,
		UserPassword:  "user-secret",
		OwnerPassword: "owner-secret",
		Permissions:   0xFFFFF0C4,
	}); err != nil {
		t.Fatalf("Protect() error = %v", err)
	}
	protectedInfo, err := Info(protected, "user-secret")
	if err != nil {
		t.Fatalf("Info(protected) error = %v", err)
	}
	if !protectedInfo.Encrypted {
		t.Fatal("protected PDF should be encrypted")
	}

	decrypted := filepath.Join(dir, "decrypted.pdf")
	if err := Decrypt(protected, decrypted, converter.ConvertOptions{Password: "user-secret"}); err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	decryptedInfo, err := Info(decrypted)
	if err != nil {
		t.Fatalf("Info(decrypted) error = %v", err)
	}
	if decryptedInfo.Encrypted {
		t.Fatal("decrypted PDF should not be encrypted")
	}

	meta := filepath.Join(dir, "metadata.pdf")
	if err := SetMetadata(path, meta, map[string]string{
		"Title":  "Updated Title",
		"Author": "Codex",
	}); err != nil {
		t.Fatalf("SetMetadata() error = %v", err)
	}
	metaInfo, err := Info(meta)
	if err != nil {
		t.Fatalf("Info(metadata) error = %v", err)
	}
	if metaInfo.Metadata["Title"] != "Updated Title" || metaInfo.Metadata["Author"] != "Codex" {
		t.Fatalf("metadata = %#v, want updated title/author", metaInfo.Metadata)
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

func writeTwoPageReadablePDF(t *testing.T) string {
	t.Helper()
	doc, err := document.NewDocument(document.A4)
	if err != nil {
		t.Fatal(err)
	}
	p1 := doc.NewPage()
	p1.Contents = []byte("BT /F1 12 Tf 72 760 Td (First page) Tj ET")
	p2 := doc.NewPage()
	p2.Contents = []byte("BT /F1 12 Tf 72 760 Td (Second page) Tj ET")

	path := filepath.Join(t.TempDir(), "two-page.pdf")
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}
