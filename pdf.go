package pdf

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/html"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/reader"
	"github.com/oarkflow/pdf/template"
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
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	doc, err := document.NewDocument(document.A4)
	if err != nil {
		return err
	}
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
		for _, fe := range pr.Fonts {
			p.FontEntries[fe.PDFName] = fe
		}
		for name, ie := range pr.Images {
			p.Images[name] = ie
		}
		applyExtGStates(p, pr.ExtGStates)
		p.Annotations = pr.Links
		doc.AddPage(p)
	}

	return doc.Save(outputPath)
}

// FromHTML converts HTML content to a PDF file.
func FromHTML(htmlContent string, outputPath string, opts ...html.Options) error {
	if htmlContent == "" {
		return errors.New("pdf: HTML content is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	var opt html.Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	result, err := html.Convert(htmlContent, opt)
	if err != nil {
		return fmt.Errorf("converting HTML: %w", err)
	}

	doc, err := document.NewDocument(document.PageSize{
		Width:  result.Config.Width,
		Height: result.Config.Height,
	})
	if err != nil {
		return fmt.Errorf("creating document: %w", err)
	}
	doc.SetMargins(document.Margins{
		Top:    result.Config.Margins[0],
		Right:  result.Config.Margins[1],
		Bottom: result.Config.Margins[2],
		Left:   result.Config.Margins[3],
	})

	if title, ok := result.Metadata["title"]; ok {
		doc.SetMetadata(document.Metadata{Title: title})
	}
	if opt.Encryption != nil {
		doc.SetEncryption(*opt.Encryption)
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
		for _, fe := range pr.Fonts {
			p.FontEntries[fe.PDFName] = fe
		}
		for name, ie := range pr.Images {
			p.Images[name] = ie
		}
		applyExtGStates(p, pr.ExtGStates)
		p.Annotations = pr.Links
		doc.AddPage(p)
	}

	return doc.Save(outputPath)
}

// Merge merges multiple PDF files into a single output file.
// This is a simplified implementation that concatenates pages.
func Merge(outputPath string, inputPaths ...string) error {
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	if len(inputPaths) == 0 {
		return fmt.Errorf("no input files provided")
	}

	// Verify all input files exist
	for _, path := range inputPaths {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("input file %s: %w", path, err)
		}
	}

	return reader.MergeFiles(inputPaths, outputPath)
}

// NewDocument creates a new document with the given page size.
// If no page size is provided, A4 is used.
func NewDocument(pageSize ...document.PageSize) (*document.Document, error) {
	ps := document.A4
	if len(pageSize) > 0 {
		ps = pageSize[0]
	}
	return document.NewDocument(ps)
}

// FromHTMLStreaming converts HTML content and writes the PDF directly to out
// without buffering the entire document in memory.
func FromHTMLStreaming(htmlContent string, out io.Writer, opts ...html.Options) error {
	if htmlContent == "" {
		return errors.New("pdf: HTML content is empty")
	}
	if out == nil {
		return errors.New("pdf: writer is nil")
	}
	var opt html.Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	result, err := html.Convert(htmlContent, opt)
	if err != nil {
		return fmt.Errorf("converting HTML: %w", err)
	}

	doc, err := document.NewDocument(document.PageSize{
		Width:  result.Config.Width,
		Height: result.Config.Height,
	})
	if err != nil {
		return fmt.Errorf("creating document: %w", err)
	}
	doc.SetMargins(document.Margins{
		Top:    result.Config.Margins[0],
		Right:  result.Config.Margins[1],
		Bottom: result.Config.Margins[2],
		Left:   result.Config.Margins[3],
	})

	if title, ok := result.Metadata["title"]; ok {
		doc.SetMetadata(document.Metadata{Title: title})
	}
	if opt.Encryption != nil {
		doc.SetEncryption(*opt.Encryption)
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
		for _, fe := range pr.Fonts {
			p.FontEntries[fe.PDFName] = fe
		}
		for name, ie := range pr.Images {
			p.Images[name] = ie
		}
		applyExtGStates(p, pr.ExtGStates)
		p.Annotations = pr.Links
		doc.AddPage(p)
	}

	return doc.WriteStreamingTo(out)
}

func applyExtGStates(page *document.Page, states map[string]layout.ExtGState) {
	if page == nil || len(states) == 0 {
		return
	}
	gsDict := core.NewDictionary()
	for name, gs := range states {
		d := core.NewDictionary()
		d.Set("Type", core.PdfName("ExtGState"))
		d.Set("ca", core.PdfNumber(gs.FillAlpha))
		d.Set("CA", core.PdfNumber(gs.StrokeAlpha))
		gsDict.Set(name, d)
	}
	page.Resources.Set("ExtGState", gsDict)
}

// FromURL fetches a webpage by URL and converts it to a PDF file.
// The URL is automatically used as BaseURL for resolving relative resources
// (CSS, images) unless overridden in opts.
func FromURL(url string, outputPath string, opts ...html.Options) error {
	if url == "" {
		return errors.New("pdf: URL is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}

	htmlContent, err := fetchHTML(url)
	if err != nil {
		return fmt.Errorf("fetching URL: %w", err)
	}

	var opt html.Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.BaseURL == "" {
		opt.BaseURL = url
	}

	return FromHTML(htmlContent, outputPath, opt)
}

// FromURLStreaming fetches a webpage by URL and writes the PDF directly to out.
func FromURLStreaming(url string, out io.Writer, opts ...html.Options) error {
	if url == "" {
		return errors.New("pdf: URL is empty")
	}
	if out == nil {
		return errors.New("pdf: writer is nil")
	}

	htmlContent, err := fetchHTML(url)
	if err != nil {
		return fmt.Errorf("fetching URL: %w", err)
	}

	var opt html.Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.BaseURL == "" {
		opt.BaseURL = url
	}

	return FromHTMLStreaming(htmlContent, out, opt)
}

// fetchHTML fetches the HTML content from a URL using the existing Fetcher.
func fetchHTML(url string) (string, error) {
	f := html.NewFetcher(url)
	data, err := f.Fetch(url)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromHTMLTemplate renders an HTML template string using fasttpl (with support
// for {{ if }}, {{ range }}, filters, nested keys, etc.) and converts the
// resulting HTML to a PDF file.
func FromHTMLTemplate(htmlTpl string, data map[string]any, outputPath string, opts ...html.Options) error {
	if htmlTpl == "" {
		return errors.New("pdf: HTML template is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}

	rendered, err := template.RenderHTML(htmlTpl, data)
	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	return FromHTML(rendered, outputPath, opts...)
}

// FromHTMLTemplateFile compiles an HTML template file using fasttpl and
// converts the rendered HTML to a PDF file.
func FromHTMLTemplateFile(templatePath string, data map[string]any, outputPath string, opts ...html.Options) error {
	if templatePath == "" {
		return errors.New("pdf: template path is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}

	rendered, err := template.RenderHTMLFile(templatePath, data)
	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	return FromHTML(rendered, outputPath, opts...)
}

// FromHTMLTemplateStreaming renders an HTML template using fasttpl and writes
// the PDF directly to the writer.
func FromHTMLTemplateStreaming(htmlTpl string, data map[string]any, out io.Writer, opts ...html.Options) error {
	if htmlTpl == "" {
		return errors.New("pdf: HTML template is empty")
	}
	if out == nil {
		return errors.New("pdf: writer is nil")
	}

	rendered, err := template.RenderHTML(htmlTpl, data)
	if err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	return FromHTMLStreaming(rendered, out, opts...)
}
