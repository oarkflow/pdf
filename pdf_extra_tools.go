package pdf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/oarkflow/pdf/converter"
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/forms"
	pdfimage "github.com/oarkflow/pdf/image"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/reader"
)

// FormFieldInfo describes one interactive PDF form field.
type FormFieldInfo = reader.AnnotationInfo

// ListFormFields returns AcroForm widget fields from a PDF.
func ListFormFields(inputPath, password string) ([]FormFieldInfo, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}
	r, err := reader.OpenWithPassword(data, password)
	if err != nil {
		return nil, err
	}
	return r.FormFields(), nil
}

// FillFormFile fills an existing PDF form from string values and writes a
// flattened filled PDF.
func FillFormFile(inputPath, outputPath string, data map[string]string) error {
	if inputPath == "" {
		return errors.New("pdf: input path is empty")
	}
	if outputPath == "" {
		return errors.New("pdf: output path is empty")
	}
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	out, err := forms.FillForm(input, forms.FormData(data))
	if err != nil {
		return err
	}
	return writeOutputFile(outputPath, out)
}

// FillFormJSONFile fills an existing PDF form from a JSON object.
func FillFormJSONFile(inputPath, jsonPath, outputPath string) error {
	if jsonPath == "" {
		return errors.New("pdf: JSON data path is empty")
	}
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}
	out, err := forms.FillFormJSON(input, data)
	if err != nil {
		return err
	}
	return writeOutputFile(outputPath, out)
}

// ImageStampOptions controls image or handwritten-signature placement.
type ImageStampOptions struct {
	ImagePath string
	Page      int
	Pages     []int
	X         float64
	Y         float64
	Width     float64
	Height    float64
	Password  string
}

// StampImage places an image on selected PDF pages.
func StampImage(inputPath, outputPath string, opts ImageStampOptions) error {
	if opts.ImagePath == "" {
		return errors.New("pdf: image path is empty")
	}
	data, err := os.ReadFile(opts.ImagePath)
	if err != nil {
		return err
	}
	img, err := pdfimage.Load(data)
	if err != nil {
		return err
	}
	pages := opts.Pages
	if opts.Page > 0 {
		pages = append(pages, opts.Page-1)
	}
	pageSet := pagesToSet(pages)
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password: opts.Password,
		Overlay: func(pageIndex int, page *document.Page) []byte {
			if len(pageSet) > 0 && !pageSet[pageIndex] {
				return nil
			}
			name := fmt.Sprintf("Stamp%d", pageIndex+1)
			w, h := opts.Width, opts.Height
			if w <= 0 && h <= 0 {
				w = float64(img.Width)
				h = float64(img.Height)
			} else if w <= 0 {
				w = h * float64(img.Width) / float64(img.Height)
			} else if h <= 0 {
				h = w * float64(img.Height) / float64(img.Width)
			}
			page.Images[name] = layout.ImageEntry{Image: img, Width: img.Width, Height: img.Height}
			return []byte(fmt.Sprintf("\nq %.3f 0 0 %.3f %.3f %.3f cm /%s Do Q\n", w, h, opts.X, opts.Y, name))
		},
	})
}

// SignatureImage places an uploaded signature image on selected PDF pages.
func SignatureImage(inputPath, outputPath string, opts ImageStampOptions) error {
	return StampImage(inputPath, outputPath, opts)
}

// CompressPDF rewrites a PDF through the writer, compressing page streams and
// normalizing copied resources.
func CompressPDF(inputPath, outputPath, password string) error {
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{Password: password})
}

