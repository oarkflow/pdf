package document

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
