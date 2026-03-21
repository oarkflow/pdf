package document

import "github.com/oarkflow/pdf/core"

// OutlineItem represents a single bookmark entry in a PDF outline.
type OutlineItem struct {
	Title    string
	Page     int
	Children []OutlineItem
}

// Outline represents the document outline (bookmarks).
type Outline struct {
	Items []OutlineItem
}

// countDescendants returns the total number of visible descendants.
func countDescendants(items []OutlineItem) int {
	n := 0
	for _, item := range items {
		n++ // the item itself
		n += countDescendants(item.Children)
	}
	return n
}

// Build creates the PDF outline tree and returns the root /Outlines object number.
// It returns 0 if there are no outline items.
func (o *Outline) Build(w *Writer) int {
	if len(o.Items) == 0 {
		return 0
	}

	// Reserve the outlines root object number.
	rootNum := w.AddObject(core.NewDictionary()) // placeholder, patched below

	// Build children recursively.
	childNums := buildOutlineItems(w, o.Items, rootNum)

	// Patch root dict.
	rootDict := core.NewDictionary()
	rootDict.Set("Type", core.PdfName("Outlines"))
	rootDict.Set("First", ref(childNums[0]))
	rootDict.Set("Last", ref(childNums[len(childNums)-1]))
	rootDict.Set("Count", core.PdfInteger(countDescendants(o.Items)))
	w.PatchObject(rootNum, rootDict)

	return rootNum
}

// buildOutlineItems creates indirect objects for a slice of sibling OutlineItems
// and returns their object numbers. parentNum is the object number of their parent.
func buildOutlineItems(w *Writer, items []OutlineItem, parentNum int) []int {
	nums := make([]int, len(items))

	// First pass: reserve object numbers so we can set Prev/Next.
	for i := range items {
		nums[i] = w.AddObject(core.NewDictionary()) // placeholder
	}

	// Second pass: build each item dict and patch.
	for i, item := range items {
		d := core.NewDictionary()
		d.Set("Title", core.PdfString(item.Title))
		d.Set("Parent", ref(parentNum))

		// Destination: [pageRef /Fit]
		pageRef := w.PageRef(item.Page)
		d.Set("Dest", core.PdfArray{pageRef, core.PdfName("Fit")})

		// Sibling linked list.
		if i > 0 {
			d.Set("Prev", ref(nums[i-1]))
		}
		if i < len(items)-1 {
			d.Set("Next", ref(nums[i+1]))
		}

		// Children.
		if len(item.Children) > 0 {
			childNums := buildOutlineItems(w, item.Children, nums[i])
			d.Set("First", ref(childNums[0]))
			d.Set("Last", ref(childNums[len(childNums)-1]))
			d.Set("Count", core.PdfInteger(countDescendants(item.Children)))
		}

		w.PatchObject(nums[i], d)
	}

	return nums
}
