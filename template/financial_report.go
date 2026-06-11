package template

import (
	"bytes"
	"fmt"
	"io"

	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
)

// FinancialReportData is a generic, data-driven native report definition.
// It is intentionally not tied to one company, schema, or report shape.
type FinancialReportData struct {
	Title      string
	Subject    string
	Author     string
	PageSize   document.PageSize
	Margins    document.Margins
	FooterText string
	PDFA       *document.PDFALevel
	PDFUA      *document.PDFUALevel
	Language   string
	Blocks     []FinancialReportBlock
}

// FinancialReportBlock is one ordered block in a native report.
type FinancialReportBlock struct {
	Heading   *FinancialReportHeading
	Table     *FinancialReportTable
	ImageGrid *FinancialReportImageGrid
	Spacer    float64
	PageBreak bool
}

// FinancialReportHeading renders a full-width title/header band.
type FinancialReportHeading struct {
	Text      string
	FontSize  float64
	Bold      bool
	Align     layout.Alignment
	TextColor [3]float64
	BgColor   *[3]float64
	Height    float64
}

// FinancialReportTable renders arbitrary rows and columns.
type FinancialReportTable struct {
	ColumnWidths []float64
	Header       []FinancialReportCell
	Rows         [][]FinancialReportCell
	CellPadding  float64
	BorderWidth  float64
	BorderColor  [3]float64
	HeaderBg     *[3]float64
	StripedBg    *[3]float64
}

// FinancialReportImageGrid renders one row of images with optional captions.
type FinancialReportImageGrid struct {
	Images       []FinancialReportImage
	ColumnWidths []float64
	Captions     []FinancialReportCell
	CellPadding  float64
	BorderWidth  float64
	BorderColor  [3]float64
}

// FinancialReportImage is an image in a report grid or table cell.
type FinancialReportImage struct {
	Data   []byte
	Alt    string
	Width  float64
	Height float64
	Align  layout.Alignment
	Bg     *[3]float64
}

// FinancialReportCell renders text or an image in a table.
type FinancialReportCell struct {
	Text      string
	Image     *FinancialReportImage
	FontSize  float64
	Bold      bool
	Align     layout.Alignment
	TextColor [3]float64
	Bg        *[3]float64
	Height    float64
}

// CompiledFinancialReport is a reusable native report plan.
type CompiledFinancialReport struct {
	elements   []layout.Element
	pages      []layout.PageResult
	docPages   []*document.Page
	pageSize   document.PageSize
	margins    document.Margins
	metadata   document.Metadata
	pdfa       *document.PDFALevel
	pdfua      *document.PDFUALevel
	language   string
	xmp        []byte
	footerText string
	sizeHint   int
}

// RenderFinancialReport renders a generic native report PDF document.
func RenderFinancialReport(data FinancialReportData) (*document.Document, error) {
	compiled, err := CompileFinancialReport(data)
	if err != nil {
		return nil, err
	}
	return compiled.Render()
}

// CompileFinancialReport converts a generic report definition into reusable
// layout elements. Use this for high-volume generation where the report shape
// is stable and each operation still needs a freshly serialized PDF.
func CompileFinancialReport(data FinancialReportData) (*CompiledFinancialReport, error) {
	if data.PageSize == (document.PageSize{}) {
		data.PageSize = document.A4
	}
	if data.Margins == (document.Margins{}) {
		data.Margins = document.Margins{Top: 36, Right: 36, Bottom: 42, Left: 36}
	}

	var elements []layout.Element
	for _, block := range data.Blocks {
		switch {
		case block.PageBreak:
			elements = append(elements, layout.NewPageBreak())
		case block.Spacer > 0:
			elements = append(elements, layout.NewSpacer(block.Spacer))
		case block.Heading != nil:
			elements = append(elements, renderFinancialHeading(*block.Heading, data.PageSize.Width-data.Margins.Left-data.Margins.Right))
		case block.Table != nil:
			elements = append(elements, renderFinancialTable(*block.Table))
		case block.ImageGrid != nil:
			elements = append(elements, renderFinancialImageGrid(*block.ImageGrid))
		}
	}

	render := layout.RenderPages
	tagged := data.PDFUA != nil
	if tagged {
		render = layout.RenderTaggedPages
	}
	pages := render(elements, data.PageSize.Width, data.PageSize.Height, data.Margins.Top, data.Margins.Right, data.Margins.Bottom, data.Margins.Left)
	if data.FooterText != "" {
		appendFinancialFooter(pages, data.FooterText, data.Margins.Left, tagged)
	}
	docPages := financialDocumentPages(pages)

	metadata := document.Metadata{
		Title:   data.Title,
		Subject: data.Subject,
		Author:  data.Author,
	}
	var xmp []byte
	if data.PDFA != nil || data.PDFUA != nil {
		xmp = document.BuildComplianceXMPMetadata(metadata, data.PDFA, data.PDFUA)
	}

	return &CompiledFinancialReport{
		elements: elements,
		pages:    pages,
		docPages: docPages,
		pageSize: data.PageSize,
		margins:  data.Margins,
		metadata: metadata,
		pdfa:     data.PDFA,
		pdfua:    data.PDFUA,
		language: data.Language,
		xmp:      xmp,
		sizeHint: estimateFinancialReportSize(docPages),
	}, nil
}

