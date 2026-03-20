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
			newPage, fontObjNums, imageObjNums := buildMergePage(w, page, suffix, reader)

			newPage.Fonts = fontObjNums
			newPage.Images = imageObjNums

			w.AddPage(newPage)
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

func buildMergePage(w *document.Writer, page *PageInfo, suffix string, reader *Reader) (*document.Page, map[string]int, map[string]layout.ImageEntry) {
	newPage := document.NewPage(document.PageSize{
		Width:  page.MediaBox[2] - page.MediaBox[0],
		Height: page.MediaBox[3] - page.MediaBox[1],
	})

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

			xobj, _ := reader.resolver.ResolveReference(xobjRef)
			if so, ok := xobj.(*StreamObject); ok {
				data, _ := reader.resolver.DecompressStream(so.Dict, so.Data)
				stream := core.NewStream(data)
				_ = stream.Compress()
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

	return newPage, fontObjNums, imageObjNums
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
