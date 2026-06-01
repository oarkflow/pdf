package reader

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/oarkflow/pdf/core"
	"github.com/oarkflow/pdf/document"
	"github.com/oarkflow/pdf/layout"
)

// PageOverlay is extra content appended to a copied page.
type PageOverlay func(pageIndex int, page *document.Page) []byte

// CopyOptions controls page copying operations.
type CopyOptions struct {
	Pages      []int
	Password   string
	Encryption *core.EncryptionConfig
	Info       map[string]string
	Rotate     map[int]int
	Overlay    PageOverlay
}

// Merge combines multiple PDF byte slices into a single PDF.
func Merge(pdfs [][]byte) ([]byte, error) {
	if len(pdfs) == 0 {
		return nil, fmt.Errorf("no PDFs to merge")
	}

	w := document.NewWriter()

	for docIdx, pdfData := range pdfs {
		reader, err := Open(pdfData)
		if err != nil {
			return nil, fmt.Errorf("opening PDF %d: %w", docIdx, err)
		}

		for pageNum := 0; pageNum < reader.NumPages(); pageNum++ {
			page, err := reader.Page(pageNum)
			if err != nil {
				return nil, fmt.Errorf("reading page %d of PDF %d: %w", pageNum, docIdx, err)
			}

			suffix := fmt.Sprintf("_doc%d", docIdx)
			newPage, fontObjNums, imageObjNums, err := buildMergePage(w, page, suffix, reader)
			if err != nil {
				return nil, fmt.Errorf("building merge page %d of PDF %d: %w", pageNum, docIdx, err)
			}

			newPage.Fonts = fontObjNums
			newPage.Images = imageObjNums

			if _, err := w.AddPage(newPage); err != nil {
				return nil, fmt.Errorf("adding page %d of PDF %d: %w", pageNum, docIdx, err)
			}
		}
	}

	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("writing merged PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// MergeFiles merges PDF files at the given paths into outputPath.
func MergeFiles(paths []string, outputPath string) error {
	var pdfs [][]byte
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		pdfs = append(pdfs, data)
	}

	merged, err := Merge(pdfs)
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, merged, 0644)
}

// ExtractPages copies selected 0-based pages from one PDF into a new PDF.
func ExtractPages(pdfData []byte, pages []int, password string) ([]byte, error) {
	return CopyPages(pdfData, CopyOptions{Pages: pages, Password: password})
}

// CopyPages copies selected 0-based pages into a new PDF, optionally modifying
// rotation, appending page overlays, and applying output encryption.
func CopyPages(pdfData []byte, opts CopyOptions) ([]byte, error) {
	reader, err := OpenWithPassword(pdfData, opts.Password)
	if err != nil {
		return nil, err
	}
	pages := opts.Pages
	if len(pages) == 0 {
		pages = make([]int, reader.NumPages())
		for i := range pages {
			pages[i] = i
		}
	}

	w := document.NewWriter()

	for _, pageNum := range pages {
		if pageNum < 0 || pageNum >= reader.NumPages() {
			return nil, fmt.Errorf("page %d out of range [1, %d]", pageNum+1, reader.NumPages())
		}
		page, err := reader.Page(pageNum)
		if err != nil {
			return nil, fmt.Errorf("reading page %d: %w", pageNum+1, err)
		}
		suffix := fmt.Sprintf("_p%d", pageNum+1)
		newPage, fontObjNums, imageObjNums, err := buildMergePage(w, page, suffix, reader)
		if err != nil {
			return nil, fmt.Errorf("building page %d: %w", pageNum+1, err)
		}
		newPage.Fonts = fontObjNums
		newPage.Images = imageObjNums
		newPage.Rotation = normalizeRotation(page.Rotation)
		if rotate, ok := opts.Rotate[pageNum]; ok {
			newPage.Rotation = normalizeRotation(rotate)
		}
		if opts.Overlay != nil {
			newPage.Contents = append(newPage.Contents, opts.Overlay(pageNum, newPage)...)
		}
		if _, err := w.AddPage(newPage); err != nil {
			return nil, fmt.Errorf("adding page %d: %w", pageNum+1, err)
		}
	}

	if opts.Encryption != nil {
		if err := document.ApplyEncryption(w, *opts.Encryption); err != nil {
			return nil, err
		}
	}
	if len(opts.Info) > 0 {
		w.SetInfo(opts.Info)
	}

	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("writing extracted PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// ExtractPagesFile copies selected pages from inputPath into outputPath.
func ExtractPagesFile(inputPath, outputPath string, pages []int, password string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inputPath, err)
	}
	extracted, err := ExtractPages(data, pages, password)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, extracted, 0644)
}

// CopyPagesFile copies pages from inputPath to outputPath with the given options.
func CopyPagesFile(inputPath, outputPath string, opts CopyOptions) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", inputPath, err)
	}
	out, err := CopyPages(data, opts)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, out, 0644)
}

