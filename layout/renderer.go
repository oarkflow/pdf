package layout

import "fmt"

// PageResult holds the rendered output for a single page.
type PageResult struct {
	Content    []byte
	Fonts      map[string]FontEntry
	Images     map[string]ImageEntry
	Links      []LinkAnnotation
	ExtGStates map[string]ExtGState
	Width      float64
	Height     float64
}

// RenderPages takes a list of elements and renders them across pages.
func RenderPages(elements []Element, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft float64) []PageResult {
	availWidth := pageWidth - marginLeft - marginRight
	availHeight := pageHeight - marginTop - marginBottom

	var pages []PageResult
	var currentCtx *DrawContext
	cursorY := 0.0 // top-down offset within content area
	remaining := availHeight

	startNewPage := func() {
		if currentCtx != nil {
			pages = append(pages, PageResult{
				Content:    currentCtx.ContentStream,
				Fonts:      currentCtx.Fonts,
				Images:     currentCtx.Images,
				Links:      currentCtx.Links,
				ExtGStates: currentCtx.ExtGStates,
				Width:      pageWidth,
				Height:     pageHeight,
			})
		}
		currentCtx = NewDrawContext(pageWidth, pageHeight)
		currentCtx.WriteString(fmt.Sprintf("1 1 1 rg 0 0 %.2f %.2f re f\n", pageWidth, pageHeight))
		cursorY = 0
		remaining = availHeight
	}

	startNewPage()

	// drawBlocks recursively draws placed blocks with coordinate transformation.
	var drawBlocks func(blocks []PlacedBlock, offsetX, offsetY float64)
	drawBlocks = func(blocks []PlacedBlock, offsetX, offsetY float64) {
		for _, b := range blocks {
			absX := offsetX + b.X
			absTopY := offsetY + b.Y
			if b.Draw != nil {
				// Transform top-down Y to PDF bottom-up Y
				pdfY := pageHeight - absTopY
				b.Draw(currentCtx, absX, pdfY)
			}
			if len(b.Children) > 0 {
				drawBlocks(b.Children, absX, absTopY)
			}
		}
	}

	// queue holds elements to process
	queue := make([]Element, len(elements))
	copy(queue, elements)

	maxRetries := 10000 // safety limit
	retries := 0

	for len(queue) > 0 {
		retries++
		if retries > maxRetries {
			// Safety: emit warning and stop to avoid infinite loops
			currentCtx.WriteString(fmt.Sprintf("%% layout: safety limit reached, %d elements dropped\n", len(queue)))
			break
		}

		el := queue[0]
		queue = queue[1:]

		area := LayoutArea{Width: availWidth, Height: remaining}
		plan := el.PlanLayout(area)

		switch plan.Status {
		case LayoutFull:
			drawBlocks(plan.Blocks, marginLeft, marginTop+cursorY)
			cursorY += plan.Consumed
			remaining -= plan.Consumed

		case LayoutPartial:
			drawBlocks(plan.Blocks, marginLeft, marginTop+cursorY)
			// Start new page for overflow
			startNewPage()
			if plan.Overflow != nil {
				// Prepend overflow to queue
				queue = append([]Element{plan.Overflow}, queue...)
			}

		case LayoutNothing:
			if remaining < availHeight {
				// Try on a fresh page
				startNewPage()
				queue = append([]Element{el}, queue...)
			} else {
				// Already on a fresh page and still nothing fits — skip to avoid infinite loop
				// but place it anyway with whatever we got
				drawBlocks(plan.Blocks, marginLeft, marginTop+cursorY)
			}
		}
	}

	// Flush last page
	if currentCtx != nil && (len(currentCtx.ContentStream) > 0 || len(pages) == 0) {
		pages = append(pages, PageResult{
			Content:    currentCtx.ContentStream,
			Fonts:      currentCtx.Fonts,
			Images:     currentCtx.Images,
			Links:      currentCtx.Links,
			ExtGStates: currentCtx.ExtGStates,
			Width:      pageWidth,
			Height:     pageHeight,
		})
	}

	return pages
}
