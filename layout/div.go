package layout

import "fmt"

// Div is a block container with box model.
type Div struct {
	Box       BoxModel
	Children  []Element
	Width     float64 // 0 = auto (fill available)
	MinHeight float64
}

// NewDiv creates a new div with children.
func NewDiv(children ...Element) *Div {
	return &Div{Children: children}
}

// PlanLayout implements Element.
func (d *Div) PlanLayout(area LayoutArea) LayoutPlan {
	boxW := d.Box.TotalHorizontal()
	boxV := d.Box.TotalVertical()

	outerWidth := area.Width
	if d.Width > 0 {
		outerWidth = d.Width
	}
	contentWidth := outerWidth - boxW
	if contentWidth < 0 {
		contentWidth = 0
	}
	contentHeight := area.Height - boxV
	if contentHeight < 0 {
		return LayoutPlan{Status: LayoutNothing}
	}

	// Layout children sequentially
	var childBlocks []PlacedBlock
	childY := 0.0
	remaining := contentHeight
	var overflowChildren []Element

	for i, child := range d.Children {
		childArea := LayoutArea{Width: contentWidth, Height: remaining}
		plan := child.PlanLayout(childArea)

		switch plan.Status {
		case LayoutFull:
			// Offset child blocks by content area position
			for _, b := range plan.Blocks {
				b.X += d.Box.ContentLeft()
				b.Y += d.Box.ContentTop() + childY
				childBlocks = append(childBlocks, b)
			}
			childY += plan.Consumed
			remaining -= plan.Consumed

		case LayoutPartial:
			for _, b := range plan.Blocks {
				b.X += d.Box.ContentLeft()
				b.Y += d.Box.ContentTop() + childY
				childBlocks = append(childBlocks, b)
			}
			// Remaining children go to overflow
			overflowChildren = []Element{plan.Overflow}
			overflowChildren = append(overflowChildren, d.Children[i+1:]...)
			goto buildResult

		case LayoutNothing:
			overflowChildren = d.Children[i:]
			goto buildResult
		}
	}

buildResult:
	totalConsumed := d.Box.ContentTop() + childY + d.Box.PaddingBottom + d.Box.BorderBottomWidth + d.Box.MarginBottom
	if d.MinHeight > 0 && totalConsumed < d.MinHeight {
		totalConsumed = d.MinHeight
	}

	// Create the box drawing block
	boxBlock := PlacedBlock{
		X: 0, Y: 0,
		Width:    outerWidth,
		Height:   totalConsumed,
		Tag:      "Div",
		Draw:     d.drawBox(outerWidth, totalConsumed),
		Children: childBlocks,
	}

	if len(overflowChildren) > 0 {
		overflowDiv := &Div{
			Box:       d.Box,
			Children:  overflowChildren,
			Width:     d.Width,
			MinHeight: 0,
		}
		// No top margin/border on overflow continuation
		overflowDiv.Box.MarginTop = 0
		overflowDiv.Box.BorderTopWidth = 0
		overflowDiv.Box.PaddingTop = 0

		return LayoutPlan{
			Status:   LayoutPartial,
			Consumed: totalConsumed,
			Blocks:   []PlacedBlock{boxBlock},
			Overflow: overflowDiv,
		}
	}

	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: totalConsumed,
		Blocks:   []PlacedBlock{boxBlock},
	}
}

func (d *Div) drawBox(width, height float64) func(ctx *DrawContext, x, pdfY float64) {
	bg := d.Box.Background
	borderTop := d.Box.BorderTopWidth
	borderRight := d.Box.BorderRightWidth
	borderBottom := d.Box.BorderBottomWidth
	borderLeft := d.Box.BorderLeftWidth
	marginLeft := d.Box.MarginLeft
	marginTop := d.Box.MarginTop

	return func(ctx *DrawContext, x, pdfY float64) {
		bx := x + marginLeft
		by := pdfY - marginTop

		innerW := width - d.Box.MarginLeft - d.Box.MarginRight
		innerH := height - d.Box.MarginTop - d.Box.MarginBottom

		// Draw background
		if bg != nil {
			ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", bg[0], bg[1], bg[2]))
			ctx.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f re f\n", bx, by-innerH, innerW, innerH))
		}

		// Draw borders
		if borderTop > 0 || borderRight > 0 || borderBottom > 0 || borderLeft > 0 {
			if borderTop > 0 {
				borderColor := resolveBorderColor(d.Box.BorderTopColor, d.Box.BorderColor)
				ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG\n", borderColor[0], borderColor[1], borderColor[2]))
				ctx.WriteString(fmt.Sprintf("%.2f w\n", borderTop))
				ctx.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n", bx, by, bx+innerW, by))
			}
			if borderBottom > 0 {
				borderColor := resolveBorderColor(d.Box.BorderBottomColor, d.Box.BorderColor)
				ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG\n", borderColor[0], borderColor[1], borderColor[2]))
				ctx.WriteString(fmt.Sprintf("%.2f w\n", borderBottom))
				ctx.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n", bx, by-innerH, bx+innerW, by-innerH))
			}
			if borderLeft > 0 {
				borderColor := resolveBorderColor(d.Box.BorderLeftColor, d.Box.BorderColor)
				ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG\n", borderColor[0], borderColor[1], borderColor[2]))
				ctx.WriteString(fmt.Sprintf("%.2f w\n", borderLeft))
				ctx.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n", bx, by, bx, by-innerH))
			}
			if borderRight > 0 {
				borderColor := resolveBorderColor(d.Box.BorderRightColor, d.Box.BorderColor)
				ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG\n", borderColor[0], borderColor[1], borderColor[2]))
				ctx.WriteString(fmt.Sprintf("%.2f w\n", borderRight))
				ctx.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n", bx+innerW, by, bx+innerW, by-innerH))
			}
		}
	}
}

func resolveBorderColor(sideColor, fallback [3]float64) [3]float64 {
	if sideColor != ([3]float64{}) {
		return sideColor
	}
	return fallback
}
