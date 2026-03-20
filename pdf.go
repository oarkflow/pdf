package pdf

import (
	"fmt"
	"os"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/html"
	"github.com/oarkflow/pdf/layout"
)

// Common page sizes re-exported for convenience.
var (
	A4     = document.A4
	A3     = document.A3
	A5     = document.A5
	Letter = document.Letter
	Legal  = document.Legal
)

// Quick creates a simple PDF with text content and saves it to outputPath.
func Quick(text string, outputPath string) error {
	doc := document.NewDocument(document.A4)
	elements := []layout.Element{
		layout.NewParagraph(text),
	}

	pages := layout.RenderPages(elements,
		document.A4.Width, document.A4.Height,
		72, 72, 72, 72,
	)

	for _, pr := range pages {
		p := document.NewPage(document.PageSize{Width: pr.Width, Height: pr.Height})
		p.Contents = pr.Content
		for name, fe := range pr.Fonts {
			p.Fonts[name] = fe.ObjectNum
		}
		for name, ie := range pr.Images {
			p.Images[name] = ie
		}
		doc.AddPage(p)
	}

	return doc.Save(outputPath)
}

// FromHTML converts HTML content to a PDF file.
func FromHTML(htmlContent string, outputPath string, opts ...html.Options) error {
	var opt html.Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	result, err := html.Convert(htmlContent, opt)
	if err != nil {
		return fmt.Errorf("converting HTML: %w", err)
	}

	doc := document.NewDocument(document.PageSize{
		Width:  result.Config.Width,
		Height: result.Config.Height,
	})
	doc.SetMargins(document.Margins{
		Top:    result.Config.Margins[0],
		Right:  result.Config.Margins[1],
		Bottom: result.Config.Margins[2],
		Left:   result.Config.Margins[3],
	})

	if title, ok := result.Metadata["title"]; ok {
		doc.SetMetadata(document.Metadata{Title: title})
	}

	pages := layout.RenderPages(
		result.Elements,
		result.Config.Width, result.Config.Height,
		result.Config.Margins[0], result.Config.Margins[1],
		result.Config.Margins[2], result.Config.Margins[3],
	)

	for _, pr := range pages {
		p := document.NewPage(document.PageSize{Width: pr.Width, Height: pr.Height})
		p.Contents = pr.Content
		for name, fe := range pr.Fonts {
			p.Fonts[name] = fe.ObjectNum
		}
		for name, ie := range pr.Images {
			p.Images[name] = ie
		}
		doc.AddPage(p)
	}

	return doc.Save(outputPath)
}

// Merge merges multiple PDF files into a single output file.
// This is a simplified implementation that concatenates pages.
func Merge(outputPath string, inputPaths ...string) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("no input files provided")
	}

	// Verify all input files exist
	for _, path := range inputPaths {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("input file %s: %w", path, err)
		}
	}

	// PDF merging requires a reader/parser which is not yet implemented.
	// For now, return an informative error.
	return fmt.Errorf("PDF merge requires a PDF reader (not yet implemented)")
}

// NewDocument creates a new document with the given page size.
// If no page size is provided, A4 is used.
func NewDocument(pageSize ...document.PageSize) *document.Document {
	ps := document.A4
	if len(pageSize) > 0 {
		ps = pageSize[0]
	}
	return document.NewDocument(ps)
}
