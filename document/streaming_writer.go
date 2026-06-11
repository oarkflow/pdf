package document

import (
	"fmt"
	"io"
	"strconv"

	"github.com/oarkflow/pdf/core"
	pdffont "github.com/oarkflow/pdf/font"
	pdfimage "github.com/oarkflow/pdf/image"
	"github.com/oarkflow/pdf/layout"
)

// StreamingWriter writes PDF objects directly to an io.Writer as they are added,
// only keeping metadata (offsets, page refs) in memory. This avoids buffering
// the entire PDF in RAM, which is important for large documents.
//
// StreamingWriter is not safe for concurrent use. Callers must synchronize access externally.
type StreamingWriter struct {
	counter  *countingWriter
	offsets  map[int]int64 // object number -> byte offset
	pages    []int         // page object numbers
	nextObj  int
	pagesObj int
	fontObjs map[string]int
	info     *core.PdfDictionary
	finished bool
}

func (sw *StreamingWriter) writeInt(v int) error {
	var buf [24]byte
	_, err := sw.counter.Write(strconv.AppendInt(buf[:0], int64(v), 10))
	return err
}

func (sw *StreamingWriter) writeInt64(v int64) error {
	var buf [24]byte
	_, err := sw.counter.Write(strconv.AppendInt(buf[:0], v, 10))
	return err
}

func (sw *StreamingWriter) writeXrefOffset(offset int64) error {
	var buf [20]byte
	for i := 0; i < 10; i++ {
		buf[i] = '0'
	}
	b := strconv.AppendInt(buf[:10], offset, 10)
	if extra := len(b) - 10; extra > 0 {
		copy(buf[:], b[extra:])
	} else if extra < 0 {
		copy(buf[-extra:], b)
	}
	copy(buf[10:], " 00000 n \r\n")
	_, err := sw.counter.Write(buf[:20])
	return err
}

// NewStreamingWriter creates a StreamingWriter that writes PDF output directly
// to out. The PDF header is written immediately.
func NewStreamingWriter(out io.Writer) (*StreamingWriter, error) {
	cw := &countingWriter{w: out}
	sw := &StreamingWriter{
		counter:  cw,
		offsets:  make(map[int]int64, 16),
		fontObjs: make(map[string]int, 4),
		pages:    make([]int, 0, 4),
		nextObj:  1,
	}
	sw.pagesObj = sw.allocObj()

	// Write PDF header immediately.
	if _, err := io.WriteString(cw, "%PDF-1.7\n"); err != nil {
		return nil, err
	}
	// Binary comment to signal binary content.
	if _, err := cw.Write([]byte{'%', 0xE2, 0xE3, 0xCF, 0xD3, '\n'}); err != nil {
		return nil, err
	}

	return sw, nil
}

// allocObj reserves the next object number without writing anything.
func (sw *StreamingWriter) allocObj() int {
	num := sw.nextObj
	sw.nextObj++
	return num
}

// writeIndirectObject writes a single indirect object to the stream and records its offset.
func (sw *StreamingWriter) writeIndirectObject(num int, obj core.PdfObject) error {
	sw.offsets[num] = sw.counter.written
	ind := &core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: num, GenerationNumber: 0},
		Object:    obj,
	}
	_, err := ind.WriteTo(sw.counter)
	return err
}

// AddObject writes the object immediately to the output stream and returns
// the assigned object number.
func (sw *StreamingWriter) AddObject(obj core.PdfObject) (int, error) {
	num := sw.allocObj()
	if err := sw.writeIndirectObject(num, obj); err != nil {
		return 0, err
	}
	return num, nil
}

// ensureStandardFont creates a Type1 font object for the given standard font
// name if it doesn't already exist, and returns its object number.
func (sw *StreamingWriter) ensureStandardFont(name string) (int, error) {
	if num, ok := sw.fontObjs[name]; ok {
		return num, nil
	}
	d := core.NewDictionaryCap(4)
	d.Set("Type", core.PdfName("Font"))
	d.Set("Subtype", core.PdfName("Type1"))
	d.Set("BaseFont", core.PdfName(name))
	d.Set("Encoding", core.PdfName("WinAnsiEncoding"))
	num, err := sw.AddObject(d)
	if err != nil {
		return 0, err
	}
	sw.fontObjs[name] = num
	return num, nil
}

func (sw *StreamingWriter) addEmbeddedFont(entry layout.FontEntry) (int, error) {
	ef := entry.Embedded
	if ef == nil && entry.Face != nil {
		ef = pdffont.NewEmbeddedFont(entry.Face, entry.PDFName)
	}
	if ef == nil {
		return 0, nil
	}

	objects := ef.BuildObjects(sw.allocObj)
	if len(objects) == 0 {
		return 0, nil
	}
	for _, obj := range objects {
		if err := sw.writeIndirectObject(obj.Reference.ObjectNumber, obj.Object); err != nil {
			return 0, err
		}
	}
	return objects[len(objects)-1].Reference.ObjectNumber, nil
}

