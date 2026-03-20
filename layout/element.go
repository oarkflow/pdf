package layout

// Element is the core interface for all layout elements.
type Element interface {
	PlanLayout(area LayoutArea) LayoutPlan
}

// Measurable allows elements to report min/max content widths (for table auto-layout).
type Measurable interface {
	MinWidth() float64
	MaxWidth() float64
}
