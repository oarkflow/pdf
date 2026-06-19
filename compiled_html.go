package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/html"
	"github.com/oarkflow/pdf/layout"
)

// CompiledHTML is a reusable render plan for HTML that is generated many times.
//
// CompileHTML performs HTML parsing, CSS application, resource loading, image
// decoding, and pagination once. Each WriteStreamingTo call only serializes the
// prepared pages to PDF, which is substantially faster for repeated reports.
type CompiledHTML struct {
	pages      []layout.PageResult
	pageSize   document.PageSize
	margins    document.Margins
	metadata   document.Metadata
	encryption *core.EncryptionConfig
	pdfa       *document.PDFALevel
	pdfua      *document.PDFUALevel
	language   string
	xmp        []byte
	doc        *document.Document
}

// HTMLComplianceOptions controls standards-oriented HTML-to-PDF generation.
type HTMLComplianceOptions struct {
	PDFA     document.PDFALevel
	PDFUA    document.PDFUALevel
	Language string
}

// FrozenPDF is an immutable PDF byte artifact optimized for repeated writes.
type FrozenPDF struct {
	data []byte
}

// DefaultHTMLComplianceOptions returns the default standards profile used by
// compliant HTML generation.
func DefaultHTMLComplianceOptions() HTMLComplianceOptions {
	return HTMLComplianceOptions{
		PDFA:     document.PDFA2b,
		PDFUA:    document.PDFUA1,
		Language: "en-US",
	}
}

// CompileHTML prepares HTML content for repeated lean PDF generation.
// It is equivalent to CompileLeanHTML and is kept for compatibility.
func CompileHTML(htmlContent string, opts ...html.Options) (*CompiledHTML, error) {
	return compileHTML(htmlContent, false, HTMLComplianceOptions{}, opts...)
}

// CompileLeanHTML prepares HTML content for repeated lean PDF generation.
func CompileLeanHTML(htmlContent string, opts ...html.Options) (*CompiledHTML, error) {
	return compileHTML(htmlContent, false, HTMLComplianceOptions{}, opts...)
}

// CompileCompliantHTML prepares HTML content for repeated compliant PDF
// generation using DefaultHTMLComplianceOptions.
func CompileCompliantHTML(htmlContent string, opts ...html.Options) (*CompiledHTML, error) {
	return CompileCompliantHTMLWithOptions(htmlContent, DefaultHTMLComplianceOptions(), opts...)
}

// CompileCompliantHTMLWithOptions prepares HTML content for repeated compliant
// PDF generation using the supplied compliance profile.
func CompileCompliantHTMLWithOptions(htmlContent string, compliance HTMLComplianceOptions, opts ...html.Options) (*CompiledHTML, error) {
	if compliance.Language == "" {
		compliance.Language = "en-US"
	}
	return compileHTML(htmlContent, true, compliance, opts...)
}

// WriteLeanHTMLToPDF converts HTML content to a lean PDF and writes it to out.
func WriteLeanHTMLToPDF(out io.Writer, htmlContent string, opts ...html.Options) error {
	compiled, err := CompileLeanHTML(htmlContent, opts...)
	if err != nil {
		return err
	}
	return compiled.WriteStreamingTo(out)
}

// WriteCompliantHTMLToPDF converts HTML content to a compliant PDF using
// DefaultHTMLComplianceOptions and writes it to out.
func WriteCompliantHTMLToPDF(out io.Writer, htmlContent string, opts ...html.Options) error {
	compiled, err := CompileCompliantHTML(htmlContent, opts...)
	if err != nil {
		return err
	}
	return compiled.WriteStreamingTo(out)
}

// WriteCompliantHTMLToPDFWithOptions converts HTML content to a compliant PDF
// using the supplied compliance profile and writes it to out.
func WriteCompliantHTMLToPDFWithOptions(out io.Writer, htmlContent string, compliance HTMLComplianceOptions, opts ...html.Options) error {
	compiled, err := CompileCompliantHTMLWithOptions(htmlContent, compliance, opts...)
	if err != nil {
		return err
	}
	return compiled.WriteStreamingTo(out)
}