// RedactionRegion describes a rectangular area to cover permanently.
type RedactionRegion struct {
	Page   int     `json:"page"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// RedactOptions controls text and region redaction.
type RedactOptions struct {
	Texts    []string
	Regions  []RedactionRegion
	Password string
}

// Redact removes matching literal text from copied page streams and covers
// configured regions with black rectangles.
func Redact(inputPath, outputPath string, opts RedactOptions) error {
	if len(opts.Texts) == 0 && len(opts.Regions) == 0 {
		return errors.New("pdf: redaction requires text or regions")
	}
	regionsByPage := make(map[int][]RedactionRegion)
	for _, region := range opts.Regions {
		if region.Page < 1 {
			return fmt.Errorf("pdf: redaction page must be 1-based")
		}
		regionsByPage[region.Page-1] = append(regionsByPage[region.Page-1], region)
	}
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password: opts.Password,
		ContentTransform: func(_ int, content []byte) []byte {
			return replaceContentStrings(content, opts.Texts, func(s, needle string) string {
				return strings.ReplaceAll(s, needle, strings.Repeat(" ", len([]rune(needle))))
			})
		},
		Overlay: func(pageIndex int, _ *document.Page) []byte {
			var out strings.Builder
			for _, region := range regionsByPage[pageIndex] {
				out.WriteString(fmt.Sprintf("\nq 0 0 0 rg %.3f %.3f %.3f %.3f re f Q\n", region.X, region.Y, region.Width, region.Height))
			}
			return []byte(out.String())
		},
	})
}

// PDFComparison describes a text/metadata comparison between two PDFs.
type PDFComparison struct {
	PagesA           int               `json:"pagesA"`
	PagesB           int               `json:"pagesB"`
	PageCountMatches bool              `json:"pageCountMatches"`
	TextMatches      bool              `json:"textMatches"`
	MetadataA        map[string]string `json:"metadataA,omitempty"`
	MetadataB        map[string]string `json:"metadataB,omitempty"`
	Differences      []string          `json:"differences,omitempty"`
}

// ComparePDF compares page count, extracted text, and metadata.
func ComparePDF(aPath, bPath string, opts ...converter.ConvertOptions) (PDFComparison, error) {
	password := ""
	if len(opts) > 0 {
		password = opts[0].Password
	}
	aInfo, err := Info(aPath, password)
	if err != nil {
		return PDFComparison{}, err
	}
	bInfo, err := Info(bPath, password)
	if err != nil {
		return PDFComparison{}, err
	}
	aText, err := ToText(aPath, converter.ConvertOptions{Password: password})
	if err != nil {
		return PDFComparison{}, err
	}
	bText, err := ToText(bPath, converter.ConvertOptions{Password: password})
	if err != nil {
		return PDFComparison{}, err
	}
	result := PDFComparison{
		PagesA:           aInfo.Pages,
		PagesB:           bInfo.Pages,
		PageCountMatches: aInfo.Pages == bInfo.Pages,
		TextMatches:      aText == bText,
		MetadataA:        aInfo.Metadata,
		MetadataB:        bInfo.Metadata,
	}
	if !result.PageCountMatches {
		result.Differences = append(result.Differences, fmt.Sprintf("page count differs: %d != %d", aInfo.Pages, bInfo.Pages))
	}
	if !result.TextMatches {
		result.Differences = append(result.Differences, "extracted text differs")
	}
	if !stringMapEqual(aInfo.Metadata, bInfo.Metadata) {
		result.Differences = append(result.Differences, "metadata differs")
	}
	return result, nil
}

// TranslateWithDictionary replaces literal text fragments from a JSON
// dictionary while copying the PDF.
func TranslateWithDictionary(inputPath, dictionaryPath, outputPath string, password string) error {
	data, err := os.ReadFile(dictionaryPath)
	if err != nil {
		return err
	}
	dict := make(map[string]string)
	if err := json.Unmarshal(data, &dict); err != nil {
		return fmt.Errorf("parsing translation dictionary: %w", err)
	}
	keys := make([]string, 0, len(dict))
	for key := range dict {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password: password,
		ContentTransform: func(_ int, content []byte) []byte {
			return replaceContentStrings(content, keys, func(s, needle string) string {
				return strings.ReplaceAll(s, needle, dict[needle])
			})
		},
	})
}

// PrepareForPrint writes a normalized print-preparation copy with optional
// metadata and page numbering.
func PrepareForPrint(inputPath, outputPath string, metadata map[string]string, addPageNumbers bool, password string) error {
	if !addPageNumbers {
		return reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{Password: password, Info: metadata})
	}
	tmp := filepath.Join(os.TempDir(), "pdf-print-prep-"+strconv.FormatInt(int64(os.Getpid()), 10)+".pdf")
	if err := reader.CopyPagesFile(inputPath, tmp, reader.CopyOptions{Password: password, Info: metadata}); err != nil {
		return err
	}
	defer os.Remove(tmp)
	return AddPageNumbers(tmp, outputPath, PageNumberOptions{Password: password})
}

// ArchivePDF writes a PDF/A-oriented copy and validates it with the built-in
// profile checks.
func ArchivePDF(inputPath, outputPath string, password string) (ComplianceReport, error) {
	if err := reader.CopyPagesFile(inputPath, outputPath, reader.CopyOptions{
		Password: password,
		Info: map[string]string{
			"Producer": "github.com/oarkflow/pdf",
		},
	}); err != nil {
		return ComplianceReport{}, err
	}
	return ValidateCompliance(outputPath, ComplianceOptions{Profiles: []ComplianceProfile{ProfilePDFA2b}, Password: password}), nil
}

// PDFGraph describes relationships discovered across PDFs.
type PDFGraph struct {
	Nodes []PDFGraphNode `json:"nodes"`
	Edges []PDFGraphEdge `json:"edges"`
}

type PDFGraphNode struct {
	ID       string            `json:"id"`
	Path     string            `json:"path"`
	Pages    int               `json:"pages"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type PDFGraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
	Page int    `json:"page,omitempty"`
}

