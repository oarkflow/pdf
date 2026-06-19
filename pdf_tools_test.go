package pdf

import (
	"bytes"
	"encoding/json"
	"fmt"
	stdimage "image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/reader"
)

func TestFromHTMLTemplateJSONFile(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.html")
	jsonPath := filepath.Join(dir, "data.json")
	out := filepath.Join(dir, "filled.pdf")
	if err := os.WriteFile(templatePath, []byte(`<html><body><p>Hello {{ user.name }}</p></body></html>`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, []byte(`{"user":{"name":"Ada"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := FromHTMLTemplateJSONFile(templatePath, jsonPath, out); err != nil {
		t.Fatalf("FromHTMLTemplateJSONFile failed: %v", err)
	}
	text, err := ToText(out)
	if err != nil {
		t.Fatalf("ToText failed: %v", err)
	}
	if !strings.Contains(text, "Hello Ada") {
		t.Fatalf("filled PDF text = %q, want placeholder value", text)
	}
}

func TestLoadTemplateJSONRequiresObject(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "data.json")
	if err := os.WriteFile(jsonPath, []byte(`["not", "object"]`), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadTemplateJSON(jsonPath); err == nil {
		t.Fatal("expected JSON object error")
	}
}

func TestToolCatalogIncludesTemplateAndSignatureTools(t *testing.T) {
	catalog := ToolCatalog()
	foundTemplate := false
	foundSignature := false
	foundFill := false
	for _, tool := range catalog {
		if tool.Key == "template-json" && tool.Status == ToolAvailable {
			foundTemplate = true
		}
		if tool.Key == "digital-signature" && tool.Status == ToolAvailable {
			foundSignature = true
		}
		if tool.Key == "acroform-fill" && tool.Status == ToolAvailable {
			foundFill = true
		}
	}
	if !foundTemplate || !foundSignature || !foundFill {
		t.Fatalf("catalog missing expected tools: %#v", catalog)
	}
}

func TestFillFormJSONFileAndListFormFields(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "form.pdf")
	dataPath := filepath.Join(dir, "data.json")
	out := filepath.Join(dir, "filled.pdf")
	writeMinimalFormPDF(t, input)
	if err := os.WriteFile(dataPath, []byte(`{"full_name":"Ada Lovelace"}`), 0644); err != nil {
		t.Fatal(err)
	}

	fields, err := ListFormFields(input, "")
	if err != nil {
		t.Fatalf("ListFormFields failed: %v", err)
	}
	if len(fields) != 1 || fields[0].Name != "full_name" || fields[0].Field != "Tx" {
		t.Fatalf("fields = %#v, want full_name text field", fields)
	}
	if err := FillFormJSONFile(input, dataPath, out); err != nil {
		t.Fatalf("FillFormJSONFile failed: %v", err)
	}
	text, err := ToText(out)
	if err != nil {
		t.Fatalf("ToText failed: %v", err)
	}
	if !strings.Contains(text, "Ada Lovelace") {
		t.Fatalf("filled text = %q, want form value", text)
	}
}

func TestExtraPDFTools(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.pdf")
	compressed := filepath.Join(dir, "compressed.pdf")
	redacted := filepath.Join(dir, "redacted.pdf")
	translated := filepath.Join(dir, "translated.pdf")
	stamped := filepath.Join(dir, "stamped.pdf")
	img := filepath.Join(dir, "stamp.png")
	dict := filepath.Join(dir, "dict.json")
	writeTextPDF(t, input, "Secret Hello")
	writeTestPNG(t, img)
	if err := os.WriteFile(dict, []byte(`{"Hello":"Namaste"}`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CompressPDF(input, compressed, ""); err != nil {
		t.Fatalf("CompressPDF failed: %v", err)
	}
	assertPDFPages(t, compressed, 1)
	if err := Redact(input, redacted, RedactOptions{Texts: []string{"Secret"}}); err != nil {
		t.Fatalf("Redact failed: %v", err)
	}
	redactedText, err := ToText(redacted)
	if err != nil {
		t.Fatalf("ToText redacted failed: %v", err)
	}
	if strings.Contains(redactedText, "Secret") {
		t.Fatalf("redacted text = %q, still contains secret", redactedText)
	}
	if err := TranslateWithDictionary(input, dict, translated, ""); err != nil {
		t.Fatalf("TranslateWithDictionary failed: %v", err)
	}
	translatedText, err := ToText(translated)
	if err != nil {
		t.Fatalf("ToText translated failed: %v", err)
	}
	if !strings.Contains(translatedText, "Namaste") {
		t.Fatalf("translated text = %q, want Namaste", translatedText)
	}
	if err := StampImage(input, stamped, ImageStampOptions{ImagePath: img, X: 72, Y: 72, Width: 24}); err != nil {
		t.Fatalf("StampImage failed: %v", err)
	}
	assertPDFPages(t, stamped, 1)
	cmp, err := ComparePDF(input, compressed)
	if err != nil {
		t.Fatalf("ComparePDF failed: %v", err)
	}
	if !cmp.PageCountMatches {
		t.Fatalf("compare = %#v, want matching page count", cmp)
	}
	graph, err := BuildPDFGraph([]string{input, stamped}, "")
	if err != nil {
		t.Fatalf("BuildPDFGraph failed: %v", err)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("graph nodes = %#v, want 2", graph.Nodes)
	}
}

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

func writeMinimalFormPDF(t *testing.T, path string) {
	t.Helper()
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R /AcroForm << /Fields [5 0 R] >> >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> >> >> /Contents 4 0 R /Annots [5 0 R] >>",
		"<< /Length 0 >>\nstream\n\nendstream",
		"<< /Type /Annot /Subtype /Widget /FT /Tx /T (full_name) /Rect [72 700 220 725] /P 3 0 R >>",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
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
