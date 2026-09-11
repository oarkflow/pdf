package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/document"
)

func main() {
	dir := filepath.Join("examples", "advanced_tools", "out")
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatalf("creating output directory: %v", err)
	}

	source := filepath.Join(dir, "source.pdf")
	redacted := filepath.Join(dir, "redacted.pdf")
	translated := filepath.Join(dir, "translated.pdf")
	archived := filepath.Join(dir, "archive.pdf")
	graphPath := filepath.Join(dir, "graph.json")
	translationPath := filepath.Join(dir, "translations.json")

	if err := writeTextPDF(source, "The secret code is ALPHA-42."); err != nil {
		fatalf("creating source PDF: %v", err)
	}

	if err := pdf.Redact(source, redacted, pdf.RedactOptions{
		Texts:              []string{"ALPHA-42"},
		AllowVisualRegions: true,
		Verify:             true,
	}); err != nil {
		fatalf("redacting PDF: %v", err)
	}

	if err := os.WriteFile(translationPath, []byte(`{"ALPHA-42":"BETA-99"}`), 0o644); err != nil {
		fatalf("writing translation file: %v", err)
	}
	if err := pdf.TranslateWithDictionary(source, translationPath, translated, ""); err != nil {
		fatalf("translating PDF: %v", err)
	}

	report, err := pdf.ArchivePDF(source, archived, "")
	if err != nil {
		fatalf("archiving PDF: %v", err)
	}

	graph, err := pdf.BuildPDFGraph([]string{source, redacted, translated}, "")
	if err != nil {
		fatalf("building PDF graph: %v", err)
	}
	graphJSON, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		fatalf("marshaling graph: %v", err)
	}
	if err := os.WriteFile(graphPath, graphJSON, 0o644); err != nil {
		fatalf("writing graph JSON: %v", err)
	}

	fmt.Printf("Created advanced PDF tools examples in %s\n", dir)
	fmt.Printf("- %s\n", redacted)
	fmt.Printf("- %s\n", translated)
	fmt.Printf("- %s\n", archived)
	fmt.Printf("- %s\n", graphPath)
	fmt.Printf("PDF/A validation valid: %t\n", report.Valid)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func writeTextPDF(path, text string) error {
	doc, err := document.NewDocument(document.A4)
	if err != nil {
		return err
	}
	page := doc.NewPage()
	page.Fonts["F1"] = 0
	page.Contents = []byte("BT /F1 12 Tf 72 720 Td (" + text + ") Tj ET")
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
