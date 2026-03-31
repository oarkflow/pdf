package document

import (
	"bytes"
	"fmt"
	"io"

	"github.com/oarkflow/pdf/core"
	pdffont "github.com/oarkflow/pdf/font"
	pdfimage "github.com/oarkflow/pdf/image"
	"github.com/oarkflow/pdf/layout"
)

// Writer serializes PDF objects into a complete PDF file.
//
// Writer is not safe for concurrent use. Callers must synchronize access externally.
type Writer struct {
	objects  []core.PdfIndirectObject
	pages    []int // object numbers of page objects
	info     *core.PdfDictionary
	nextObj  int
	fontObjs map[string]int // font name -> object number (cached)

	// PDF/A fields
	metadataRef   int   // object number for /Metadata on catalog
	outputIntents []int // object numbers for /OutputIntents array
	docIDPair     [2]core.PdfHexString
	hasDocID      bool

	// Encryption fields
	encryptKey    []byte
	encryptAlgo   core.EncryptionAlgorithm
	encryptObjNum int
	documentID    []byte

	// Outline fields
	outlinesNum int

	// Tagged PDF fields
	structTreeRoot int
	markInfo       bool
}

// NewWriter creates a new Writer ready to accept objects.
func NewWriter() *Writer {
	return &Writer{
		nextObj:  1,
		fontObjs: make(map[string]int),
	}
}

// AddObject assigns the next object number to obj, stores it, and returns the
// object number.
func (w *Writer) AddObject(obj core.PdfObject) int {
	num := w.nextObj
	w.nextObj++
	w.objects = append(w.objects, core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: num, GenerationNumber: 0},
		Object:    obj,
	})
	return num
}

// ensureStandardFont creates a Type1 font object for the given standard font
// name if it doesn't already exist, and returns its object number.
func (w *Writer) ensureStandardFont(name string) int {
	if num, ok := w.fontObjs[name]; ok {
		return num
	}
	d := core.NewDictionary()
	d.Set("Type", core.PdfName("Font"))
	d.Set("Subtype", core.PdfName("Type1"))
	d.Set("BaseFont", core.PdfName(name))
	d.Set("Encoding", core.PdfName("WinAnsiEncoding"))
	num := w.AddObject(d)
	w.fontObjs[name] = num
	return num
}

func (w *Writer) addEmbeddedFont(entry layout.FontEntry) int {
	ef := entry.Embedded
	if ef == nil && entry.Face != nil {
		ef = pdffont.NewEmbeddedFont(entry.Face, entry.PDFName)
	}
	if ef == nil {
		return 0
	}

	objects := ef.BuildObjects(func() int {
		return w.ReserveObject()
	})
	if len(objects) == 0 {
		return 0
	}
	for _, obj := range objects {
		w.FillReserved(obj.Reference.ObjectNumber, obj.Object)
	}
	return objects[len(objects)-1].Reference.ObjectNumber
}

