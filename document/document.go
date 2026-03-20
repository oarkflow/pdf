package document

import (
	"io"
	"os"

	"github.com/oarkflow/pdf/core"
)

// Metadata holds PDF document information fields.
type Metadata struct {
	Title, Author, Subject, Keywords, Creator, Producer string
}

// PageDecorator is a function that returns content stream bytes for headers/footers.
type PageDecorator func(info PageInfo) []byte

// PageInfo provides context to a PageDecorator.
type PageInfo struct {
	Number int
	Total  int
	Width  float64
	Height float64
}

// WatermarkConfig configures a text watermark overlay.
type WatermarkConfig struct {
	Text     string
	FontSize float64
	Color    [3]float64 // RGB
	Opacity  float64
	Angle    float64 // degrees
}

// PDFALevel indicates the PDF/A conformance level.
type PDFALevel int

const (
	PDFA1b PDFALevel = iota
	PDFA2b
)

// Document is the high-level API for building a PDF.
type Document struct {
	pageSize  PageSize
	margins   Margins
	pages     []*Page
	metadata  Metadata
	header    PageDecorator
	footer    PageDecorator
	watermark *WatermarkConfig
	pdfaLevel *PDFALevel
	encConfig *core.EncryptionConfig
}

// NewDocument creates a new document with the given page size and default margins.
func NewDocument(pageSize PageSize) *Document {
	return &Document{
		pageSize: pageSize,
		margins:  DefaultMargins(),
	}
}

func (d *Document) SetMargins(m Margins)              { d.margins = m }
func (d *Document) AddPage(p *Page)                    { d.pages = append(d.pages, p) }
func (d *Document) SetHeader(fn PageDecorator)         { d.header = fn }
func (d *Document) SetFooter(fn PageDecorator)         { d.footer = fn }
func (d *Document) SetWatermark(cfg WatermarkConfig)   { d.watermark = &cfg }
func (d *Document) SetMetadata(meta Metadata)          { d.metadata = meta }
func (d *Document) SetEncryption(cfg core.EncryptionConfig) { d.encConfig = &cfg }
func (d *Document) Pages() []*Page                     { return d.pages }
func (d *Document) PageSize() PageSize                 { return d.pageSize }
func (d *Document) Margins() Margins                   { return d.margins }

// SetPDFA sets the target PDF/A conformance level.
func (d *Document) SetPDFA(level PDFALevel) {
	d.pdfaLevel = &level
}

// NewPage creates a new page with the document's page size and appends it.
func (d *Document) NewPage() *Page {
	p := NewPage(d.pageSize)
	d.pages = append(d.pages, p)
	return p
}

// Save writes the PDF to the given file path.
func (d *Document) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = d.WriteTo(f)
	return err
}

// WriteTo serializes the document as a complete PDF to w.
func (d *Document) WriteTo(w io.Writer) (int64, error) {
	wr := NewWriter()

	// Set metadata if any field is non-empty.
	info := make(map[string]string)
	if d.metadata.Title != "" {
		info["Title"] = d.metadata.Title
	}
	if d.metadata.Author != "" {
		info["Author"] = d.metadata.Author
	}
	if d.metadata.Subject != "" {
		info["Subject"] = d.metadata.Subject
	}
	if d.metadata.Keywords != "" {
		info["Keywords"] = d.metadata.Keywords
	}
	if d.metadata.Creator != "" {
		info["Creator"] = d.metadata.Creator
	}
	if d.metadata.Producer != "" {
		info["Producer"] = d.metadata.Producer
	}
	if len(info) > 0 {
		wr.SetInfo(info)
	}

	total := len(d.pages)
	for i, page := range d.pages {
		// Apply header/footer decorators to content stream.
		var content []byte
		content = append(content, page.Contents...)
		if d.header != nil {
			content = append(content, d.header(PageInfo{
				Number: i + 1, Total: total,
				Width: page.Size.Width, Height: page.Size.Height,
			})...)
		}
		if d.footer != nil {
			content = append(content, d.footer(PageInfo{
				Number: i + 1, Total: total,
				Width: page.Size.Width, Height: page.Size.Height,
			})...)
		}

		p := &Page{
			Size:      page.Size,
			Resources: page.Resources,
			Contents:  content,
			Fonts:     page.Fonts,
			Images:    page.Images,
		}
		wr.AddPage(p)
	}

	// PDF/A objects (stub).
	if d.pdfaLevel != nil {
		d.buildPDFAObjects(wr)
	}

	// Encryption (stub).
	if d.encConfig != nil {
		d.applyEncryption(wr)
	}

	return wr.WriteTo(w)
}
