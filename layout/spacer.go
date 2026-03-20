package layout

// Spacer is a vertical space element.
type Spacer struct {
	Height float64
}

// NewSpacer creates a new spacer.
func NewSpacer(height float64) *Spacer {
	return &Spacer{Height: height}
}

// PlanLayout implements Element.
func (s *Spacer) PlanLayout(area LayoutArea) LayoutPlan {
	if s.Height > area.Height {
		return LayoutPlan{Status: LayoutNothing}
	}
	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: s.Height,
	}
}
