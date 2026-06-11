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
	doc        *document.Document
}

// FrozenPDF is an immutable PDF byte artifact optimized for repeated writes.
type FrozenPDF struct {
	data []byte
}

// CompileHTML prepares HTML content for repeated PDF generation.
func CompileHTML(htmlContent string, opts ...html.Options) (*CompiledHTML, error) {
	if htmlContent == "" {
		return nil, errors.New("pdf: HTML content is empty")
	}
	var opt html.Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	result, err := html.Convert(htmlContent, opt)
	if err != nil {
		return nil, fmt.Errorf("converting HTML: %w", err)
	}

	pages := layout.RenderPages(
		result.Elements,
		result.Config.Width, result.Config.Height,
		result.Config.Margins[0], result.Config.Margins[1],
		result.Config.Margins[2], result.Config.Margins[3],
	)

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
