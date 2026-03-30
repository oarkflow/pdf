package converter

import (
	"os"
	"strings"
	"testing"
)

func TestConvertInvoice(t *testing.T) {
	data, err := os.ReadFile("../invoice.pdf")
	if err != nil {
		t.Skip("invoice.pdf not found, skipping integration test")
	}

	conv, err := New(data, ConvertOptions{
		Mode:          "reflowed",
		ExtractImages: true,
		DetectTables:  true,
	})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if conv.NumPages() < 1 {
		t.Fatal("expected at least 1 page")
	}

	result, err := conv.Convert()
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	if result.HTML == "" {
		t.Fatal("expected non-empty HTML output")
	}

	if !strings.Contains(result.HTML, "<!DOCTYPE html>") {
		t.Error("HTML should start with DOCTYPE")
	}

	if !strings.Contains(result.HTML, "<div class=\"pdf-page\"") {
		t.Error("HTML should contain page divs")
	}

	if len(result.Pages) != conv.NumPages() {
		t.Errorf("expected %d pages, got %d", conv.NumPages(), len(result.Pages))
	}

	t.Logf("Converted %d pages, HTML length: %d bytes", len(result.Pages), len(result.HTML))
	t.Logf("Metadata: %v", result.Metadata)
	for i, p := range result.Pages {
		t.Logf("Page %d: %d lines, %d paragraphs, %d images, %d links, %d tables",
			i, len(p.Lines), len(p.Paragraphs), len(p.Images), len(p.Links), len(p.Tables))
	}
}

func TestConvertPositioned(t *testing.T) {
	data, err := os.ReadFile("../invoice.pdf")
	if err != nil {
		t.Skip("invoice.pdf not found, skipping")
	}

	conv, err := New(data, ConvertOptions{Mode: "positioned"})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	result, err := conv.Convert()
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	if !strings.Contains(result.HTML, "position: relative") {
		t.Error("positioned mode should use relative positioning on pages")
	}
	if !strings.Contains(result.HTML, "text-span") {
		t.Error("positioned mode should use text-span class")
	}
}

func TestConvertPageRange(t *testing.T) {
	data, err := os.ReadFile("../spotlight.pdf")
	if err != nil {
		t.Skip("spotlight.pdf not found, skipping")
	}

	conv, err := New(data, ConvertOptions{Pages: []int{0}})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	result, err := conv.Convert()
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	if len(result.Pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(result.Pages))
	}
}

func TestReconstructLines(t *testing.T) {
	spans := []StyledSpan{
		{Text: "Hello", X: 10, Y: 700, FontSize: 12},
		{Text: "World", X: 60, Y: 700, FontSize: 12},
		{Text: "Second line", X: 10, Y: 685, FontSize: 12},
	}

	lines := reconstructLines(spans, 800)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	if len(lines[0].Spans) != 2 {
		t.Errorf("first line should have 2 spans, got %d", len(lines[0].Spans))
	}
	if len(lines[1].Spans) != 1 {
		t.Errorf("second line should have 1 span, got %d", len(lines[1].Spans))
	}
}

func TestDetectHeadings(t *testing.T) {
	lines := []Line{
		{Spans: []StyledSpan{{Text: "Title", FontSize: 24}}, Y: 700},
		{Spans: []StyledSpan{{Text: "Body text", FontSize: 12}}, Y: 680},
		{Spans: []StyledSpan{{Text: "More body", FontSize: 12}}, Y: 665},
	}

	result := detectHeadings(lines)
	if !result[0].IsHeading {
		t.Error("first line should be detected as heading")
	}
	if result[1].IsHeading {
		t.Error("second line should not be a heading")
	}
}
