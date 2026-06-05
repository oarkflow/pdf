package pdf

import (
	"bytes"
	"encoding/json"
	stdimage "image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/reader"
)

func TestMergeWithOptionsMixedPDFAndImages(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "doc.pdf")
	img1 := filepath.Join(dir, "a.png")
	img2 := filepath.Join(dir, "b.png")
	out := filepath.Join(dir, "merged.pdf")
	writeTestPDF(t, pdfPath, 1)
	writeTestPNG(t, img1)
	writeTestPNG(t, img2)

	err := MergeWithOptions(MergeOptions{
		Output: out,
		Inputs: []MergeInput{
			{Path: img1},
			{Path: pdfPath},
			{Path: img2},
		},
	})
	if err != nil {
		t.Fatalf("MergeWithOptions failed: %v", err)
	}
	assertPDFPages(t, out, 3)
}

func TestMergeWithOptionsDirectoryAndConfig(t *testing.T) {
	dir := t.TempDir()
	inputDir := filepath.Join(dir, "inputs")
	if err := os.Mkdir(inputDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestPDF(t, filepath.Join(inputDir, "b.pdf"), 1)
	writeTestPNG(t, filepath.Join(inputDir, "a.png"))
	if err := os.WriteFile(filepath.Join(inputDir, "skip.txt"), []byte("skip"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := MergeOptions{
		Output: "out.pdf",
		Page:   MergePageOptions{Size: "A4", Margin: 36, ImageFit: "contain"},
		Inputs: []MergeInput{
			{Directory: "inputs", Include: []string{"*.pdf", "*.png"}, Sort: "name"},
		},
	}
	cfgPath := filepath.Join(dir, "merge.json")
	writeJSONFile(t, cfgPath, cfg)

	if err := MergeWithConfig(cfgPath); err != nil {
		t.Fatalf("MergeWithConfig failed: %v", err)
	}
	assertPDFPages(t, filepath.Join(dir, "out.pdf"), 2)
}

func TestMergeWithOptionsDirectorySequence(t *testing.T) {
	dir := t.TempDir()
	writeTestPDF(t, filepath.Join(dir, "one.pdf"), 1)
	writeTestPNG(t, filepath.Join(dir, "two.png"))
	out := filepath.Join(dir, "sequence.pdf")

	err := MergeWithOptions(MergeOptions{
		Output: out,
		Inputs: []MergeInput{
			{Directory: dir, Sequence: []string{"two.png", "one.pdf"}},
		},
	})
	if err != nil {
		t.Fatalf("MergeWithOptions sequence failed: %v", err)
	}
	assertPDFPages(t, out, 2)
}

func TestMergeWithOptionsUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(path, []byte("nope"), 0644); err != nil {
		t.Fatal(err)
	}
	err := MergeWithOptions(MergeOptions{
		Output: filepath.Join(dir, "out.pdf"),
		Inputs: []MergeInput{{Path: path}},
	})
	if err == nil {
		t.Fatal("expected unsupported extension error")
	}
}

func TestSplitWithConfigMultipleOutputs(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.pdf")
	writeTestPDF(t, input, 3)
	cfg := SplitConfig{
		Input: "source.pdf",
		Outputs: []SplitOutput{
			{Output: "page-1.pdf", Pages: "1"},
			{Output: "pages-2-3.pdf", Pages: "2-3"},
		},
	}
	cfgPath := filepath.Join(dir, "split.json")
	writeJSONFile(t, cfgPath, cfg)

	if err := SplitWithConfig(cfgPath); err != nil {
		t.Fatalf("SplitWithConfig failed: %v", err)
	}
	assertPDFPages(t, filepath.Join(dir, "page-1.pdf"), 1)
	assertPDFPages(t, filepath.Join(dir, "pages-2-3.pdf"), 2)
}

func TestSplitWithConfigInvalidPages(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.pdf")
	writeTestPDF(t, input, 1)
	err := SplitWithOptions(SplitConfig{
		Input: input,
		Outputs: []SplitOutput{
			{Output: filepath.Join(dir, "out.pdf"), Pages: "3-1"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid page range error")
	}
}

func TestImagesToPDF(t *testing.T) {
	dir := t.TempDir()
	img1 := filepath.Join(dir, "one.png")
	img2 := filepath.Join(dir, "two.png")
	out := filepath.Join(dir, "images.pdf")
	writeTestPNG(t, img1)
	writeTestPNG(t, img2)

	if err := ImagesToPDF(ImagePDFOptions{
		Output: out,
		Inputs: []MergeInput{{Path: img1}, {Path: img2}},
	}); err != nil {
		t.Fatalf("ImagesToPDF failed: %v", err)
	}
	assertPDFPages(t, out, 2)
}

func TestSearchTextAndValidatePDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.pdf")
	writeTextPDF(t, path, "Find the needle")

	matches, err := SearchText(path, SearchOptions{Query: "needle"})
	if err != nil {
		t.Fatalf("SearchText failed: %v", err)
	}
	if len(matches) != 1 || matches[0].Page != 1 {
		t.Fatalf("unexpected matches: %#v", matches)
	}
	result := ValidatePDF(path, "")
	if !result.Valid || result.Pages != 1 {
		t.Fatalf("unexpected validation result: %#v", result)
	}
}

func writeTestPDF(t *testing.T, path string, pages int) {
	t.Helper()
	w := document.NewWriter()
	for i := 0; i < pages; i++ {
		page := document.NewPage(document.A4)
		if _, err := w.AddPage(page); err != nil {
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

func writeTextPDF(t *testing.T, path, text string) {
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

func writeTestPNG(t *testing.T, path string) {
	t.Helper()
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, 4, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 180, G: 40, B: 80, A: 255})
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

func writeJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshaling config: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func assertPDFPages(t *testing.T, path string, pages int) {
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
