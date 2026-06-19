package layout

import "fmt"

// PageResult holds the rendered output for a single page.
type PageResult struct {
	Content    []byte
	Fonts      map[string]FontEntry
	Images     map[string]ImageEntry
	Links      []LinkAnnotation
	ExtGStates map[string]ExtGState
	Structure  []StructureElement
	Width      float64
	Height     float64
}

// RenderPages takes a list of elements and renders them across pages.
func RenderPages(elements []Element, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft float64) []PageResult {
	return renderPages(elements, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft, false)
}

// RenderTaggedPages renders pages with marked content and structure metadata.
func RenderTaggedPages(elements []Element, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft float64) []PageResult {
	return renderPages(elements, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft, true)
}

// RenderPagesWithHeaderFooter renders pages with repeating header and footer elements.
func RenderPagesWithHeaderFooter(elements []Element, headerEls, footerEls []Element, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft float64) []PageResult {
	return renderPagesWithHF(elements, headerEls, footerEls, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft, false)
}

// RenderTaggedPagesWithHeaderFooter renders tagged pages with repeating header and footer elements.
func RenderTaggedPagesWithHeaderFooter(elements []Element, headerEls, footerEls []Element, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft float64) []PageResult {
	return renderPagesWithHF(elements, headerEls, footerEls, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft, true)
}

func renderPages(elements []Element, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft float64, tagged bool) []PageResult {
	return renderPagesWithHF(elements, nil, nil, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft, tagged)
}

func renderPagesWithHF(elements []Element, headerEls, footerEls []Element, pageWidth, pageHeight, marginTop, marginRight, marginBottom, marginLeft float64, tagged bool) []PageResult {
	availWidth := pageWidth - marginLeft - marginRight
	availHeight := pageHeight - marginTop - marginBottom

	// Plan header elements to measure height and capture blocks
	var headerBlocks []PlacedBlock
	headerHeight := 0.0
	if len(headerEls) > 0 {
		for _, el := range headerEls {
			plan := el.PlanLayout(LayoutArea{Width: availWidth, Height: availHeight})
			headerBlocks = append(headerBlocks, plan.Blocks...)
			headerHeight += plan.Consumed
		}
	}

	// Plan footer elements
	var footerBlocks []PlacedBlock
	footerHeight := 0.0
	if len(footerEls) > 0 {
		for _, el := range footerEls {
			plan := el.PlanLayout(LayoutArea{Width: availWidth, Height: availHeight})
			footerBlocks = append(footerBlocks, plan.Blocks...)
			footerHeight += plan.Consumed
		}
	}

	if headerHeight+footerHeight > availHeight {
		headerHeight = 0
		footerHeight = 0
	}

	bodyAvailHeight := availHeight - headerHeight - footerHeight
	if bodyAvailHeight < 0 {
		bodyAvailHeight = 0
	}

	var pages []PageResult
	var currentCtx *DrawContext
	var currentStruct []StructureElement
	nextMCID := 0
	cursorY := 0.0 // top-down offset within content area
	remaining := availHeight

	flushPage := func() {
		if currentCtx == nil {
			return
		}
		// Draw footer at the bottom of the content area before saving
		if len(footerBlocks) > 0 {
			footerY := marginTop + headerHeight + bodyAvailHeight
			drawBlocksFn(footerBlocks, marginLeft, footerY, currentCtx, &currentStruct, &nextMCID, pageHeight, len(pages), tagged)
		}
		pages = append(pages, PageResult{
			Content:    currentCtx.ContentStream,
			Fonts:      currentCtx.Fonts,
			Images:     currentCtx.Images,
			Links:      currentCtx.Links,
			ExtGStates: currentCtx.ExtGStates,
			Structure:  currentStruct,
			Width:      pageWidth,
			Height:     pageHeight,
		})
	}

	startNewPage := func() {
		flushPage()
		currentCtx = NewDrawContext(pageWidth, pageHeight)
		currentStruct = nil
		nextMCID = 0
		if tagged {
			currentCtx.WriteString("/Artifact BMC\n")
		}
		currentCtx.WriteString(fmt.Sprintf("1 1 1 rg 0 0 %.2f %.2f re f\n", pageWidth, pageHeight))
		if tagged {
			currentCtx.WriteString("EMC\n")
		}
		// Draw header at the top of the content area
		if len(headerBlocks) > 0 {
			drawBlocksFn(headerBlocks, marginLeft, marginTop, currentCtx, &currentStruct, &nextMCID, pageHeight, len(pages), tagged)
		}
		cursorY = marginTop + headerHeight
		remaining = bodyAvailHeight
	}

	startNewPage()

	// queue holds elements to process
	queue := make([]Element, len(elements))
	copy(queue, elements)

	maxRetries := 10000 // safety limit
	retries := 0

	for len(queue) > 0 {
		retries++
		if retries > maxRetries {
			currentCtx.WriteString(fmt.Sprintf("%% layout: safety limit reached, %d elements dropped\n", len(queue)))
			break
		}

		el := queue[0]
		queue = queue[1:]

		area := LayoutArea{Width: availWidth, Height: remaining}
		plan := el.PlanLayout(area)

		switch plan.Status {
		case LayoutFull:
			drawBlocksFn(plan.Blocks, marginLeft, cursorY, currentCtx, &currentStruct, &nextMCID, pageHeight, len(pages), tagged)
			cursorY += plan.Consumed
			remaining -= plan.Consumed

		case LayoutPartial:
			drawBlocksFn(plan.Blocks, marginLeft, cursorY, currentCtx, &currentStruct, &nextMCID, pageHeight, len(pages), tagged)
			startNewPage()
			if plan.Overflow != nil {
				queue = append([]Element{plan.Overflow}, queue...)
			}

		case LayoutNothing:
			if remaining < bodyAvailHeight || cursorY > marginTop+headerHeight {
				startNewPage()
				queue = append([]Element{el}, queue...)
			} else {
				drawBlocksFn(plan.Blocks, marginLeft, cursorY, currentCtx, &currentStruct, &nextMCID, pageHeight, len(pages), tagged)
			}
		}
	}

	// Flush last page
	if currentCtx != nil && (len(currentCtx.ContentStream) > 0 || len(pages) == 0) {
		flushPage()
	}

	return pages
}

