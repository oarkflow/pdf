package layout

import "fmt"

// HeadingLevel represents heading levels H1-H6.
type HeadingLevel int

const (
	H1 HeadingLevel = 1
	H2 HeadingLevel = 2
	H3 HeadingLevel = 3
	H4 HeadingLevel = 4
	H5 HeadingLevel = 5
	H6 HeadingLevel = 6
)

// Heading is a heading element.
type Heading struct {
	Level HeadingLevel
	Text  string
	Color [3]float64
	Align Alignment
	ID    string // for bookmarks
}

// NewHeading creates a heading with default styling.
func NewHeading(level HeadingLevel, text string) *Heading {
	return &Heading{
		Level: level,
		Text:  text,
		Color: [3]float64{0, 0, 0},
	}
}

func (h *Heading) fontSize() float64 {
	switch h.Level {
	case H1:
		return 24
	case H2:
		return 20
	case H3:
		return 16
	case H4:
		return 14
	case H5:
		return 12
	case H6:
		return 10
	default:
		return 12
	}
}

// PlanLayout implements Element.
func (h *Heading) PlanLayout(area LayoutArea) LayoutPlan {
	fs := h.fontSize()
	p := &Paragraph{
		Runs: []TextRun{{
			Text:     h.Text,
			FontName: "Helvetica",
			FontSize: fs,
			Bold:     true,
			Color:    h.Color,
		}},
		Align:       h.Align,
		LineHeight:  1.2,
		SpaceBefore: fs * 0.5,
		SpaceAfter:  fs * 0.3,
	}

	plan := p.PlanLayout(area)
	// Re-tag blocks
	tag := fmt.Sprintf("H%d", h.Level)
	for i := range plan.Blocks {
		plan.Blocks[i].Tag = tag
	}
	return plan
}
