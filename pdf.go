package pdf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/oarkflow/pdf/converter"
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

type PDFALevel = document.PDFALevel
type PDFUALevel = document.PDFUALevel

const (
	PDFA1b = document.PDFA1b
	PDFA2b = document.PDFA2b
	PDFA4  = document.PDFA4
	PDFUA1 = document.PDFUA1
	PDFUA2 = document.PDFUA2
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
	return FromLeanHTML(htmlContent, outputPath, opts...)
}

// FromLeanHTML converts HTML content to a lean PDF file.
func FromLeanHTML(htmlContent string, outputPath string, opts ...html.Options) error {
	if htmlContent == "" {
		return errors.New("pdf: HTML content is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteLeanHTMLToPDF(f, htmlContent, opts...)
}

// FromCompliantHTML converts HTML content to a compliant PDF file using
// DefaultHTMLComplianceOptions.
func FromCompliantHTML(htmlContent string, outputPath string, opts ...html.Options) error {
	return FromCompliantHTMLWithOptions(htmlContent, outputPath, DefaultHTMLComplianceOptions(), opts...)
}

// FromCompliantHTMLWithOptions converts HTML content to a compliant PDF file
// using the supplied compliance profile.
func FromCompliantHTMLWithOptions(htmlContent string, outputPath string, compliance HTMLComplianceOptions, opts ...html.Options) error {
	if htmlContent == "" {
		return errors.New("pdf: HTML content is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteCompliantHTMLToPDFWithOptions(f, htmlContent, compliance, opts...)
}

// ToHTML converts a PDF file to an HTML document.
func ToHTML(inputPath string, opts ...converter.ConvertOptions) (string, error) {
	if inputPath == "" {
		return "", errors.New("pdf: input path is empty")
	}
	var opt converter.ConvertOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	conv, err := converter.NewFromFile(inputPath, opt)
	if err != nil {
		return "", err
	}
	result, err := conv.Convert()
	if err != nil {
		return "", err
	}
	return result.HTML, nil
}

// ToText converts a PDF file to plain text.
func ToText(inputPath string, opts ...converter.ConvertOptions) (string, error) {
	if inputPath == "" {
		return "", errors.New("pdf: input path is empty")
	}
	var opt converter.ConvertOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	conv, err := converter.NewFromFile(inputPath, opt)
	if err != nil {
		return "", err
	}
	return conv.ConvertText()
}

// ToMarkdown converts a PDF file to Markdown.
func ToMarkdown(inputPath string, opts ...converter.ConvertOptions) (string, error) {
	if inputPath == "" {
		return "", errors.New("pdf: input path is empty")
	}
	var opt converter.ConvertOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	conv, err := converter.NewFromFile(inputPath, opt)
	if err != nil {
		return "", err
	}
	return conv.ConvertMarkdown()
}

// ToJSON converts a PDF file to structured JSON.
func ToJSON(inputPath string, opts ...converter.ConvertOptions) ([]byte, error) {
	if inputPath == "" {
		return nil, errors.New("pdf: input path is empty")
	}
	var opt converter.ConvertOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	conv, err := converter.NewFromFile(inputPath, opt)
	if err != nil {
		return nil, err
	}
	return conv.ConvertJSON()
}

// PDFInfo describes a PDF document.
type PDFInfo struct {
	Path        string                  `json:"path,omitempty"`
	Pages       int                     `json:"pages"`
	Encrypted   bool                    `json:"encrypted"`
	Metadata    map[string]string       `json:"metadata,omitempty"`
	PageSizes   []PDFPageSize           `json:"pageSizes,omitempty"`
	Outlines    []reader.OutlineInfo    `json:"outlines,omitempty"`
	Annotations []reader.AnnotationInfo `json:"annotations,omitempty"`
	Trailer     map[string]string       `json:"trailer,omitempty"`
	Error       string                  `json:"error,omitempty"`
}

// PDFPageSize describes one page's dimensions.
type PDFPageSize struct {
	Number   int     `json:"number"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation int     `json:"rotation,omitempty"`
}

// Info reads a PDF file and returns basic document information.
func Info(inputPath string, password ...string) (*PDFInfo, error) {
	if inputPath == "" {
		return nil, errors.New("pdf: input path is empty")
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}
	info := &PDFInfo{Path: inputPath}
	if encrypted, _ := reader.IsEncrypted(data); encrypted {
		info.Encrypted = true
	}
	pass := ""
	if len(password) > 0 {
		pass = password[0]
	}
	r, err := reader.OpenWithPassword(data, pass)
	if err != nil {
		info.Error = err.Error()
		return info, err
	}
	info.Pages = r.NumPages()
	info.Metadata = r.Metadata()
	info.Outlines = r.Outlines()
	info.Annotations = r.Annotations()
	info.Trailer = stringifyKeys(r.Trailer())
	for i := 0; i < r.NumPages(); i++ {
		page, err := r.Page(i)
		if err != nil {
			continue
		}
		info.PageSizes = append(info.PageSizes, PDFPageSize{
			Number:   i + 1,
			Width:    page.MediaBox[2] - page.MediaBox[0],
			Height:   page.MediaBox[3] - page.MediaBox[1],
			Rotation: page.Rotation,
		})
	}
	return info, nil
}

func stringifyKeys(m map[string]interface{}) map[string]string {
	out := make(map[string]string)
	for k, v := range m {
		if k == "/Encrypt" {
			out["Encrypt"] = "present"
			continue
		}
		if len(k) > 0 && k[0] == '/' {
			k = k[1:]
		}
		switch val := v.(type) {
		case string:
			out[k] = val
		case int64, float64, bool:
			b, _ := json.Marshal(val)
			out[k] = string(b)
		default:
			out[k] = fmt.Sprintf("%T", v)
		}
	}
	return out
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

// Split writes selected pages from inputPath to outputPath. Pages are 0-based;
// nil or empty pages copies all pages.
func Split(inputPath, outputPath string, pages []int, opts ...converter.ConvertOptions) error {
	if inputPath == "" {
		return errors.New("pdf: input path is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	password := ""
	if len(opts) > 0 {
		password = opts[0].Password
	}
	return reader.ExtractPagesFile(inputPath, outputPath, pages, password)
}

// ExtractImages extracts images from a PDF file.
func ExtractImages(inputPath string, opts ...converter.ConvertOptions) ([]converter.ExtractedImage, error) {
	if inputPath == "" {
		return nil, errors.New("pdf: input path is empty")
	}
	var opt converter.ConvertOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	conv, err := converter.NewFromFile(inputPath, opt)
	if err != nil {
		return nil, err
	}
	return conv.ExtractImages()
}

// SetMetadata writes a copy of inputPath with a replacement info dictionary.
func SetMetadata(inputPath, outputPath string, metadata map[string]string, opts ...converter.ConvertOptions) error {
	if inputPath == "" {
		return errors.New("pdf: input path is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	password := ""
	if len(opts) > 0 {
		password = opts[0].Password
	}
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password: password,
		Info:     metadata,
	})
}

// Protect encrypts a PDF using AES-128 while copying its pages into a new file.
func Protect(inputPath, outputPath string, cfg core.EncryptionConfig, opts ...converter.ConvertOptions) error {
	if inputPath == "" {
		return errors.New("pdf: input path is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	if cfg.Algorithm == 0 {
		cfg.Algorithm = core.AES_128
	}
	if cfg.Permissions == 0 {
		cfg.Permissions = 0xFFFFF0C4
	}
	password := ""
	if len(opts) > 0 {
		password = opts[0].Password
	}
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password:   password,
		Encryption: &cfg,
	})
}

// Decrypt copies an encrypted PDF into a new unencrypted PDF.
func Decrypt(inputPath, outputPath string, opts ...converter.ConvertOptions) error {
	if inputPath == "" {
		return errors.New("pdf: input path is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	password := ""
	if len(opts) > 0 {
		password = opts[0].Password
	}
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{Password: password})
}

// Rotate writes a copy of inputPath with selected pages rotated to rotation degrees.
func Rotate(inputPath, outputPath string, pages []int, rotation int, opts ...converter.ConvertOptions) error {
	if inputPath == "" {
		return errors.New("pdf: input path is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	password := ""
	if len(opts) > 0 {
		password = opts[0].Password
	}
	rotate := make(map[int]int)
	for _, page := range pages {
		rotate[page] = rotation
	}
	if len(pages) == 0 {
		info, err := Info(inputPath, password)
		if err != nil {
			return err
		}
		for i := 0; i < info.Pages; i++ {
			rotate[i] = rotation
		}
	}
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password: password,
		Rotate:   rotate,
	})
}

// Reorder writes pages from inputPath to outputPath in the supplied 0-based order.
func Reorder(inputPath, outputPath string, pages []int, opts ...converter.ConvertOptions) error {
	return Split(inputPath, outputPath, pages, opts...)
}

// DeletePages writes a copy of inputPath without the supplied 0-based pages.
func DeletePages(inputPath, outputPath string, deletePages []int, opts ...converter.ConvertOptions) error {
	password := ""
	if len(opts) > 0 {
		password = opts[0].Password
	}
	info, err := Info(inputPath, password)
	if err != nil {
		return err
	}
	deleted := make(map[int]bool)
	for _, p := range deletePages {
		deleted[p] = true
	}
	var keep []int
	for i := 0; i < info.Pages; i++ {
		if !deleted[i] {
			keep = append(keep, i)
		}
	}
	if len(keep) == 0 {
		return errors.New("pdf: delete would remove all pages")
	}
	return Split(inputPath, outputPath, keep, opts...)
}

// WatermarkOptions controls text watermark placement.
type WatermarkOptions struct {
	Text     string
	FontSize float64
	Opacity  float64
	Angle    float64
	Color    [3]float64
	Pages    []int
	Password string
}

// Watermark adds a text watermark to selected pages.
func Watermark(inputPath, outputPath string, opts WatermarkOptions) error {
	if strings.TrimSpace(opts.Text) == "" {
		return errors.New("pdf: watermark text is empty")
	}
	pageSet := pagesToSet(opts.Pages)
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password: opts.Password,
		Overlay: func(pageIndex int, page *document.Page) []byte {
			if len(pageSet) > 0 && !pageSet[pageIndex] {
				return nil
			}
			return watermarkContent(page, opts)
		},
	})
}

// PageNumberOptions controls page number stamping.
type PageNumberOptions struct {
	Format   string
	FontSize float64
	Margin   float64
	Password string
}

// AddPageNumbers stamps page numbers onto every page.
func AddPageNumbers(inputPath, outputPath string, opts PageNumberOptions) error {
	info, err := Info(inputPath, opts.Password)
	if err != nil {
		return err
	}
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password: opts.Password,
		Overlay: func(pageIndex int, page *document.Page) []byte {
			format := opts.Format
			if format == "" {
				format = "Page %d of %d"
			}
			text := fmt.Sprintf(format, pageIndex+1, info.Pages)
			size := opts.FontSize
			if size <= 0 {
				size = 10
			}
			margin := opts.Margin
			if margin <= 0 {
				margin = 36
			}
			page.Fonts["FPN"] = 0
			x := page.Size.Width / 2
			y := margin
			return []byte(fmt.Sprintf("\nBT /FPN %.2f Tf %.3f %.3f Td (%s) Tj ET\n",
				size, x-float64(len(text))*size*0.22, y, escapePDFText(text)))
		},
	})
}

func pagesToSet(pages []int) map[int]bool {
	if len(pages) == 0 {
		return nil
	}
	set := make(map[int]bool)
	for _, page := range pages {
		set[page] = true
	}
	return set
}

func watermarkContent(page *document.Page, opts WatermarkOptions) []byte {
	size := opts.FontSize
	if size <= 0 {
		size = 48
	}
	color := opts.Color
	if color == [3]float64{} {
		color = [3]float64{0.5, 0.5, 0.5}
	}
	angle := opts.Angle
	if angle == 0 {
		angle = 35
	}
	rad := angle * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	x := page.Size.Width / 2
	y := page.Size.Height / 2
	page.Fonts["FWM"] = 0
	return []byte(fmt.Sprintf(
		"\nq %.3f %.3f %.3f rg %.5f %.5f %.5f %.5f %.3f %.3f cm BT /FWM %.2f Tf %.3f 0 Td (%s) Tj ET Q\n",
		color[0], color[1], color[2], cos, sin, -sin, cos, x, y, size, -float64(len(opts.Text))*size*0.25, escapePDFText(opts.Text),
	))
}

func escapePDFText(s string) string {
	var out strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '(', ')':
			out.WriteByte('\\')
			out.WriteRune(r)
		case '\n', '\r':
			out.WriteByte(' ')
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
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

// FromHTMLTemplateJSONFile renders an HTML template file with placeholder data
// loaded from a JSON object and converts the result to a PDF file.
func FromHTMLTemplateJSONFile(templatePath, jsonPath, outputPath string, opts ...html.Options) error {
	data, err := LoadTemplateJSON(jsonPath)
	if err != nil {
		return err
	}
	return FromHTMLTemplateFile(templatePath, data, outputPath, opts...)
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

// LoadTemplateJSON reads a JSON object for use with HTML/PDF placeholders.
func LoadTemplateJSON(jsonPath string) (map[string]any, error) {
	if jsonPath == "" {
		return nil, errors.New("pdf: JSON data path is empty")
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, fmt.Errorf("reading JSON data: %w", err)
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parsing JSON data: %w", err)
	}
	if values == nil {
		return nil, errors.New("pdf: JSON data must be an object")
	}
	return values, nil
}
