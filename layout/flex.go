package layout

// FlexDirection is the main axis direction.
type FlexDirection int

const (
	FlexRow    FlexDirection = iota
	FlexColumn
)

// FlexWrap controls wrapping.
type FlexWrap int

const (
	FlexNoWrap FlexWrap = iota
	FlexWrapOn
)

// FlexJustify is main-axis alignment.
type FlexJustify int

const (
	JustifyStart       FlexJustify = iota
	JustifyEnd
	JustifyCenter
	JustifySpaceBetween
	JustifySpaceAround
	JustifySpaceEvenly
)

// FlexAlignItems is cross-axis alignment.
type FlexAlignItems int

const (
	AlignItemsStart   FlexAlignItems = iota
	AlignItemsEnd
	AlignItemsCenter
	AlignItemsStretch
)

// FlexContainer is a flexbox layout element.
type FlexContainer struct {
	Direction  FlexDirection
	Wrap       FlexWrap
	Justify    FlexJustify
	AlignItems FlexAlignItems
	Gap        float64
	Children   []FlexChild
	Box        BoxModel
}

// FlexChild is a child in a flex container.
type FlexChild struct {
	Element Element
	Grow    float64
	Shrink  float64
	Basis   float64 // 0 = auto
}

// NewFlex creates a flex container with default children (grow=1).
func NewFlex(direction FlexDirection, children ...Element) *FlexContainer {
	fc := &FlexContainer{
		Direction: direction,
	}
	for _, el := range children {
		fc.Children = append(fc.Children, FlexChild{
			Element: el,
			Grow:    1,
			Shrink:  1,
		})
	}
	return fc
}

type flexMeasure struct {
	basis    float64
	minSize  float64
	plan     LayoutPlan
}

// PlanLayout implements Element.
func (fc *FlexContainer) PlanLayout(area LayoutArea) LayoutPlan {
	contentW := area.Width - fc.Box.TotalHorizontal()
	contentH := area.Height - fc.Box.TotalVertical()
	if contentW < 0 {
		contentW = 0
	}
	if contentH < 0 {
		return LayoutPlan{Status: LayoutNothing}
	}

	if fc.Direction == FlexRow {
		return fc.layoutRow(area, contentW, contentH)
	}
	return fc.layoutColumn(area, contentW, contentH)
}

func (fc *FlexContainer) layoutRow(area LayoutArea, contentW, contentH float64) LayoutPlan {
	n := len(fc.Children)
	if n == 0 {
		return LayoutPlan{Status: LayoutFull, Consumed: fc.Box.TotalVertical()}
	}

	totalGap := fc.Gap * float64(n-1)
	availForChildren := contentW - totalGap

	// Phase 1: measure each child with basis or equal share
	measures := make([]flexMeasure, n)
	totalBasis := 0.0
	totalGrow := 0.0
	totalShrink := 0.0

	for i, ch := range fc.Children {
		basis := ch.Basis
		if basis == 0 {
			basis = availForChildren / float64(n)
		}
		measures[i].basis = basis
		totalBasis += basis
		totalGrow += ch.Grow
		totalShrink += ch.Shrink
	}

	// Phase 2: distribute space
	finalWidths := make([]float64, n)
	freeSpace := availForChildren - totalBasis

	for i, ch := range fc.Children {
		w := measures[i].basis
		if freeSpace > 0 && totalGrow > 0 {
			w += freeSpace * (ch.Grow / totalGrow)
		} else if freeSpace < 0 && totalShrink > 0 {
			w += freeSpace * (ch.Shrink / totalShrink)
		}
		if w < 0 {
			w = 0
		}
		finalWidths[i] = w
	}

	// Phase 3: layout and position
	var blocks []PlacedBlock
	maxHeight := 0.0

	// Calculate starting X based on justify
	positions := fc.justifyPositions(finalWidths, contentW, n)

	for i, ch := range fc.Children {
		plan := ch.Element.PlanLayout(LayoutArea{Width: finalWidths[i], Height: contentH})
		if plan.Consumed > maxHeight {
			maxHeight = plan.Consumed
		}

		// Cross-axis alignment
		offsetY := 0.0
		// Will be adjusted after we know maxHeight

		for _, b := range plan.Blocks {
			b.X += fc.Box.ContentLeft() + positions[i]
			b.Y += fc.Box.ContentTop() + offsetY
			blocks = append(blocks, b)
		}
	}

	// Adjust cross-axis after knowing max height
	// (simplified: would need second pass for proper alignment)

	consumed := fc.Box.ContentTop() + maxHeight + fc.Box.PaddingBottom + fc.Box.BorderBottomWidth + fc.Box.MarginBottom
	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: consumed,
		Blocks:   blocks,
	}
}