// Render renders a fresh PDF document from a compiled report plan.
func (c *CompiledFinancialReport) Render() (*document.Document, error) {
	if c == nil {
		return nil, fmt.Errorf("template: compiled financial report is nil")
	}
	doc, err := document.NewDocument(c.pageSize)
	if err != nil {
		return nil, err
	}
	doc.SetMargins(c.margins)
	doc.SetMetadata(c.metadata)
	if c.pdfa != nil {
		doc.SetPDFA(*c.pdfa)
	}
	if c.pdfua != nil {
		doc.SetPDFUA(*c.pdfua)
	}
	if c.language != "" {
		doc.SetLanguage(c.language)
	}
	for _, p := range c.docPages {
		doc.AddPage(p)
	}
	return doc, nil
}

// WriteStreamingTo serializes a fresh PDF directly from the compiled report.
func (c *CompiledFinancialReport) WriteStreamingTo(out io.Writer) error {
	if c == nil {
		return fmt.Errorf("template: compiled financial report is nil")
	}
	if out == nil {
		return fmt.Errorf("template: writer is nil")
	}
	sw, err := document.NewStreamingWriter(out)
	if err != nil {
		return err
	}
	sw.SetMetadata(c.metadata)
	if c.pdfa != nil {
		sw.SetPDFA(*c.pdfa)
	}
	if c.pdfua != nil {
		sw.SetPDFUA(*c.pdfua)
	}
	if c.language != "" {
		sw.SetLanguage(c.language)
	}
	if len(c.xmp) > 0 {
		sw.SetXMPMetadata(c.xmp)
	}
	for _, page := range c.docPages {
		if _, err := sw.AddPage(page); err != nil {
			return err
		}
	}
	return sw.Finish()
}

// EstimatedSize returns a conservative byte-size hint for output buffers.
func (c *CompiledFinancialReport) EstimatedSize() int {
	if c == nil || c.sizeHint <= 0 {
		return 64 * 1024
	}
	return c.sizeHint
}

func estimateFinancialReportSize(pages []*document.Page) int {
	size := 4096
	for _, page := range pages {
		size += len(page.Contents) + 2048
		for _, entry := range page.Images {
			if entry.Image == nil {
				continue
			}
			if len(entry.Image.RawStream) > 0 {
				size += len(entry.Image.RawStream)
				continue
			}
			if data, err := entry.Image.CompressedData(); err == nil {
				size += len(data)
			} else {
				size += len(entry.Image.Data)
			}
			if len(entry.Image.AlphaData) > 0 {
				if data, err := entry.Image.CompressedAlphaData(); err == nil {
					size += len(data)
				} else {
					size += len(entry.Image.AlphaData)
				}
			}
		}
	}
	return size
}

func appendFinancialFooter(pages []layout.PageResult, footer string, left float64, tagged bool) {
	var buf bytes.Buffer
	if tagged {
		buf.WriteString("/Artifact BMC\n")
	}
	buf.WriteString("0.35 0.40 0.45 rg BT /Helvetica 8 Tf ")
	buf.WriteString(fmt.Sprintf("%.2f %.2f Td (%s) Tj ET\n", left, 22.0, escapeFooterText(footer)))
	if tagged {
		buf.WriteString("EMC\n")
	}
	footerBytes := buf.Bytes()
	for i := range pages {
		pages[i].Content = append(pages[i].Content, footerBytes...)
	}
}

func financialDocumentPages(pages []layout.PageResult) []*document.Page {
	docPages := make([]*document.Page, 0, len(pages))
	for _, pr := range pages {
		p := document.NewPage(document.PageSize{Width: pr.Width, Height: pr.Height})
		p.Contents = pr.Content
		for _, fe := range pr.Fonts {
			p.FontEntries[fe.PDFName] = fe
		}
		for name, ie := range pr.Images {
			p.Images[name] = ie
		}
		p.Annotations = pr.Links
		p.Structure = append(p.Structure, pr.Structure...)
		docPages = append(docPages, p)
	}
	return docPages
}

