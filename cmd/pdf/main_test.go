package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	pdfapi "github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/reader"
)

func TestCmdMergePositional(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.pdf")
	b := filepath.Join(dir, "b.pdf")
	out := filepath.Join(dir, "out.pdf")
	writeCmdTestPDF(t, a, 1)
	writeCmdTestPDF(t, b, 1)

	runCmd(t, "pdf", "merge", out, a, b)
	assertCmdPDFPages(t, out, 2)
}

func TestCmdMergeConfig(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.pdf")
	b := filepath.Join(dir, "b.pdf")
	writeCmdTestPDF(t, a, 1)
	writeCmdTestPDF(t, b, 1)
	cfg := pdfapi.MergeOptions{
		Output: "out.pdf",
		Inputs: []pdfapi.MergeInput{
			{Path: "a.pdf"},
			{Path: "b.pdf"},
		},
	}
	cfgPath := filepath.Join(dir, "merge.json")
	writeCmdJSON(t, cfgPath, cfg)

	runCmd(t, "pdf", "merge", "-config", cfgPath)
	assertCmdPDFPages(t, filepath.Join(dir, "out.pdf"), 2)
}

func TestCmdSplitExtract(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.pdf")
	out := filepath.Join(dir, "out.pdf")
	writeCmdTestPDF(t, input, 2)

	runCmd(t, "pdf", "split", "-o", out, "-page", "1", input)
	assertCmdPDFPages(t, out, 1)
}

func TestCmdSplitConfig(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.pdf")
	writeCmdTestPDF(t, input, 3)
	cfg := pdfapi.SplitConfig{
		Input: "input.pdf",
		Outputs: []pdfapi.SplitOutput{
			{Output: "one.pdf", Pages: "1"},
			{Output: "rest.pdf", Pages: "2-3"},
		},
	}
	cfgPath := filepath.Join(dir, "split.json")
	writeCmdJSON(t, cfgPath, cfg)

	runCmd(t, "pdf", "split", "-config", cfgPath)
	assertCmdPDFPages(t, filepath.Join(dir, "one.pdf"), 1)
	assertCmdPDFPages(t, filepath.Join(dir, "rest.pdf"), 2)
}

func TestCmdPasswordAddAndRemove(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.pdf")
	protected := filepath.Join(dir, "protected.pdf")
	unlocked := filepath.Join(dir, "unlocked.pdf")
	writeCmdTestPDF(t, input, 1)

	withCmdStdin(t, "secret\n", func() {
		runCmd(t, "pdf", "password", "add", "-o", protected, input)
	})
	withCmdStdin(t, "secret\n", func() {
		runCmd(t, "pdf", "password", "remove", "-o", unlocked, protected)
	})
	assertCmdPDFPages(t, unlocked, 1)
}

func TestCmdImagesToPDFSearchAndValidate(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "image.png")
	imagesPDF := filepath.Join(dir, "images.pdf")
	textPDF := filepath.Join(dir, "text.pdf")
	writeCmdTestPNG(t, img)
	writeCmdTextPDF(t, textPDF, "ProcessGate search target")

	runCmd(t, "pdf", "images-to-pdf", "-o", imagesPDF, img)
	assertCmdPDFPages(t, imagesPDF, 1)
	runCmd(t, "pdf", "search", "-q", "target", textPDF)
	runCmd(t, "pdf", "validate", textPDF)
}

func TestCmdHelpTopics(t *testing.T) {
	runCmd(t, "pdf", "help")
	runCmd(t, "pdf", "merge", "--help")
	runCmd(t, "pdf", "password", "--help")
	runCmd(t, "pdf", "password", "add", "--help")
}

func runCmd(t *testing.T, args ...string) {
	t.Helper()
	if err := runCLI(args); err != nil {
		t.Fatalf("%v: %v", args, err)
	}
}

func writeCmdTestPDF(t *testing.T, path string, pages int) {
	t.Helper()
	w := document.NewWriter()
	for i := 0; i < pages; i++ {
		if _, err := w.AddPage(document.NewPage(document.A4)); err != nil {
			t.Fatalf("adding page: %v", err)
		}
	}
	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatalf("writing PDF: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func writeCmdTextPDF(t *testing.T, path, text string) {
	t.Helper()
	w := document.NewWriter()
	page := document.NewPage(document.A4)
	page.Fonts["F1"] = 0
	page.Contents = []byte("BT /F1 12 Tf 72 720 Td (" + text + ") Tj ET")
	if _, err := w.AddPage(page); err != nil {
		t.Fatalf("adding page: %v", err)
	}
	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		t.Fatalf("writing PDF: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func writeCmdTestPNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: 100, B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encoding PNG: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func writeCmdJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshaling JSON: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func assertCmdPDFPages(t *testing.T, path string, pages int) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	r, err := reader.Open(data)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	if r.NumPages() != pages {
		t.Fatalf("%s pages = %d, want %d", path, r.NumPages(), pages)
	}
}

func withCmdStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdin pipe: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("writing stdin pipe: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing stdin pipe: %v", err)
	}
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()
	fn()
}