// AddPage creates a page indirect object from the given Page and records it for
// the pages tree. The parent reference is patched later during WriteTo.
// Returns the object number of the page object.
func (w *Writer) AddPage(page *Page) (int, error) {
	// Build content stream.
	stream := core.NewStream(page.Contents)
	if err := stream.Compress(); err != nil {
		return 0, fmt.Errorf("compressing page content: %w", err)
	}
	contentsNum := w.AddObject(stream)

	// Build resource dict
	res := core.NewDictionary()
	if page.Resources != nil {
		for _, k := range page.Resources.Keys() {
			res.Set(k, page.Resources.Get(k))
		}
	}

	// Handle fonts: embedded fonts are page-local so glyph subsetting stays correct.
	if len(page.FontEntries) > 0 || len(page.Fonts) > 0 {
		fontDict := core.NewDictionary()
		for resourceName, entry := range page.FontEntries {
			objNum := entry.ObjectNum
			if entry.Embedded != nil || (entry.Face != nil && !pdffont.IsStandardFont(entry.Face.PostScriptName())) {
				objNum = w.addEmbeddedFont(entry)
			} else if objNum == 0 {
				fontName := resourceName
				if entry.Name != "" {
					fontName = entry.Name
				}
				if entry.Face != nil && pdffont.IsStandardFont(entry.Face.PostScriptName()) {
					fontName = entry.Face.PostScriptName()
				}
				fontObjNum := w.ensureStandardFont(fontName)
				objNum = fontObjNum
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
				objNum = w.ensureStandardFont(name)
			}
			fontDict.Set(name, core.PdfIndirectReference{ObjectNumber: objNum})
		}
		res.Set("Font", fontDict)
	}
	if len(page.Images) > 0 {
		xobjDict := core.NewDictionary()
		for name, entry := range page.Images {
			objNum := entry.ObjectNum
			if objNum == 0 && entry.Image != nil {
				var err error
				objNum, err = w.addImageObject(entry.Image)
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

	pageDict := core.NewDictionary()
	pageDict.Set("Type", core.PdfName("Page"))
	// Parent will be patched in WriteTo.
	pageDict.Set("MediaBox", core.PdfArray{
		core.PdfNumber(0), core.PdfNumber(0),
		core.PdfNumber(page.Size.Width), core.PdfNumber(page.Size.Height),
	})
	pageDict.Set("Contents", core.PdfIndirectReference{ObjectNumber: contentsNum})
	pageDict.Set("Resources", res)

	// Add link annotations
	if len(page.Annotations) > 0 {
		var annotRefs core.PdfArray
		for _, link := range page.Annotations {
			annotDict := core.NewDictionary()
			annotDict.Set("Type", core.PdfName("Annot"))
			annotDict.Set("Subtype", core.PdfName("Link"))
			annotDict.Set("Rect", core.PdfArray{
				core.PdfNumber(link.X1), core.PdfNumber(link.Y1),
				core.PdfNumber(link.X2), core.PdfNumber(link.Y2),
			})
			annotDict.Set("Border", core.PdfArray{
				core.PdfNumber(0), core.PdfNumber(0), core.PdfNumber(0),
			})
			actionDict := core.NewDictionary()
			actionDict.Set("S", core.PdfName("URI"))
			actionDict.Set("URI", core.PdfString(link.URI))
			annotDict.Set("A", actionDict)
			annotNum := w.AddObject(annotDict)
			annotRefs = append(annotRefs, core.PdfIndirectReference{ObjectNumber: annotNum})
		}
		pageDict.Set("Annots", annotRefs)
	}

	num := w.AddObject(pageDict)
	w.pages = append(w.pages, num)
	return num, nil
}

func (w *Writer) addImageObject(img *pdfimage.Image) (int, error) {
	if img == nil {
		return 0, nil
	}

	mainNum := w.nextObj
	smaskNum := 0
	if len(img.AlphaData) > 0 {
		smaskNum = mainNum + 1
	}

	mainObj, smaskObj, err := img.BuildXObject(mainNum, smaskNum)
	if err != nil {
		return 0, err
	}
	w.objects = append(w.objects, mainObj)
	w.nextObj = mainNum + 1
	if smaskObj != nil {
		w.objects = append(w.objects, *smaskObj)
		w.nextObj = smaskNum + 1
	}
	return mainNum, nil
}

// SetInfo stores the document information dictionary.
func (w *Writer) SetInfo(info map[string]string) {
	d := core.NewDictionary()
	for k, v := range info {
		d.Set(k, core.PdfString(v))
	}
	w.info = d
}

// SetMetadataRef stores the metadata stream object number for the catalog.
func (w *Writer) SetMetadataRef(objNum int) {
	w.metadataRef = objNum
}

// SetOutputIntents stores output intent object numbers for the catalog.
func (w *Writer) SetOutputIntents(objNums []int) {
	w.outputIntents = objNums
}

// SetDocumentID sets the document ID pair for the trailer.
func (w *Writer) SetDocumentID(id1, id2 core.PdfHexString) {
	w.docIDPair = [2]core.PdfHexString{id1, id2}
	w.hasDocID = true
}

// SetEncryption stores encryption parameters for use during WriteTo.
func (w *Writer) SetEncryption(key []byte, algo core.EncryptionAlgorithm, encDictObjNum int, docID []byte) {
	w.encryptKey = key
	w.encryptAlgo = algo
	w.encryptObjNum = encDictObjNum
	w.documentID = docID
}

// SetOutlines stores the object number of the /Outlines root dictionary.
func (w *Writer) SetOutlines(objNum int) { w.outlinesNum = objNum }

// SetStructTreeRoot stores the StructTreeRoot object number for the catalog.
func (w *Writer) SetStructTreeRoot(objNum int) { w.structTreeRoot = objNum }

// SetMarkInfo sets whether the catalog should include /MarkInfo << /Marked true >>.
func (w *Writer) SetMarkInfo(marked bool) { w.markInfo = marked }

// PageRef returns an indirect reference for the page at the given 0-based index.
func (w *Writer) PageRef(pageIndex int) core.PdfIndirectReference {
	if pageIndex < 0 || pageIndex >= len(w.pages) {
		if len(w.pages) > 0 {
			return ref(w.pages[0])
		}
		return ref(0)
	}
	return ref(w.pages[pageIndex])
}

// PatchObject replaces the inner object for an already-allocated object number.
func (w *Writer) PatchObject(objNum int, obj core.PdfObject) {
	for i := range w.objects {
		if w.objects[i].Reference.ObjectNumber == objNum {
			w.objects[i].Object = obj
			return
		}
	}
}

// ReserveObject reserves the next object number without storing an object yet.
func (w *Writer) ReserveObject() int {
	num := w.nextObj
	w.nextObj++
	w.objects = append(w.objects, core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: num, GenerationNumber: 0},
		Object:    nil,
	})
	return num
}

// FillReserved replaces the placeholder object for a previously reserved object number.
func (w *Writer) FillReserved(num int, obj core.PdfObject) {
	for i := range w.objects {
		if w.objects[i].Reference.ObjectNumber == num {
			w.objects[i].Object = obj
			return
		}
	}
}

// encryptObject encrypts strings and stream data within an object in-place.
func (w *Writer) encryptObject(obj *core.PdfIndirectObject) error {
	objNum := obj.Reference.ObjectNumber
	genNum := obj.Reference.GenerationNumber

	// Don't encrypt the encryption dictionary itself.
	if objNum == w.encryptObjNum {
		return nil
	}

	return w.encryptPdfObject(obj.Object, objNum, genNum)
}

func (w *Writer) encryptPdfObject(obj core.PdfObject, objNum, genNum int) error {
	switch v := obj.(type) {
	case *core.PdfStream:
		encrypted, err := core.EncryptData(v.Data, w.encryptKey, objNum, genNum, w.encryptAlgo)
		if err != nil {
			return err
		}
		v.Data = encrypted
		v.Dictionary.Set("Length", core.PdfInteger(len(v.Data)))
		// Encrypt strings inside stream dictionary
		if err := w.encryptDict(v.Dictionary, objNum, genNum); err != nil {
			return err
		}
	case *core.PdfDictionary:
		if err := w.encryptDict(v, objNum, genNum); err != nil {
			return err
		}
	case core.PdfArray:
		for _, item := range v {
			if err := w.encryptPdfObject(item, objNum, genNum); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *Writer) encryptDict(d *core.PdfDictionary, objNum, genNum int) error {
	for _, key := range d.Keys() {
		val := d.Get(key)
		switch v := val.(type) {
		case core.PdfString:
			encrypted, err := core.EncryptData([]byte(v), w.encryptKey, objNum, genNum, w.encryptAlgo)
			if err != nil {
				return err
			}
			d.Set(key, core.PdfHexString(encrypted))
		case *core.PdfDictionary:
			if err := w.encryptDict(v, objNum, genNum); err != nil {
				return err
			}
		case core.PdfArray:
			if err := w.encryptPdfObject(v, objNum, genNum); err != nil {
				return err
			}
		case *core.PdfStream:
			if err := w.encryptPdfObject(v, objNum, genNum); err != nil {
				return err
			}
		}
	}
	return nil
}

// ref returns an indirect reference for the given object number.
func ref(num int) core.PdfIndirectReference {
	return core.PdfIndirectReference{ObjectNumber: num, GenerationNumber: 0}
}

// WriteTo writes a complete, valid PDF 1.7 file to out.
func (w *Writer) WriteTo(out io.Writer) (int64, error) {
	// Build pages object.
	kids := make(core.PdfArray, len(w.pages))
	for i, pn := range w.pages {
		kids[i] = ref(pn)
	}
	pagesDict := core.NewDictionary()
	pagesDict.Set("Type", core.PdfName("Pages"))
	pagesDict.Set("Kids", kids)
	pagesDict.Set("Count", core.PdfInteger(len(w.pages)))
	pagesNum := w.AddObject(pagesDict)

	// Patch each page's Parent reference.
	for _, obj := range w.objects {
		if dict, ok := obj.Object.(*core.PdfDictionary); ok {
			if t := dict.Get("Type"); t != nil {
				if name, ok2 := t.(core.PdfName); ok2 && name == "Page" {
					dict.Set("Parent", ref(pagesNum))
				}
			}
		}
	}

	// Build catalog.
	catalog := core.NewDictionary()
	catalog.Set("Type", core.PdfName("Catalog"))
	catalog.Set("Pages", ref(pagesNum))
	if w.metadataRef > 0 {
		catalog.Set("Metadata", ref(w.metadataRef))
	}
	if len(w.outputIntents) > 0 {
		arr := make(core.PdfArray, len(w.outputIntents))
		for i, n := range w.outputIntents {
			arr[i] = ref(n)
		}
		catalog.Set("OutputIntents", arr)
	}
	if w.outlinesNum > 0 {
		catalog.Set("Outlines", ref(w.outlinesNum))
	}
	if w.markInfo {
		mi := core.NewDictionary()
		mi.Set("Marked", core.PdfBoolean(true))
		catalog.Set("MarkInfo", mi)
	}
	if w.structTreeRoot > 0 {
		catalog.Set("StructTreeRoot", ref(w.structTreeRoot))
	}
	catalogNum := w.AddObject(catalog)

	// Build info object if set.
	var infoNum int
	if w.info != nil {
		infoNum = w.AddObject(w.info)
	}

	// Serialize.
	var buf bytes.Buffer

	// Header.
	buf.WriteString("%PDF-1.7\n")
	buf.Write([]byte{'%', 0xE2, 0xE3, 0xCF, 0xD3, '\n'})

	// Encrypt objects if encryption is configured.
	if len(w.encryptKey) > 0 {
		for i := range w.objects {
			if err := w.encryptObject(&w.objects[i]); err != nil {
				return 0, fmt.Errorf("encrypting object %d: %w", w.objects[i].Reference.ObjectNumber, err)
			}
		}
	}

	// Write objects and record offsets.
	offsets := make(map[int]int) // object number -> byte offset
	for i := range w.objects {
		offsets[w.objects[i].Reference.ObjectNumber] = buf.Len()
		w.objects[i].WriteTo(&buf)
	}

	// xref table.
	xrefOffset := buf.Len()
	totalObjects := w.nextObj // nextObj is one past the last used number
	fmt.Fprintf(&buf, "xref\n0 %d\n", totalObjects)
	fmt.Fprintf(&buf, "0000000000 65535 f \r\n")
	for num := 1; num < totalObjects; num++ {
		fmt.Fprintf(&buf, "%010d 00000 n \r\n", offsets[num])
	}

	// Trailer.
	buf.WriteString("trailer\n")
	trailer := core.NewDictionary()
	trailer.Set("Size", core.PdfInteger(totalObjects))
	trailer.Set("Root", ref(catalogNum))
	if infoNum > 0 {
		trailer.Set("Info", ref(infoNum))
	}
	if w.encryptObjNum > 0 {
		trailer.Set("Encrypt", ref(w.encryptObjNum))
		trailer.Set("ID", core.PdfArray{
			core.PdfHexString(w.documentID),
			core.PdfHexString(w.documentID),
		})
	} else if w.hasDocID {
		trailer.Set("ID", core.PdfArray{
			w.docIDPair[0],
			w.docIDPair[1],
		})
	}
	trailer.WriteTo(&buf)
	fmt.Fprintf(&buf, "\nstartxref\n%d\n%%%%EOF\n", xrefOffset)

	n, err := out.Write(buf.Bytes())
	return int64(n), err
}