func buildMergePage(w *document.Writer, page *PageInfo, suffix string, reader *Reader) (*document.Page, map[string]int, map[string]layout.ImageEntry, error) {
	newPage := document.NewPage(document.PageSize{
		Width:  page.MediaBox[2] - page.MediaBox[0],
		Height: page.MediaBox[3] - page.MediaBox[1],
	})
	newPage.Rotation = normalizeRotation(page.Rotation)

	// Rewrite content stream to rename font/image references.
	contents := rewriteResourceNames(page.Contents, suffix)
	newPage.Contents = contents

	fontObjNums := make(map[string]int)
	imageObjNums := make(map[string]layout.ImageEntry)

	// Add fonts from resources.
	if fontRes, ok := page.Resources["/Font"]; ok {
		fontMap := resolveMap(reader.resolver, fontRes)
		for name, fontRef := range fontMap {
			newName := name + suffix
			// Strip leading / if present.
			cleanName := name
			if len(cleanName) > 0 && cleanName[0] == '/' {
				cleanName = cleanName[1:]
			}
			cleanNew := cleanName + suffix

			// Create a basic font object.
			fontDict := resolveMap(reader.resolver, fontRef)
			if fontDict == nil {
				continue
			}
			coreDict := mapToCoreDictionary(fontDict)
			objNum := w.AddObject(coreDict)
			fontObjNums[cleanNew] = objNum
			_ = newName
		}
	}

	// Add XObjects (images) from resources.
	if xobjRes, ok := page.Resources["/XObject"]; ok {
		xobjMap := resolveMap(reader.resolver, xobjRes)
		for name, xobjRef := range xobjMap {
			cleanName := name
			if len(cleanName) > 0 && cleanName[0] == '/' {
				cleanName = cleanName[1:]
			}
			cleanNew := cleanName + suffix

			xobj, err := reader.resolver.ResolveReference(xobjRef)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("reader: merge: failed to resolve XObject %s: %w", name, err)
			}
			if so, ok := xobj.(*StreamObject); ok {
				data, err := reader.resolver.DecompressStream(so.Dict, so.Data)
				if err != nil {
					return nil, nil, nil, fmt.Errorf("reader: merge: failed to decompress XObject %s: %w", name, err)
				}
				stream := core.NewStream(data)
				if err := stream.Compress(); err != nil {
					return nil, nil, nil, fmt.Errorf("reader: merge: failed to compress XObject %s: %w", name, err)
				}
				// Copy relevant dictionary entries.
				for k, v := range so.Dict {
					if k == "/Length" || k == "/Filter" {
						continue
					}
					cleanKey := k
					if len(cleanKey) > 0 && cleanKey[0] == '/' {
						cleanKey = cleanKey[1:]
					}
					stream.Dictionary.Set(cleanKey, convertToCoreObject(v))
				}
				objNum := w.AddObject(stream)
				imageObjNums[cleanNew] = layout.ImageEntry{ObjectNum: objNum}
			}
		}
	}

	return newPage, fontObjNums, imageObjNums, nil
}

func normalizeRotation(rotation int) int {
	rotation %= 360
	if rotation < 0 {
		rotation += 360
	}
	switch rotation {
	case 90, 180, 270:
		return rotation
	default:
		return 0
	}
}

// rewriteResourceNames appends suffix to font and image name references in a content stream.
func rewriteResourceNames(contents []byte, suffix string) []byte {
	if len(suffix) == 0 {
		return contents
	}

	var buf bytes.Buffer
	tok := NewTokenizer(contents)

	for {
		t, err := tok.Next()
		if err != nil || t.Type == TokenEOF {
			break
		}

		switch t.Type {
		case TokenName:
			buf.WriteByte('/')
			buf.WriteString(t.Value)
			buf.WriteString(suffix)
		case TokenInteger:
			buf.WriteString(strconv.FormatInt(t.Int, 10))
		case TokenReal:
			buf.WriteString(t.Value)
		case TokenString:
			buf.WriteByte('(')
			buf.WriteString(escapeStringContent(t.Value))
			buf.WriteByte(')')
		case TokenHexString:
			buf.WriteByte('<')
			buf.WriteString(t.Value)
			buf.WriteByte('>')
		case TokenKeyword:
			buf.WriteString(t.Value)
		case TokenArrayBegin:
			buf.WriteByte('[')
		case TokenArrayEnd:
			buf.WriteByte(']')
		case TokenDictBegin:
			buf.WriteString("<<")
		case TokenDictEnd:
			buf.WriteString(">>")
		default:
			buf.WriteString(t.Value)
		}
		buf.WriteByte(' ')
	}

	return buf.Bytes()
}

func escapeStringContent(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '(':
			buf.WriteString("\\(")
		case ')':
			buf.WriteString("\\)")
		case '\\':
			buf.WriteString("\\\\")
		default:
			buf.WriteByte(c)
		}
	}
	return buf.String()
}

func resolveMap(resolver *Resolver, v interface{}) map[string]interface{} {
	v, _ = resolver.ResolveReference(v)
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func mapToCoreDictionary(m map[string]interface{}) *core.PdfDictionary {
	d := core.NewDictionary()
	for k, v := range m {
		key := k
		if len(key) > 0 && key[0] == '/' {
			key = key[1:]
		}
		d.Set(key, convertToCoreObject(v))
	}
	return d
}

func convertToCoreObject(v interface{}) core.PdfObject {
	switch val := v.(type) {
	case bool:
		return core.PdfBoolean(val)
	case int64:
		return core.PdfInteger(val)
	case float64:
		return core.PdfNumber(val)
	case string:
		if len(val) > 0 && val[0] == '/' {
			return core.PdfName(val[1:])
		}
		return core.PdfString(val)
	case []interface{}:
		arr := make(core.PdfArray, len(val))
		for i, item := range val {
			arr[i] = convertToCoreObject(item)
		}
		return arr
	case map[string]interface{}:
		return mapToCoreDictionary(val)
	case nil:
		return core.PdfNull{}
	default:
		return core.PdfNull{}
	}
}
