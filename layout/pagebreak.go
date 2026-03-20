package layout

// PageBreak forces a new page.
type PageBreak struct{}

// NewPageBreak creates a page break element.
func NewPageBreak() *PageBreak {
	return &PageBreak{}
}

// PlanLayout implements Element. Always returns LayoutNothing to trigger a new page.
func (pb *PageBreak) PlanLayout(area LayoutArea) LayoutPlan {
	return LayoutPlan{Status: LayoutNothing}
}