func compileHTML(htmlContent string, compliant bool, compliance HTMLComplianceOptions, opts ...html.Options) (*CompiledHTML, error) {
	if htmlContent == "" {
		return nil, errors.New("pdf: HTML content is empty")
	}
	var opt html.Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if compliant && opt.Encryption != nil {
		return nil, errors.New("pdf: compliant HTML generation does not support encryption")
	}

	result, err := html.Convert(htmlContent, opt)
	if err != nil {
		return nil, fmt.Errorf("converting HTML: %w", err)
	}

	hasHF := len(result.HeaderElements) > 0 || len(result.FooterElements) > 0
	var pages []layout.PageResult
	if hasHF {
		if compliant {
			pages = layout.RenderTaggedPagesWithHeaderFooter(
				result.Elements, result.HeaderElements, result.FooterElements,
				result.Config.Width, result.Config.Height,
				result.Config.Margins[0], result.Config.Margins[1],
				result.Config.Margins[2], result.Config.Margins[3],
			)
		} else {
			pages = layout.RenderPagesWithHeaderFooter(
				result.Elements, result.HeaderElements, result.FooterElements,
				result.Config.Width, result.Config.Height,
				result.Config.Margins[0], result.Config.Margins[1],
				result.Config.Margins[2], result.Config.Margins[3],
			)
		}
	} else {
		if compliant {
			pages = layout.RenderTaggedPages(
				result.Elements,
				result.Config.Width, result.Config.Height,
				result.Config.Margins[0], result.Config.Margins[1],
				result.Config.Margins[2], result.Config.Margins[3],
			)
		} else {
			pages = layout.RenderPages(
				result.Elements,
				result.Config.Width, result.Config.Height,
				result.Config.Margins[0], result.Config.Margins[1],
				result.Config.Margins[2], result.Config.Margins[3],
			)
		}
	}

	meta := document.Metadata{}
	if title, ok := result.Metadata["title"]; ok {
		meta.Title = title
	}

	compiled := &CompiledHTML{
		pages:    pages,
		pageSize: document.PageSize{Width: result.Config.Width, Height: result.Config.Height},
		margins: document.Margins{
			Top:    result.Config.Margins[0],
			Right:  result.Config.Margins[1],
			Bottom: result.Config.Margins[2],
			Left:   result.Config.Margins[3],
		},
		metadata:   meta,
		encryption: opt.Encryption,
	}
	if compliant {
		compiled.pdfa = &compliance.PDFA
		compiled.pdfua = &compliance.PDFUA
		compiled.language = compliance.Language
		compiled.xmp = document.BuildComplianceXMPMetadata(meta, compiled.pdfa, compiled.pdfua)
	}
	compiled.doc, err = compiled.buildDocument()
	if err != nil {
		return nil, err
	}
	return compiled, nil
}

// WriteStreamingTo writes a PDF from the compiled render plan.
func (c *CompiledHTML) WriteStreamingTo(out io.Writer) error {
	if c == nil {
		return errors.New("pdf: compiled HTML is nil")
	}
	if out == nil {
		return errors.New("pdf: writer is nil")
	}
	if c.doc != nil {
		return c.doc.WriteStreamingTo(out)
	}

	doc, err := c.buildDocument()
	if err != nil {
		return err
	}
	return doc.WriteStreamingTo(out)
}

func (c *CompiledHTML) buildDocument() (*document.Document, error) {
	doc, err := document.NewDocument(c.pageSize)
	if err != nil {
		return nil, fmt.Errorf("creating document: %w", err)
	}
	doc.SetMargins(c.margins)
	doc.SetMetadata(c.metadata)
	if c.encryption != nil {
		doc.SetEncryption(*c.encryption)
	}
	if c.pdfa != nil {
		doc.SetPDFA(*c.pdfa)
	}
	if c.pdfua != nil {
		doc.SetPDFUA(*c.pdfua)
	}
	if c.language != "" {
		doc.SetLanguage(c.language)
	}
	if len(c.xmp) > 0 {
		doc.SetXMPMetadata(c.xmp)
	}

	for _, pr := range c.pages {
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
		p.Structure = append(p.Structure, pr.Structure...)
		doc.AddPage(p)
	}

	return doc, nil
}

// WriteTo writes a PDF from the compiled render plan.
func (c *CompiledHTML) WriteTo(out io.Writer) (int64, error) {
	counter := &compiledCountingWriter{w: out}
	if err := c.WriteStreamingTo(counter); err != nil {
		return counter.written, err
	}
	return counter.written, nil
}

// Freeze renders the compiled HTML once and stores the resulting PDF bytes.
func (c *CompiledHTML) Freeze() (*FrozenPDF, error) {
	if c == nil {
		return nil, errors.New("pdf: compiled HTML is nil")
	}
	var buf bytes.Buffer
	if err := c.WriteStreamingTo(&buf); err != nil {
		return nil, err
	}
	data := append([]byte(nil), buf.Bytes()...)
	return &FrozenPDF{data: data}, nil
}

// Bytes returns a copy of the frozen PDF bytes.
func (f *FrozenPDF) Bytes() []byte {
	if f == nil {
		return nil
	}
	return append([]byte(nil), f.data...)
}

// WriteTo writes the frozen PDF bytes to out.
func (f *FrozenPDF) WriteTo(out io.Writer) (int64, error) {
	if f == nil {
		return 0, errors.New("pdf: frozen PDF is nil")
	}
	if out == nil {
		return 0, errors.New("pdf: writer is nil")
	}
	n, err := out.Write(f.data)
	if err == nil && n != len(f.data) {
		err = io.ErrShortWrite
	}
	return int64(n), err
}

// WriteStreamingTo writes the frozen PDF bytes to out.
func (f *FrozenPDF) WriteStreamingTo(out io.Writer) error {
	_, err := f.WriteTo(out)
	return err
}

type compiledCountingWriter struct {
	w       io.Writer
	written int64
}

func (w *compiledCountingWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.written += int64(n)
	return n, err
}
