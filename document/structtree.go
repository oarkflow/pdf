package document

import (
	"fmt"

	"github.com/oarkflow/pdf/core"
)

// StructElement represents a node in the PDF structure tree.
type StructElement struct {
	Type       string // "Document", "H1"-"H6", "P", "Table", "TR", "TH", "TD", "Figure", "L", "LI", "Span"
	Children   []*StructElement
	MCID       int    // marked content ID; -1 if this is a container-only element
	PageNum    int    // 0-based page index this content appears on
	AltText    string // alternative text (for figures)
	Lang       string // language tag (e.g. "en-US")
	ActualText string // replacement text for the content
}

// StructureTree holds the root of the tagged PDF structure tree.
type StructureTree struct {
	Root     *StructElement
	nextMCID int
}

// NewStructureTree creates a new structure tree with a Document root element.
func NewStructureTree() *StructureTree {
	return &StructureTree{
		Root: &StructElement{
			Type: "Document",
			MCID: -1,
		},
	}
}

// AddElement adds a child element to the tree root.
func (t *StructureTree) AddElement(el *StructElement) {
	t.Root.Children = append(t.Root.Children, el)
}

// NextMCID returns the next available marked content ID and increments the counter.
func (t *StructureTree) NextMCID() int {
	id := t.nextMCID
	t.nextMCID++
	return id
}

// MCIDCount returns the total number of MCIDs allocated.
func (t *StructureTree) MCIDCount() int {
	return t.nextMCID
}

// Build serializes the structure tree into PDF objects via the writer and returns
// the object number of the StructTreeRoot.
func (t *StructureTree) Build(w *Writer) int {
	if t.Root == nil {
		return 0
	}

	// We need page object numbers. The writer records them in w.pages.
	// Collect all MCID-bearing elements for ParentTree construction.
	var mcidElems []*mcidEntry

	// Reserve an object number for the StructTreeRoot so children can reference it.
	rootObjNum := w.ReserveObject()

	// Recursively build StructElem objects.
	rootElemRef := t.buildElement(w, t.Root, rootObjNum, &mcidElems)

	// Build the ParentTree (a number tree mapping MCID -> StructElem reference).
	parentTreeNum := t.buildParentTree(w, mcidElems)

	// Build the StructTreeRoot dictionary.
	rootDict := core.NewDictionary()
	rootDict.Set("Type", core.PdfName("StructTreeRoot"))
	rootDict.Set("K", ref(rootElemRef))
	rootDict.Set("ParentTree", ref(parentTreeNum))

	// Build page array for ParentTreeNextKey.
	rootDict.Set("ParentTreeNextKey", core.PdfInteger(t.nextMCID))

	w.FillReserved(rootObjNum, rootDict)
	return rootObjNum
}

type mcidEntry struct {
	mcid       int
	elemObjNum int
}

func (t *StructureTree) buildElement(w *Writer, el *StructElement, parentObjNum int, mcidElems *[]*mcidEntry) int {
	dict := core.NewDictionary()
	dict.Set("Type", core.PdfName("StructElem"))
	dict.Set("S", core.PdfName(el.Type))
	dict.Set("P", ref(parentObjNum))

	if el.AltText != "" {
		dict.Set("Alt", core.PdfString(el.AltText))
	}
	if el.Lang != "" {
		dict.Set("Lang", core.PdfString(el.Lang))
	}
	if el.ActualText != "" {
		dict.Set("ActualText", core.PdfString(el.ActualText))
	}

	objNum := w.ReserveObject()

	if len(el.Children) > 0 {
		// Container element: build children.
		kids := make(core.PdfArray, 0, len(el.Children))
		for _, child := range el.Children {
			childNum := t.buildElement(w, child, objNum, mcidElems)
			kids = append(kids, ref(childNum))
		}
		dict.Set("K", kids)
	} else if el.MCID >= 0 {
		// Leaf element with marked content reference.
		mcRef := core.NewDictionary()
		mcRef.Set("Type", core.PdfName("MCR"))
		mcRef.Set("MCID", core.PdfInteger(el.MCID))
		if el.PageNum >= 0 && el.PageNum < len(w.pages) {
			mcRef.Set("Pg", ref(w.pages[el.PageNum]))
		}
		dict.Set("K", mcRef)

		*mcidElems = append(*mcidElems, &mcidEntry{
			mcid:       el.MCID,
			elemObjNum: objNum,
		})
	}

	w.FillReserved(objNum, dict)
	return objNum
}

func (t *StructureTree) buildParentTree(w *Writer, entries []*mcidEntry) int {
	// Number tree: Nums array with [key1 val1 key2 val2 ...]
	nums := make(core.PdfArray, 0, len(entries)*2)
	for _, e := range entries {
		nums = append(nums, core.PdfInteger(e.mcid), ref(e.elemObjNum))
	}
	dict := core.NewDictionary()
	dict.Set("Nums", nums)
	return w.AddObject(dict)
}

// BeginMarkedContent writes a BDC operator with the given tag and MCID to the content stream.
func BeginMarkedContent(stream []byte, tag string, mcid int) []byte {
	return append(stream, fmt.Sprintf("/%s <</MCID %d>> BDC\n", tag, mcid)...)
}

// EndMarkedContent writes an EMC operator to the content stream.
func EndMarkedContent(stream []byte) []byte {
	return append(stream, "EMC\n"...)
}
