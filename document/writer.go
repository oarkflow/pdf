package document

import (
	"bytes"
	"fmt"
	"io"

	"github.com/oarkflow/pdf/core"
	pdfimage "github.com/oarkflow/pdf/image"
)

// Writer serializes PDF objects into a complete PDF file.
type Writer struct {
	objects   []core.PdfIndirectObject
	pages     []int // object numbers of page objects
	info      *core.PdfDictionary
	nextObj   int
	fontObjs  map[string]int // font name -> object number (cached)
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

// AddPage creates a page indirect object from the given Page and records it for
// the pages tree. The parent reference is patched later during WriteTo.
// Returns the object number of the page object.
func (w *Writer) AddPage(page *Page) int {
	// Build content stream.
	stream := core.NewStream(page.Contents)
	_ = stream.Compress()
	contentsNum := w.AddObject(stream)

	// Build resource dict
	res := core.NewDictionary()
	if page.Resources != nil {
		for _, k := range page.Resources.Keys() {
			res.Set(k, page.Resources.Get(k))
		}
	}

	// Handle fonts: create font objects for standard font names
	if len(page.Fonts) > 0 {
		fontDict := core.NewDictionary()
		for name, objNum := range page.Fonts {
			if objNum > 0 {
				// Already has an object number
				fontDict.Set(name, core.PdfIndirectReference{ObjectNumber: objNum})
			} else {
				// Create a standard font object
				fontObjNum := w.ensureStandardFont(name)
				fontDict.Set(name, core.PdfIndirectReference{ObjectNumber: fontObjNum})
			}
		}
		res.Set("Font", fontDict)
	}
	if len(page.Images) > 0 {
		xobjDict := core.NewDictionary()
		for name, entry := range page.Images {
			objNum := entry.ObjectNum
			if objNum == 0 && entry.Image != nil {
				objNum = w.addImageObject(entry.Image)
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

	num := w.AddObject(pageDict)
	w.pages = append(w.pages, num)
	return num
}

func (w *Writer) addImageObject(img *pdfimage.Image) int {
	if img == nil {
		return 0
	}

	mainNum := w.nextObj
	smaskNum := 0
	if len(img.AlphaData) > 0 {
		smaskNum = mainNum + 1
	}

	mainObj, smaskObj := img.BuildXObject(mainNum, smaskNum)
	w.objects = append(w.objects, mainObj)
	w.nextObj = mainNum + 1
	if smaskObj != nil {
		w.objects = append(w.objects, *smaskObj)
		w.nextObj = smaskNum + 1
	}
	return mainNum
}

// SetInfo stores the document information dictionary.
func (w *Writer) SetInfo(info map[string]string) {
	d := core.NewDictionary()
	for k, v := range info {
		d.Set(k, core.PdfString(v))
	}
	w.info = d
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
	trailer.WriteTo(&buf)
	fmt.Fprintf(&buf, "\nstartxref\n%d\n%%%%EOF\n", xrefOffset)

	n, err := out.Write(buf.Bytes())
	return int64(n), err
}