func renderFinancialHeading(h FinancialReportHeading, width float64) layout.Element {
	if h.FontSize == 0 {
		h.FontSize = 12
	}
	if h.TextColor == [3]float64{} {
		h.TextColor = [3]float64{0, 0, 0}
	}
	if h.Align == 0 {
		h.Align = layout.AlignLeft
	}
	table := layout.NewTable(1)
	table.CellPadding = 0
	table.BorderWidth = 0
	table.SetColumnWidths(width)
	table.Rows = append(table.Rows, layout.TableRow{Cells: []layout.TableCell{{
		Content: reportParagraph(h.Text, h.FontSize, h.Bold, h.TextColor, h.Align, h.Height),
		Bg:      h.BgColor,
	}}})
	return table
}

func renderFinancialTable(src FinancialReportTable) layout.Element {
	cols := len(src.ColumnWidths)
	if cols == 0 && len(src.Header) > 0 {
		cols = len(src.Header)
	}
	if cols == 0 && len(src.Rows) > 0 {
		cols = len(src.Rows[0])
	}
	if cols == 0 {
		return layout.NewSpacer(0)
	}

	table := layout.NewTable(cols)
	table.SetColumnWidths(src.ColumnWidths...)
	table.CellPadding = src.CellPadding
	if table.CellPadding == 0 {
		table.CellPadding = 4
	}
	table.BorderWidth = src.BorderWidth
	table.BorderColor = src.BorderColor
	if table.BorderColor == [3]float64{} {
		table.BorderColor = [3]float64{0.82, 0.84, 0.86}
	}
	table.HeaderBg = src.HeaderBg
	table.StripedBg = src.StripedBg

	if len(src.Header) > 0 {
		table.Rows = append(table.Rows, reportRow(src.Header))
		table.HeaderRows = 1
	}
	for _, row := range src.Rows {
		table.Rows = append(table.Rows, reportRow(row))
	}
	return table
}

func renderFinancialImageGrid(grid FinancialReportImageGrid) layout.Element {
	cols := len(grid.Images)
	if cols == 0 {
		return layout.NewSpacer(0)
	}
	table := layout.NewTable(cols)
	table.SetColumnWidths(grid.ColumnWidths...)
	table.CellPadding = grid.CellPadding
	if table.CellPadding == 0 {
		table.CellPadding = 4
	}
	table.BorderWidth = grid.BorderWidth
	table.BorderColor = grid.BorderColor
	if table.BorderColor == [3]float64{} {
		table.BorderColor = [3]float64{0.82, 0.84, 0.86}
	}

	imageRow := layout.TableRow{}
	for _, image := range grid.Images {
		img := image
		imageRow.Cells = append(imageRow.Cells, layout.TableCell{
			Content: reportImage(img),
			Bg:      img.Bg,
		})
	}
	table.Rows = append(table.Rows, imageRow)
	if len(grid.Captions) > 0 {
		table.Rows = append(table.Rows, reportRow(grid.Captions))
	}
	return table
}

func reportRow(cells []FinancialReportCell) layout.TableRow {
	row := layout.TableRow{}
	for _, cell := range cells {
		row.Cells = append(row.Cells, layout.TableCell{
			Content: reportCellContent(cell),
			Bg:      cell.Bg,
		})
	}
	return row
}

func reportCellContent(cell FinancialReportCell) layout.Element {
	if cell.Image != nil {
		return reportImage(*cell.Image)
	}
	fontSize := cell.FontSize
	if fontSize == 0 {
		fontSize = 10
	}
	color := cell.TextColor
	if color == [3]float64{} {
		color = [3]float64{0, 0, 0}
	}
	return reportParagraph(cell.Text, fontSize, cell.Bold, color, cell.Align, cell.Height)
}

func reportImage(image FinancialReportImage) layout.Element {
	img := layout.NewImage(image.Data, 0, 0)
	img.Width = image.Width
	img.Height = image.Height
	img.Align = image.Align
	img.Alt = image.Alt
	return img
}

func reportParagraph(text string, fontSize float64, bold bool, color [3]float64, align layout.Alignment, height float64) *layout.Paragraph {
	p := layout.NewStyledParagraph(layout.TextRun{
		Text:     text,
		FontName: "Helvetica",
		FontSize: fontSize,
		Bold:     bold,
		Color:    color,
	})
	p.Align = align
	p.LineHeight = 1.0
	p.SpaceAfter = 0
	if height > 0 {
		p.MaxLines = 1
	}
	return p
}

func escapeFooterText(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
