package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/document"
)

func main() {
	dir := filepath.Join("examples", "page_numbers", "out")
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatalf("creating output directory: %v", err)
	}

	sourcePath := filepath.Join(dir, "source.pdf")
	numberedPath := filepath.Join(dir, "selected-pages-numbered.pdf")
	watermarkedPath := filepath.Join(dir, "watermarked-pages.pdf")
	customPath := filepath.Join(dir, "custom-position.pdf")

	if err := writeSampleMultiPagePDF(sourcePath); err != nil {
		fatalf("creating sample PDF: %v", err)
	}

	if err := pdf.AddPageNumbers(sourcePath, numberedPath, pdf.PageNumberOptions{
		Format:   "Page %d / %d",
		Pages:    []int{1, 3},
		FontSize: 12,
		Margin:   26,
	}); err != nil {
		fatalf("adding selected page numbers: %v", err)
	}

	if err := pdf.Watermark(sourcePath, watermarkedPath, pdf.WatermarkOptions{
		Text:     "CONFIDENTIAL",
		Pages:    []int{2},
		FontSize: 42,
		Angle:    35,
		Color:    [3]float64{0.75, 0.0, 0.0},
	}); err != nil {
		fatalf("adding watermarked page: %v", err)
	}

	if err := pdf.AddPageNumbers(sourcePath, customPath, pdf.PageNumberOptions{
		Format:   "Page %d of %d",
		Pages:    []int{1, 2, 3},
		FontSize: 11,
		X:        492,
		Y:        48,
		Color:    [3]float64{0.1, 0.4, 0.8},
	}); err != nil {
		fatalf("adding custom-position page numbers: %v", err)
	}

	fmt.Printf("Created sample PDF workflow in %s\n", dir)
	fmt.Printf("- %s\n", sourcePath)
	fmt.Printf("- %s\n", numberedPath)
	fmt.Printf("- %s\n", watermarkedPath)
	fmt.Printf("- %s\n", customPath)
}

func writeSampleMultiPagePDF(path string) error {
	doc, err := document.NewDocument(document.A4)
	if err != nil {
		return err
	}

	for i := 1; i <= 3; i++ {
		page := doc.NewPage()
		page.Fonts["F1"] = 0
		page.Contents = []byte(fmt.Sprintf("BT /F1 12 Tf 72 760 Td (Sample page %d) Tj ET", i))
	}

	var out *os.File
	out, err = os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = doc.WriteTo(out)
	return err
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