// addImageObject writes an image XObject immediately and returns its object number.
func (sw *StreamingWriter) addImageObject(img *pdfimage.Image) (int, error) {
	if img == nil {
		return 0, nil
	}

	mainNum := sw.allocObj()
	smaskNum := 0
	if len(img.AlphaData) > 0 {
		smaskNum = sw.allocObj()
	}

	mainObj, smaskObj, err := img.BuildXObject(mainNum, smaskNum)
	if err != nil {
		return 0, err
	}

	sw.offsets[mainNum] = sw.counter.written
	if _, err := mainObj.WriteTo(sw.counter); err != nil {
		return 0, err
	}

	if smaskObj != nil {
		sw.offsets[smaskNum] = sw.counter.written
		if _, err := smaskObj.WriteTo(sw.counter); err != nil {
			return 0, err
		}
	}

	return mainNum, nil
}

// AddPage writes a page's content stream and page dictionary immediately to
// the output. The page's Parent reference will point to object number that is
// written during Finish(). Returns the page object number.
func (sw *StreamingWriter) AddPage(page *Page) (int, error) {
	// Write content stream.
	contentData, err := page.CompressedContents()
	if err != nil {
		return 0, err
	}
	stream := core.NewStream(contentData)
	stream.Dictionary.Set("Filter", core.PdfName("FlateDecode"))
	stream.Dictionary.Set("Length", core.PdfInteger(len(contentData)))
	contentsNum, err := sw.AddObject(stream)
	if err != nil {
		return 0, err
	}

	// Build resource dict.
	res := core.NewDictionaryCap(3)
	if page.Resources != nil {
		for _, k := range page.Resources.Keys() {
			res.Set(k, page.Resources.Get(k))
		}
	}

	// Fonts.
	if len(page.FontEntries) > 0 || len(page.Fonts) > 0 {
		fontDict := core.NewDictionaryCap(len(page.FontEntries) + len(page.Fonts))
		for resourceName, entry := range page.FontEntries {
			objNum := entry.ObjectNum
			if entry.Embedded != nil || (entry.Face != nil && !pdffont.IsStandardFont(entry.Face.PostScriptName())) {
				objNum, err = sw.addEmbeddedFont(entry)
				if err != nil {
					return 0, err
				}
			} else if objNum == 0 {
				fontName := resourceName
				if entry.Name != "" {
					fontName = entry.Name
				}
				if entry.Face != nil && pdffont.IsStandardFont(entry.Face.PostScriptName()) {
					fontName = entry.Face.PostScriptName()
				}
				objNum, err = sw.ensureStandardFont(fontName)
				if err != nil {
					return 0, err
				}
			}
			if objNum == 0 {
				continue
			}
			fontDict.Set(resourceName, core.PdfIndirectReference{ObjectNumber: objNum})
		}
		for name, objNum := range page.Fonts {
			if page.FontEntries[name].PDFName != "" {
				continue
			}
			if objNum == 0 {
				objNum, err = sw.ensureStandardFont(name)
				if err != nil {
					return 0, err
				}
			}
			fontDict.Set(name, core.PdfIndirectReference{ObjectNumber: objNum})
		}
		res.Set("Font", fontDict)
	}

	// Images.
	if len(page.Images) > 0 {
		xobjDict := core.NewDictionaryCap(len(page.Images))
		for name, entry := range page.Images {
			objNum := entry.ObjectNum
			if objNum == 0 && entry.Image != nil {
				objNum, err = sw.addImageObject(entry.Image)
				if err != nil {
					return 0, err
				}
			}
			if objNum == 0 {
				continue
			}
			xobjDict.Set(name, core.PdfIndirectReference{ObjectNumber: objNum})
		}
		res.Set("XObject", xobjDict)
	}

	pageDict := core.NewDictionaryCap(6)
	pageDict.Set("Type", core.PdfName("Page"))
	pageDict.Set("Parent", core.PdfIndirectReference{ObjectNumber: sw.pagesObj})
	pageDict.Set("MediaBox", core.PdfArray{
		core.PdfNumber(0), core.PdfNumber(0),
		core.PdfNumber(page.Size.Width), core.PdfNumber(page.Size.Height),
	})
	pageDict.Set("Contents", core.PdfIndirectReference{ObjectNumber: contentsNum})
	pageDict.Set("Resources", res)
	if page.Rotation != 0 {
		pageDict.Set("Rotate", core.PdfInteger(int64(page.Rotation)))
	}

	// Add link annotations
	if len(page.Annotations) > 0 {
		var annotRefs core.PdfArray
		for _, link := range page.Annotations {
			annotDict := core.NewDictionaryCap(5)
			annotDict.Set("Type", core.PdfName("Annot"))
			annotDict.Set("Subtype", core.PdfName("Link"))
			annotDict.Set("Rect", core.PdfArray{
				core.PdfNumber(link.X1), core.PdfNumber(link.Y1),
				core.PdfNumber(link.X2), core.PdfNumber(link.Y2),
			})
			annotDict.Set("Border", core.PdfArray{
				core.PdfNumber(0), core.PdfNumber(0), core.PdfNumber(0),
			})
			actionDict := core.NewDictionaryCap(2)
			actionDict.Set("S", core.PdfName("URI"))
			actionDict.Set("URI", core.PdfString(link.URI))
			annotDict.Set("A", actionDict)
			annotNum, annotErr := sw.AddObject(annotDict)
			if annotErr != nil {
				return 0, annotErr
			}
			annotRefs = append(annotRefs, core.PdfIndirectReference{ObjectNumber: annotNum})
		}
		pageDict.Set("Annots", annotRefs)
	}

	num, err := sw.AddObject(pageDict)
	if err != nil {
		return 0, err
	}
	sw.pages = append(sw.pages, num)
	return num, nil
}