func (fc *FlexContainer) layoutColumn(area LayoutArea, contentW, contentH float64) LayoutPlan {
	n := len(fc.Children)
	if n == 0 {
		return LayoutPlan{Status: LayoutFull, Consumed: fc.Box.TotalVertical()}
	}

	var blocks []PlacedBlock
	curY := 0.0
	remaining := contentH

	for i, ch := range fc.Children {
		if i > 0 {
			curY += fc.Gap
			remaining -= fc.Gap
		}

		plan := ch.Element.PlanLayout(LayoutArea{Width: contentW, Height: remaining})

		switch plan.Status {
		case LayoutFull:
			for _, b := range plan.Blocks {
				b.X += fc.Box.ContentLeft()
				b.Y += fc.Box.ContentTop() + curY
				blocks = append(blocks, b)
			}
			curY += plan.Consumed
			remaining -= plan.Consumed

		case LayoutPartial:
			for _, b := range plan.Blocks {
				b.X += fc.Box.ContentLeft()
				b.Y += fc.Box.ContentTop() + curY
				blocks = append(blocks, b)
			}
			consumed := fc.Box.ContentTop() + curY + plan.Consumed + fc.Box.PaddingBottom + fc.Box.BorderBottomWidth + fc.Box.MarginBottom

			// Build overflow flex
			overflowChildren := []FlexChild{{Element: plan.Overflow, Grow: ch.Grow, Shrink: ch.Shrink, Basis: ch.Basis}}
			overflowChildren = append(overflowChildren, fc.Children[i+1:]...)
			overflowFlex := &FlexContainer{
				Direction:  fc.Direction,
				Wrap:       fc.Wrap,
				Justify:    fc.Justify,
				AlignItems: fc.AlignItems,
				Gap:        fc.Gap,
				Children:   overflowChildren,
				Box:        fc.Box,
			}
			return LayoutPlan{
				Status:   LayoutPartial,
				Consumed: consumed,
				Blocks:   blocks,
				Overflow: overflowFlex,
			}

		case LayoutNothing:
			if curY == 0 {
				return LayoutPlan{Status: LayoutNothing}
			}
			consumed := fc.Box.ContentTop() + curY + fc.Box.PaddingBottom + fc.Box.BorderBottomWidth + fc.Box.MarginBottom
			overflowChildren := fc.Children[i:]
			overflowFlex := &FlexContainer{
				Direction:  fc.Direction,
				Wrap:       fc.Wrap,
				Justify:    fc.Justify,
				AlignItems: fc.AlignItems,
				Gap:        fc.Gap,
				Children:   overflowChildren,
				Box:        fc.Box,
			}
			return LayoutPlan{
				Status:   LayoutPartial,
				Consumed: consumed,
				Blocks:   blocks,
				Overflow: overflowFlex,
			}
		}
	}

	consumed := fc.Box.ContentTop() + curY + fc.Box.PaddingBottom + fc.Box.BorderBottomWidth + fc.Box.MarginBottom
	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: consumed,
		Blocks:   blocks,
	}
}

func (fc *FlexContainer) justifyPositions(widths []float64, totalAvail float64, n int) []float64 {
	positions := make([]float64, n)
	totalChildWidth := 0.0
	for _, w := range widths {
		totalChildWidth += w
	}
	totalGaps := fc.Gap * float64(n-1)

	switch fc.Justify {
	case JustifyStart:
		x := 0.0
		for i := range widths {
			positions[i] = x
			x += widths[i] + fc.Gap
		}
	case JustifyEnd:
		x := totalAvail - totalChildWidth - totalGaps
		for i := range widths {
			positions[i] = x
			x += widths[i] + fc.Gap
		}
	case JustifyCenter:
		x := (totalAvail - totalChildWidth - totalGaps) / 2
		for i := range widths {
			positions[i] = x
			x += widths[i] + fc.Gap
		}
	case JustifySpaceBetween:
		if n == 1 {
			positions[0] = 0
		} else {
			gap := (totalAvail - totalChildWidth) / float64(n-1)
			x := 0.0
			for i := range widths {
				positions[i] = x
				x += widths[i] + gap
			}
		}
	case JustifySpaceAround:
		gap := (totalAvail - totalChildWidth) / float64(n)
		x := gap / 2
		for i := range widths {
			positions[i] = x
			x += widths[i] + gap
		}
	case JustifySpaceEvenly:
		gap := (totalAvail - totalChildWidth) / float64(n+1)
		x := gap
		for i := range widths {
			positions[i] = x
			x += widths[i] + gap
		}
	default:
		x := 0.0
		for i := range widths {
			positions[i] = x
			x += widths[i] + fc.Gap
		}
	}
	return positions
}
