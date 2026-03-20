package layout

import "fmt"

// ListType is the type of list.
type ListType int

const (
	UnorderedList ListType = iota
	OrderedList
)

// List is a list layout element.
type List struct {
	Type    ListType
	Items   []ListItem
	Indent  float64
	Spacing float64
	Bullet  string // custom bullet for unordered
	Start   int    // start number for ordered
}

// ListItem is an item in a list.
type ListItem struct {
	Content  Element
	Children *List // nested list
}

// NewList creates a list with string items.
func NewList(listType ListType, items ...string) *List {
	l := &List{
		Type:   listType,
		Indent: 20,
		Spacing: 2,
		Bullet: "\u2022", // bullet character
		Start:  1,
	}
	for _, text := range items {
		l.Items = append(l.Items, ListItem{Content: NewParagraph(text)})
	}
	return l
}

// NewElementList creates a list with Element items.
func NewElementList(listType ListType, items ...Element) *List {
	l := &List{
		Type:   listType,
		Indent: 20,
		Spacing: 2,
		Bullet: "\u2022",
		Start:  1,
	}
	for _, el := range items {
		l.Items = append(l.Items, ListItem{Content: el})
	}
	return l
}

// PlanLayout implements Element.
func (l *List) PlanLayout(area LayoutArea) LayoutPlan {
	contentWidth := area.Width - l.Indent
	if contentWidth < 0 {
		contentWidth = 0
	}

	var blocks []PlacedBlock
	curY := 0.0
	remaining := area.Height

	for i, item := range l.Items {
		// Marker
		marker := l.Bullet
		if l.Type == OrderedList {
			marker = fmt.Sprintf("%d.", l.Start+i)
		}

		markerWidth := measureText(marker, 12)
		_ = markerWidth

		// Layout item content
		if item.Content != nil {
			plan := item.Content.PlanLayout(LayoutArea{Width: contentWidth, Height: remaining})

			if plan.Status == LayoutNothing {
				if curY == 0 {
					return LayoutPlan{Status: LayoutNothing}
				}
				// Overflow
				return l.overflow(i, curY, blocks)
			}

			// Draw marker
			localMarker := marker
			localY := curY
			markerBlock := PlacedBlock{
				X: 0, Y: curY,
				Width: l.Indent, Height: 14,
				Draw: func(ctx *DrawContext, x, pdfY float64) {
					fontKey := ensureFont(ctx, "Helvetica", false, false)
					ctx.WriteString("BT\n")
					ctx.WriteString(fmt.Sprintf("/%s 12 Tf\n", fontKey))
					ctx.WriteString(fmt.Sprintf("%.2f %.2f Td\n", x, pdfY-10))
					ctx.WriteString(fmt.Sprintf("(%s) Tj\n", pdfEscapeString(localMarker)))
					ctx.WriteString("ET\n")
					_ = localY
				},
			}
			blocks = append(blocks, markerBlock)

			// Content blocks offset by indent
			for _, b := range plan.Blocks {
				b.X += l.Indent
				b.Y += curY
				blocks = append(blocks, b)
			}

			curY += plan.Consumed + l.Spacing
			remaining -= plan.Consumed + l.Spacing

			if plan.Status == LayoutPartial {
				// Create overflow list
				overflowItems := []ListItem{{Content: plan.Overflow}}
				if item.Children != nil {
					overflowItems[0].Children = item.Children
				}
				overflowItems = append(overflowItems, l.Items[i+1:]...)
				overflowList := &List{
					Type:    l.Type,
					Items:   overflowItems,
					Indent:  l.Indent,
					Spacing: l.Spacing,
					Bullet:  l.Bullet,
					Start:   l.Start + i,
				}
				return LayoutPlan{
					Status:   LayoutPartial,
					Consumed: curY,
					Blocks:   blocks,
					Overflow: overflowList,
				}
			}
		}

		// Nested list
		if item.Children != nil {
			nestedPlan := item.Children.PlanLayout(LayoutArea{Width: contentWidth, Height: remaining})
			for _, b := range nestedPlan.Blocks {
				b.X += l.Indent
				b.Y += curY
				blocks = append(blocks, b)
			}
			curY += nestedPlan.Consumed
			remaining -= nestedPlan.Consumed
		}
	}

	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: curY,
		Blocks:   blocks,
	}
}

func (l *List) overflow(fromIdx int, consumed float64, blocks []PlacedBlock) LayoutPlan {
	overflowList := &List{
		Type:    l.Type,
		Items:   l.Items[fromIdx:],
		Indent:  l.Indent,
		Spacing: l.Spacing,
		Bullet:  l.Bullet,
		Start:   l.Start + fromIdx,
	}
	return LayoutPlan{
		Status:   LayoutPartial,
		Consumed: consumed,
		Blocks:   blocks,
		Overflow: overflowList,
	}
}