// SetInfo sets document metadata, which will be written during Finish().
func (sw *StreamingWriter) SetInfo(info map[string]string) {
	d := core.NewDictionaryCap(len(info))
	for k, v := range info {
		d.Set(k, core.PdfString(v))
	}
	sw.info = d
}

// SetMetadata sets document metadata without requiring callers to allocate a map.
func (sw *StreamingWriter) SetMetadata(meta Metadata) {
	count := 0
	if meta.Title != "" {
		count++
	}
	if meta.Author != "" {
		count++
	}
	if meta.Subject != "" {
		count++
	}
	if meta.Keywords != "" {
		count++
	}
	if meta.Creator != "" {
		count++
	}
	if meta.Producer != "" {
		count++
	}
	if count == 0 {
		sw.info = nil
		return
	}
	d := core.NewDictionaryCap(count)
	if meta.Title != "" {
		d.Set("Title", core.PdfString(meta.Title))
	}
	if meta.Author != "" {
		d.Set("Author", core.PdfString(meta.Author))
	}
	if meta.Subject != "" {
		d.Set("Subject", core.PdfString(meta.Subject))
	}
	if meta.Keywords != "" {
		d.Set("Keywords", core.PdfString(meta.Keywords))
	}
	if meta.Creator != "" {
		d.Set("Creator", core.PdfString(meta.Creator))
	}
	if meta.Producer != "" {
		d.Set("Producer", core.PdfString(meta.Producer))
	}
	sw.info = d
}

// Finish writes the pages tree, catalog, xref table, and trailer. It must be
// called exactly once after all pages have been added.
func (sw *StreamingWriter) Finish() error {
	if sw.finished {
		return fmt.Errorf("streaming writer: Finish already called")
	}
	sw.finished = true

	// Write Pages tree object. Its object number was reserved before pages
	// were streamed so page dictionaries can point to a strict Parent ref.
	kids := make(core.PdfArray, len(sw.pages))
	for i, pn := range sw.pages {
		kids[i] = core.PdfIndirectReference{ObjectNumber: pn}
	}
	pagesDict := core.NewDictionaryCap(3)
	pagesDict.Set("Type", core.PdfName("Pages"))
	pagesDict.Set("Kids", kids)
	pagesDict.Set("Count", core.PdfInteger(len(sw.pages)))
	if err := sw.writeIndirectObject(sw.pagesObj, pagesDict); err != nil {
		return err
	}

	// Build catalog.
	catalog := core.NewDictionaryCap(2)
	catalog.Set("Type", core.PdfName("Catalog"))
	catalog.Set("Pages", core.PdfIndirectReference{ObjectNumber: sw.pagesObj})
	catalogNum, err := sw.AddObject(catalog)
	if err != nil {
		return err
	}

	// Info object.
	var infoNum int
	if sw.info != nil {
		infoNum, err = sw.AddObject(sw.info)
		if err != nil {
			return err
		}
	}

	// Write xref table.
	xrefOffset := sw.counter.written
	totalObjects := sw.nextObj
	if _, err := io.WriteString(sw.counter, "xref\n0 "); err != nil {
		return err
	}
	if err := sw.writeInt(totalObjects); err != nil {
		return err
	}
	if _, err := io.WriteString(sw.counter, "\n0000000000 65535 f \r\n"); err != nil {
		return err
	}
	for num := 1; num < totalObjects; num++ {
		if err := sw.writeXrefOffset(sw.offsets[num]); err != nil {
			return err
		}
	}

	// Write trailer.
	if _, err := io.WriteString(sw.counter, "trailer\n"); err != nil {
		return err
	}
	trailer := core.NewDictionaryCap(3)
	trailer.Set("Size", core.PdfInteger(totalObjects))
	trailer.Set("Root", core.PdfIndirectReference{ObjectNumber: catalogNum})
	if infoNum > 0 {
		trailer.Set("Info", core.PdfIndirectReference{ObjectNumber: infoNum})
	}
	if _, err := trailer.WriteTo(sw.counter); err != nil {
		return err
	}
	if _, err := io.WriteString(sw.counter, "\nstartxref\n"); err != nil {
		return err
	}
	if err := sw.writeInt64(xrefOffset); err != nil {
		return err
	}
	if _, err := io.WriteString(sw.counter, "\n%%EOF\n"); err != nil {
		return err
	}

	return nil
}
