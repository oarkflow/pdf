package document

import (
	"errors"
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
//
// Document is not safe for concurrent use. Callers must synchronize access externally.
type Document struct {
	pageSize   PageSize
	margins    Margins
	pages      []*Page
	metadata   Metadata
	header     PageDecorator
	footer     PageDecorator
	watermark  *WatermarkConfig
	pdfaLevel  *PDFALevel
	encConfig  *core.EncryptionConfig
	outline    *Outline
	tagged     bool
	structTree *StructureTree
}

// NewDocument creates a new document with the given page size and default margins.
func NewDocument(pageSize PageSize) (*Document, error) {
	if pageSize.Width <= 0 || pageSize.Height <= 0 {
		return nil, errors.New("document: page size width and height must be positive")
	}
	return &Document{
		pageSize: pageSize,
		margins:  DefaultMargins(),
	}, nil
}

func (d *Document) SetMargins(m Margins) { d.margins = m }
func (d *Document) AddPage(p *Page) {
	if p == nil {
		return
	}
	d.pages = append(d.pages, p)
}
func (d *Document) SetHeader(fn PageDecorator)              { d.header = fn }
func (d *Document) SetFooter(fn PageDecorator)              { d.footer = fn }
func (d *Document) SetWatermark(cfg WatermarkConfig)        { d.watermark = &cfg }
func (d *Document) SetMetadata(meta Metadata)               { d.metadata = meta }
func (d *Document) SetEncryption(cfg core.EncryptionConfig) { d.encConfig = &cfg }
func (d *Document) Pages() []*Page                          { return d.pages }
func (d *Document) PageSize() PageSize                      { return d.pageSize }
func (d *Document) Margins() Margins                        { return d.margins }

// SetPDFA sets the target PDF/A conformance level.
func (d *Document) SetPDFA(level PDFALevel) {
	d.pdfaLevel = &level
}

// SetOutline sets the document outline (bookmarks).
func (d *Document) SetOutline(outline *Outline) { d.outline = outline }

// AddBookmark appends a top-level bookmark pointing to the given 0-based page index.
func (d *Document) AddBookmark(title string, pageIndex int) {
	if d.outline == nil {
		d.outline = &Outline{}
	}
	d.outline.Items = append(d.outline.Items, OutlineItem{Title: title, Page: pageIndex})
}

// AddBookmarkWithChildren appends a top-level bookmark with nested children.
func (d *Document) AddBookmarkWithChildren(title string, pageIndex int, children []OutlineItem) {
	if d.outline == nil {
		d.outline = &Outline{}
	}
	d.outline.Items = append(d.outline.Items, OutlineItem{Title: title, Page: pageIndex, Children: children})
}

// EnableTagging turns on tagged PDF (PDF/UA basic compliance).
func (d *Document) EnableTagging() {
	d.tagged = true
	d.structTree = NewStructureTree()
}

// IsTagged returns whether tagging is enabled.
func (d *Document) IsTagged() bool { return d.tagged }

// StructTree returns the structure tree, or nil if tagging is not enabled.
func (d *Document) StructTree() *StructureTree { return d.structTree }

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
			Size:        page.Size,
			Resources:   page.Resources,
			Contents:    content,
			Fonts:       page.Fonts,
			Images:      page.Images,
			Annotations: page.Annotations,
		}
		if _, err := wr.AddPage(p); err != nil {
			return 0, err
		}
	}

	// PDF/A objects.
	if d.pdfaLevel != nil {
		if err := d.buildPDFAObjects(wr); err != nil {
			return 0, err
		}
	}

	// Build outlines if set.
	if d.outline != nil {
		if objNum := d.outline.Build(wr); objNum > 0 {
			wr.SetOutlines(objNum)
		}
	}

	// Tagged PDF structure tree.
	if d.tagged && d.structTree != nil {
		wr.SetMarkInfo(true)
		rootObjNum := d.structTree.Build(wr)
		if rootObjNum > 0 {
			wr.SetStructTreeRoot(rootObjNum)
		}
	}

	// Encryption.
	if d.encConfig != nil {
		if err := d.applyEncryption(wr); err != nil {
			return 0, err
		}
	}

	return wr.WriteTo(w)
}

// WriteStreamingTo serializes the document directly to w without buffering the
// entire PDF in memory. This is more memory-efficient for large documents.
func (d *Document) WriteStreamingTo(w io.Writer) error {
	if d.encConfig != nil {
		_, err := d.WriteTo(w)
		return err
	}

	sw, err := NewStreamingWriter(w)
	if err != nil {
		return err
	}

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
		sw.SetInfo(info)
	}

	total := len(d.pages)
	for i, page := range d.pages {
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
			Size:        page.Size,
			Resources:   page.Resources,
			Contents:    content,
			Fonts:       page.Fonts,
			Images:      page.Images,
			Annotations: page.Annotations,
		}
		if _, err := sw.AddPage(p); err != nil {
			return err
		}
	}

	return sw.Finish()
}