// drawBlocksFn recursively draws placed blocks with coordinate transformation.
func drawBlocksFn(blocks []PlacedBlock, offsetX, offsetY float64, ctx *DrawContext, structOut *[]StructureElement, mcidOut *int, pageHeight float64, pageNum int, tagged bool) {
	for _, b := range blocks {
		absX := offsetX + b.X
		absTopY := offsetY + b.Y
		var mcid int
		hasTag := tagged && b.Tag != ""
		structOnly := hasTag && b.StructOnly
		if hasTag && !structOnly {
			mcid = *mcidOut
			*mcidOut++
			ctx.BeginMarkedContent(b.Tag, mcid)
		} else if tagged && b.Draw != nil && structOnly {
			ctx.WriteString("/Artifact BMC\n")
		}
		if b.Draw != nil {
			pdfY := pageHeight - absTopY
			b.Draw(ctx, absX, pdfY)
		}
		if tagged && b.Draw != nil && structOnly {
			ctx.WriteString("EMC\n")
		}
		beforeChildren := len(*structOut)
		if len(b.Children) > 0 {
			drawBlocksFn(b.Children, absX, absTopY, ctx, structOut, mcidOut, pageHeight, pageNum, tagged)
		}
		var children []StructureElement
		if hasTag && len(*structOut) > beforeChildren {
			children = append(children, (*structOut)[beforeChildren:]...)
			*structOut = (*structOut)[:beforeChildren]
		}
		if hasTag {
			if !structOnly {
				ctx.EndMarkedContent()
			}
			*structOut = append(*structOut, StructureElement{
				Type:     b.Tag,
				MCID:     mcidForBlockFn(mcid, structOnly),
				PageNum:  pageNum,
				AltText:  b.AltText,
				Children: children,
			})
		}
	}
}

func mcidForBlockFn(mcid int, structOnly bool) int {
	if structOnly {
		return -1
	}
	return mcid
}