// BuildPDFGraph extracts file, link, and metadata relationships.
func BuildPDFGraph(paths []string, password string) (PDFGraph, error) {
	var graph PDFGraph
	known := make(map[string]string)
	for _, path := range paths {
		info, err := Info(path, password)
		if err != nil {
			return graph, err
		}
		id := filepath.Base(path)
		known[path] = id
		known[filepath.Base(path)] = id
		graph.Nodes = append(graph.Nodes, PDFGraphNode{ID: id, Path: path, Pages: info.Pages, Metadata: info.Metadata})
		for _, annot := range info.Annotations {
			if annot.URI == "" {
				continue
			}
			graph.Edges = append(graph.Edges, PDFGraphEdge{From: id, To: annot.URI, Type: "link", Page: annot.Page})
		}
	}
	for i := range graph.Edges {
		if target, ok := known[graph.Edges[i].To]; ok {
			graph.Edges[i].To = target
			graph.Edges[i].Type = "file-link"
		}
	}
	return graph, nil
}

// ScanImagesToPDF converts scanned image files into a single PDF.
func ScanImagesToPDF(opts ImagePDFOptions) error {
	if opts.Page.ImageFit == "" {
		opts.Page.ImageFit = "contain"
	}
	return ImagesToPDF(opts)
}

func replaceContentStrings(content []byte, needles []string, replace func(string, string) string) []byte {
	if len(needles) == 0 {
		return content
	}
	var buf bytes.Buffer
	tok := reader.NewTokenizer(content)
	for {
		t, err := tok.Next()
		if err != nil || t.Type == reader.TokenEOF {
			break
		}
		switch t.Type {
		case reader.TokenString:
			value := t.Value
			for _, needle := range needles {
				if needle == "" {
					continue
				}
				value = replace(value, needle)
			}
			buf.WriteByte('(')
			buf.WriteString(escapePDFText(value))
			buf.WriteByte(')')
		case reader.TokenHexString:
			buf.WriteByte('<')
			buf.WriteString(t.Value)
			buf.WriteByte('>')
		case reader.TokenName:
			buf.WriteByte('/')
			buf.WriteString(t.Value)
		case reader.TokenInteger:
			buf.WriteString(strconv.FormatInt(t.Int, 10))
		default:
			buf.WriteString(t.Value)
		}
		buf.WriteByte(' ')
	}
	if buf.Len() == 0 {
		return content
	}
	return buf.Bytes()
}

func stringMapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, av := range a {
		if b[key] != av {
			return false
		}
	}
	return true
}
