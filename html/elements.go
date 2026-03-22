package html

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	pdfimage "github.com/oarkflow/pdf/image"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/svg"
)

// ParagraphElement renders styled text as a paragraph.
type ParagraphElement struct {
	Runs     []layout.TextRun
	Style    *ComputedStyle
	BoxModel layout.BoxModel
	PreWrap  bool
}

func (e *ParagraphElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	if len(e.Runs) == 0 {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	fontSize := 12.0
	lineHeightMul := 1.4
	if e.Style != nil {
		fontSize = e.Style.FontSize
		if e.Style.LineHeight > 0 {
			lineHeightMul = e.Style.LineHeight
		}
	}
	lineHeight := fontSize * lineHeightMul
	bm := e.BoxModel

	contentWidth := area.Width - bm.TotalHorizontal()
	if contentWidth <= 0 {
		contentWidth = area.Width
	}
	lines := wrapRuns(e.Runs, contentWidth, fontSize)
	if len(lines) == 0 {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	totalHeight := float64(len(lines))*lineHeight + bm.TotalVertical()

	block := makeParagraphBlock(lines, totalHeight, area.Width, lineHeight, fontSize, bm, e.Style)
	return layout.LayoutPlan{
		Status:   layout.LayoutFull,
		Consumed: totalHeight,
		Blocks:   []layout.PlacedBlock{block},
	}
}

// HeadingElement renders a heading.
type HeadingElement struct {
	Level    int
	Runs     []layout.TextRun
	Style    *ComputedStyle
	BoxModel layout.BoxModel
}

func (e *HeadingElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	fontSize := 24.0
	switch e.Level {
	case 2:
		fontSize = 20
	case 3:
		fontSize = 16
	case 4:
		fontSize = 14
	case 5:
		fontSize = 12
	case 6:
		fontSize = 10
	}
	if e.Style != nil && e.Style.FontSize > 0 {
		fontSize = e.Style.FontSize
	}
	runs := make([]layout.TextRun, len(e.Runs))
	copy(runs, e.Runs)
	for i := range runs {
		runs[i].Bold = true
		runs[i].FontSize = fontSize
	}
	pe := &ParagraphElement{Runs: runs, Style: e.Style, BoxModel: e.BoxModel}
	plan := pe.PlanLayout(area)
	tag := fmt.Sprintf("H%d", e.Level)
	for i := range plan.Blocks {
		plan.Blocks[i].Tag = tag
	}
	return plan
}

// DivElement is a generic block container.
type DivElement struct {
	Children []layout.Element
	Style    *ComputedStyle
	BoxModel layout.BoxModel
}

// InlineBoxElement renders an inline-block-like box with styled text content.
type InlineBoxElement struct {
	Runs       []layout.TextRun
	Style      *ComputedStyle
	BoxModel   layout.BoxModel
	OuterAlign string
	InnerAlign string
}

func (e *DivElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	bm := e.BoxModel
	innerWidth := area.Width - bm.TotalHorizontal()
	if innerWidth <= 0 {
		innerWidth = area.Width
	}
	contentHeight := area.Height - bm.TotalVertical()
	if contentHeight < 0 {
		return layout.LayoutPlan{Status: layout.LayoutNothing}
	}

	var childBlocks []layout.PlacedBlock
	childY := 0.0
	remaining := contentHeight
	var overflowChildren []layout.Element

	for i, child := range e.Children {
		childPlan := child.PlanLayout(layout.LayoutArea{Width: innerWidth, Height: remaining})
		switch childPlan.Status {
		case layout.LayoutFull:
			for _, b := range childPlan.Blocks {
				b.X += bm.ContentLeft()
				b.Y += bm.ContentTop() + childY
				childBlocks = append(childBlocks, b)
			}
			childY += childPlan.Consumed
			remaining -= childPlan.Consumed
		case layout.LayoutPartial:
			for _, b := range childPlan.Blocks {
				b.X += bm.ContentLeft()
				b.Y += bm.ContentTop() + childY
				childBlocks = append(childBlocks, b)
			}
			childY += childPlan.Consumed
			overflowChildren = append([]layout.Element{childPlan.Overflow}, e.Children[i+1:]...)
			goto buildResult
		case layout.LayoutNothing:
			if childY == 0 {
				return layout.LayoutPlan{Status: layout.LayoutNothing}
			}
			overflowChildren = e.Children[i:]
			goto buildResult
		}
	}

buildResult:
	totalHeight := childY + bm.TotalVertical()
	w := area.Width

	var allBlocks []layout.PlacedBlock
	if bm.Background != nil || bm.BorderTopWidth > 0 || bm.BorderBottomWidth > 0 || bm.BorderLeftWidth > 0 || bm.BorderRightWidth > 0 {
		capturedBm := bm
		capturedW := w
		capturedH := totalHeight
		bgBlock := layout.PlacedBlock{
			X: 0, Y: 0, Width: w, Height: totalHeight,
			Draw: func(ctx *layout.DrawContext, x, topY float64) {
				drawBoxModel(ctx, x, topY, capturedW, capturedH, capturedBm)
			},
		}
		allBlocks = append(allBlocks, bgBlock)
	}
	if shouldClipBoxChildren(e.Style, bm) {
		capturedBm := bm
		capturedW := w
		capturedH := totalHeight
		clipStart := layout.PlacedBlock{
			X: 0, Y: 0, Width: w, Height: totalHeight,
			Draw: func(ctx *layout.DrawContext, x, topY float64) {
				beginBoxClip(ctx, x, topY, capturedW, capturedH, capturedBm)
			},
		}
		clipEnd := layout.PlacedBlock{
			X: 0, Y: 0, Width: w, Height: totalHeight,
			Draw: func(ctx *layout.DrawContext, x, topY float64) {
				ctx.WriteString("Q\n")
			},
		}
		allBlocks = append(allBlocks, clipStart)
		allBlocks = append(allBlocks, childBlocks...)
		allBlocks = append(allBlocks, clipEnd)
	} else {
		allBlocks = append(allBlocks, childBlocks...)
	}

	if len(overflowChildren) > 0 {
		overflowDiv := &DivElement{
			Children: overflowChildren,
			Style:    e.Style,
			BoxModel: bm,
		}
		overflowDiv.BoxModel.MarginTop = 0
		overflowDiv.BoxModel.BorderTopWidth = 0
		overflowDiv.BoxModel.PaddingTop = 0

		return layout.LayoutPlan{
			Status:   layout.LayoutPartial,
			Consumed: totalHeight,
			Blocks:   allBlocks,
			Overflow: overflowDiv,
		}
	}

	return layout.LayoutPlan{
		Status:   layout.LayoutFull,
		Consumed: totalHeight,
		Blocks:   allBlocks,
	}
}

func (e *InlineBoxElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	if len(e.Runs) == 0 {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	fontSize := 12.0
	lineHeightMul := 1.4
	if e.Style != nil {
		if e.Style.FontSize > 0 {
			fontSize = e.Style.FontSize
		}
		if e.Style.LineHeight > 0 {
			lineHeightMul = e.Style.LineHeight
		}
	}
	lineHeight := fontSize * lineHeightMul
	bm := e.BoxModel

	maxContentWidth := area.Width - bm.TotalHorizontal()
	if maxContentWidth <= 0 {
		return layout.LayoutPlan{Status: layout.LayoutNothing}
	}

	contentWidth := measureRunsForIntrinsicWidth(e.Runs, e.Style)
	if e.Style != nil && !e.Style.Width.IsAuto() && e.Style.Width.Value > 0 {
		contentWidth = e.Style.Width.ToPoints(area.Width, fontSize) - bm.TotalHorizontal()
	}
	if contentWidth <= 0 {
		contentWidth = maxContentWidth
	}
	if contentWidth > maxContentWidth {
		contentWidth = maxContentWidth
	}

	lines := wrapRuns(e.Runs, contentWidth, fontSize)
	if len(lines) == 0 {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	usedContentWidth := 0.0
	for _, line := range lines {
		if line.width > usedContentWidth {
			usedContentWidth = line.width
		}
	}
	if e.Style == nil || e.Style.Width.IsAuto() || e.Style.Width.Value == 0 {
		contentWidth = usedContentWidth
	}
	if contentWidth <= 0 {
		contentWidth = maxContentWidth
	}

	totalWidth := contentWidth + bm.TotalHorizontal()
	totalHeight := float64(len(lines))*lineHeight + bm.TotalVertical()

	return layout.LayoutPlan{
		Status:   layout.LayoutFull,
		Consumed: totalHeight,
		Blocks: []layout.PlacedBlock{{
			X: 0, Y: 0, Width: totalWidth, Height: totalHeight, Tag: "Span",
			Draw: func(ctx *layout.DrawContext, x, topY float64) {
				drawX := x
				switch e.OuterAlign {
				case "center":
					drawX += (area.Width - totalWidth) / 2
				case "right", "end":
					drawX += area.Width - totalWidth
				}
				if drawX < x {
					drawX = x
				}

				drawBoxModel(ctx, drawX, topY, totalWidth, totalHeight, bm)

				textX := drawX + bm.ContentLeft()
				textY := topY - bm.ContentTop() - fontSize
				defaultColor := [3]float64{0.2, 0.2, 0.2}
				if e.Style != nil {
					defaultColor = e.Style.Color
				}

				for _, line := range lines {
					lineX := textX
					switch e.InnerAlign {
					case "center":
						lineX += (contentWidth - line.width) / 2
					case "right", "end":
						lineX += contentWidth - line.width
					}
					if lineX < textX {
						lineX = textX
					}
					for _, run := range line.runs {
						fs := run.FontSize
						if fs <= 0 {
							fs = fontSize
						}
						lineX += drawStyledRun(ctx, run, defaultColor, lineX, textY, textY+fs)
					}
					textY -= lineHeight
				}
			},
		}},
	}
}

// ListItem represents a single list item.
type ListItem struct {
	Marker   string
	Runs     []layout.TextRun
	Children []layout.Element
}

// ListElement renders an ordered or unordered list.
type ListElement struct {
	Items    []ListItem
	Ordered  bool
	Style    *ComputedStyle
	BoxModel layout.BoxModel
}

func (e *ListElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	fontSize := 12.0
	lineHeight := 16.8
	if e.Style != nil {
		fontSize = e.Style.FontSize
		lineHeight = e.Style.FontSize * e.Style.LineHeight
	}
	bm := e.BoxModel
	indent := 20.0
	consumed := bm.ContentTop()
	var blocks []layout.PlacedBlock

	for _, item := range e.Items {
		y := consumed
		cY := y
		cRuns := item.Runs
		cMarker := item.Marker
		cFS := fontSize
		cLH := lineHeight
		cBm := bm
		cColor := [3]float64{0.2, 0.2, 0.2}
		if e.Style != nil {
			cColor = e.Style.Color
		}

		block := layout.PlacedBlock{
			X: 0, Y: cY, Width: area.Width, Height: cLH, Tag: "LI",
			Draw: func(ctx *layout.DrawContext, x, topY float64) {
				fn := resolveFontName(cFS, false, false)
				ensureFont(ctx, fn)
				ctx.WriteString(fmt.Sprintf("BT\n/%s %.1f Tf\n%.3f %.3f %.3f rg\n%.2f %.2f Td\n(%s) Tj\nET\n",
					fn, cFS, cColor[0], cColor[1], cColor[2],
					x+cBm.ContentLeft(), topY-cFS, escPDF(cMarker)))
				drawTextRuns(ctx, cRuns, x+cBm.ContentLeft()+indent, topY, cFS, cLH)
			},
		}
		blocks = append(blocks, block)
		consumed += lineHeight
	}
	consumed += bm.PaddingBottom + bm.BorderBottomWidth + bm.MarginBottom

	return layout.LayoutPlan{
		Status:   layout.LayoutFull,
		Consumed: consumed,
		Blocks:   blocks,
	}
}

// TableCell represents a table cell.
type TableCell struct {
	Runs     []layout.TextRun
	IsHeader bool
	Style    *ComputedStyle
	Colspan  int
	Rowspan  int
}

// TableRow represents a table row.
type TableRow struct {
	Cells []TableCell
	Style *ComputedStyle
}

// TableElement renders an HTML table.
type TableElement struct {
	Rows     []TableRow
	Style    *ComputedStyle
	BoxModel layout.BoxModel
}

func (e *TableElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	if len(e.Rows) == 0 {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	bm := e.BoxModel
	tableWidth := area.Width - bm.TotalHorizontal()
	cellPad := 6.0
	defaultFontSize := 10.0

	numCols := tableColumnCount(e.Rows)
	if numCols == 0 {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}
	colWidths := resolveTableColumnWidths(tableWidth, e.Rows, cellPad, defaultFontSize)

	// Pre-compute wrapped lines and row heights for each row
	type cellLayout struct {
		lines []wrappedLine
		fs    float64
		lh    float64
		width float64
	}
	type rowLayout struct {
		cells  []cellLayout
		height float64
	}
	rowLayouts := make([]rowLayout, len(e.Rows))

	for ri, row := range e.Rows {
		rl := rowLayout{}
		maxH := 0.0
		colIdx := 0
		for _, cell := range row.Cells {
			fs := defaultFontSize
			if cell.Style != nil && cell.Style.FontSize > 0 {
				fs = cell.Style.FontSize
			}
			lhMul := 1.4
			if cell.Style != nil && cell.Style.LineHeight > 0 {
				lhMul = cell.Style.LineHeight
			}
			lh := fs * lhMul

			span := maxInt(1, cell.Colspan)
			cellWidth := sumColumnWidths(colWidths, colIdx, span)
			contentW := cellWidth - cellPad*2
			if contentW < 0 {
				contentW = 0
			}
			lines := wrapRuns(cell.Runs, contentW, fs)
			if len(lines) == 0 {
				lines = []wrappedLine{{}}
			}
			cellH := float64(len(lines))*lh + cellPad*2
			if cellH > maxH {
				maxH = cellH
			}
			rl.cells = append(rl.cells, cellLayout{lines: lines, fs: fs, lh: lh, width: cellWidth})
			colIdx += span
		}
		if maxH == 0 {
			maxH = defaultFontSize*1.4 + cellPad*2
		}
		rl.height = maxH
		rowLayouts[ri] = rl
	}

	consumed := bm.ContentTop()
	var blocks []layout.PlacedBlock

	for rowIdx, row := range e.Rows {
		cRowIdx := rowIdx
		cRow := row
		cRowY := consumed
		cRowHeight := rowLayouts[rowIdx].height
		cTableWidth := tableWidth
		cCellPad := cellPad
		cBm := bm
		cRowLayout := rowLayouts[rowIdx]

		block := layout.PlacedBlock{
			X: 0, Y: cRowY, Width: area.Width, Height: cRowHeight, Tag: "TR",
			Draw: func(ctx *layout.DrawContext, x, topY float64) {
				xOff := x + cBm.ContentLeft()

				// Row background
				if len(cRow.Cells) > 0 && cRow.Cells[0].IsHeader {
					bg := [3]float64{0.243, 0.243, 0.322}
					if cRow.Style != nil && cRow.Style.BackgroundColor != nil {
						bg = *cRow.Style.BackgroundColor
					} else if len(cRow.Cells) > 0 && cRow.Cells[0].Style != nil && cRow.Cells[0].Style.BackgroundColor != nil {
						bg = *cRow.Cells[0].Style.BackgroundColor
					}
					ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
						bg[0], bg[1], bg[2], xOff, topY-cRowHeight, cTableWidth, cRowHeight))
				} else if cRow.Style != nil && cRow.Style.BackgroundColor != nil {
					bg := *cRow.Style.BackgroundColor
					ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
						bg[0], bg[1], bg[2], xOff, topY-cRowHeight, cTableWidth, cRowHeight))
				} else if cRowIdx%2 == 0 {
					bg := [3]float64{0.973, 0.976, 0.984}
					ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
						bg[0], bg[1], bg[2], xOff, topY-cRowHeight, cTableWidth, cRowHeight))
				}

				// Per-cell background colors
				cellX := xOff
				for ci, cell := range cRow.Cells {
					if cell.Style != nil && cell.Style.BackgroundColor != nil {
						bg := *cell.Style.BackgroundColor
						cw := cRowLayout.cells[ci].width
						ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
							bg[0], bg[1], bg[2], cellX, topY-cRowHeight, cw, cRowHeight))
					}
					cellX += cRowLayout.cells[ci].width
				}

				cellX = xOff
				for ci, cell := range cRow.Cells {
					cellWidth := cRowLayout.cells[ci].width
					fs := cRowLayout.cells[ci].fs
					lh := cRowLayout.cells[ci].lh
					lines := cRowLayout.cells[ci].lines
					contentW := cellWidth - cCellPad*2

					fn := resolveFontName(fs, cell.IsHeader, false)
					ensureFont(ctx, fn)

					// Determine text color
					tc := [3]float64{0.235, 0.263, 0.341}
					if cell.IsHeader {
						tc = [3]float64{1, 1, 1}
						if cell.Style != nil && (cell.Style.Color != [3]float64{}) {
							tc = cell.Style.Color
						}
						ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", tc[0], tc[1], tc[2]))
					} else {
						if cell.Style != nil && (cell.Style.Color != [3]float64{}) {
							tc = cell.Style.Color
						}
						ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", tc[0], tc[1], tc[2]))
					}

					// Determine text alignment from cell style
					align := "left"
					if cell.Style != nil && cell.Style.TextAlign != "" {
						align = cell.Style.TextAlign
					}

					// Draw each wrapped line
					textY := topY - cCellPad - fs
					for _, line := range lines {
						if len(line.runs) == 0 {
							textY -= lh
							continue
						}

						// Calculate line X based on alignment
						lineX := cellX + cCellPad
						switch align {
						case "right":
							lineX = cellX + cellWidth - cCellPad - line.width
						case "center":
							lineX = cellX + cCellPad + (contentW-line.width)/2
						}
						if lineX < cellX+cCellPad {
							lineX = cellX + cCellPad
						}

						for _, run := range line.runs {
							runFS := run.FontSize
							if runFS <= 0 {
								runFS = fs
							}
							run.Bold = run.Bold || cell.IsHeader
							lineX += drawStyledRun(ctx, run, tc, lineX, textY, textY+runFS)
						}
						textY -= lh
					}

					cellX += cellWidth
				}

				// Row bottom border
				borderY := topY - cRowHeight
				ctx.WriteString(fmt.Sprintf("0.918 0.929 0.945 RG 0.5 w %.2f %.2f m %.2f %.2f l S\n",
					xOff, borderY, xOff+cTableWidth, borderY))
			},
		}
		blocks = append(blocks, block)
		consumed += cRowHeight
	}
	consumed += bm.PaddingBottom + bm.BorderBottomWidth + bm.MarginBottom

	return layout.LayoutPlan{
		Status:   layout.LayoutFull,
		Consumed: consumed,
		Blocks:   blocks,
	}
}

// ImageElement renders an image.
type ImageElement struct {
	Src      string
	Alt      string
	Style    *ComputedStyle
	BoxModel layout.BoxModel
	Fetcher  *Fetcher
}

func (e *ImageElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	bm := e.BoxModel
	contentWidth := area.Width - bm.TotalHorizontal()
	contentHeight := area.Height - bm.TotalVertical()
	if contentWidth <= 0 {
		contentWidth = area.Width
	}
	if contentHeight <= 0 {
		contentHeight = area.Height
	}

	makePlaceholder := func() layout.LayoutPlan {
		alt := e.Alt
		w, h := e.computeImageDisplaySize(100, 50, contentWidth, contentHeight)
		return layout.LayoutPlan{
			Status:   layout.LayoutFull,
			Consumed: h + bm.TotalVertical(),
			Blocks: []layout.PlacedBlock{{
				X: bm.ContentLeft(), Y: bm.ContentTop(),
				Width:  w,
				Height: h,
				Draw: func(ctx *layout.DrawContext, x, pdfY float64) {
					// Gray background
					ctx.WriteString(fmt.Sprintf("0.94 0.94 0.94 rg %.2f %.2f %.2f %.2f re f\n", x, pdfY-h, w, h))
					// Border
					ctx.WriteString(fmt.Sprintf("0.8 0.8 0.8 RG 0.5 w %.2f %.2f %.2f %.2f re S\n", x, pdfY-h, w, h))
					// Alt text
					if alt != "" {
						fn := "Helvetica"
						ensureFont(ctx, fn)
						ctx.WriteString(fmt.Sprintf("0.5 0.5 0.5 rg BT /%s 8 Tf %.2f %.2f Td (%s) Tj ET\n", fn, x+4, pdfY-h/2-4, escPDF(alt)))
					}
				},
			}},
		}
	}

	if e.Fetcher == nil || e.Src == "" {
		return makePlaceholder()
	}

	data, err := e.Fetcher.Fetch(e.Src)
	if err != nil {
		return makePlaceholder()
	}

	// Detect SVG content and render as vector graphics.
	if isSVGData(data) {
		return e.planSVGLayout(data, contentWidth, contentHeight, bm)
	}

	decoded, err := pdfimage.Load(data)
	if err != nil {
		return makePlaceholder()
	}

	imgW, imgH := e.computeImageDisplaySize(
		float64(decoded.Width), float64(decoded.Height),
		contentWidth, contentHeight,
	)

	// Map object-fit to layout fit mode.
	fit := layout.FitContain
	if e.Style != nil {
		switch e.Style.ObjectFit {
		case "cover":
			fit = layout.FitCover
		case "fill":
			fit = layout.FitFill
		case "none":
			fit = layout.FitNone
		}
	}

	img := &layout.ImageElement{
		Source: data,
		Image: layout.ImageEntry{
			Image: decoded,
		},
		OrigW:  decoded.Width,
		OrigH:  decoded.Height,
		Alt:    e.Alt,
		Width:  imgW,
		Height: imgH,
		Fit:    fit,
	}

	plan := img.PlanLayout(layout.LayoutArea{Width: contentWidth, Height: contentHeight})
	for i := range plan.Blocks {
		plan.Blocks[i].X += bm.ContentLeft()
		plan.Blocks[i].Y += bm.ContentTop()
	}
	plan.Consumed += bm.TotalVertical()
	return plan
}

// computeImageDisplaySize resolves CSS width/height/max-width/max-height for an
// image with the given native dimensions (in points), constrained to the
// available content area.
func (e *ImageElement) computeImageDisplaySize(nativeW, nativeH, contentWidth, contentHeight float64) (float64, float64) {
	if nativeW <= 0 || nativeH <= 0 {
		nativeW, nativeH = 100, 100
	}
	aspect := nativeW / nativeH

	hasW := e.Style != nil && !e.Style.Width.IsAuto() && e.Style.Width.Value > 0
	hasH := e.Style != nil && !e.Style.Height.IsAuto() && e.Style.Height.Value > 0

	w, h := nativeW, nativeH
	if hasW {
		w = e.Style.Width.ToPoints(contentWidth, 12)
	}
	if hasH {
		h = e.Style.Height.ToPoints(contentHeight, 12)
	}

	// Maintain aspect ratio when only one dimension is specified.
	if hasW && !hasH {
		h = w / aspect
	} else if hasH && !hasW {
		w = h * aspect
	}

	// Apply max-width constraint.
	maxW := contentWidth
	if e.Style != nil && !e.Style.MaxWidth.IsAuto() && e.Style.MaxWidth.Value > 0 {
		mw := e.Style.MaxWidth.ToPoints(contentWidth, 12)
		if mw < maxW {
			maxW = mw
		}
	}
	if w > maxW {
		h = h * maxW / w
		w = maxW
	}

	// Apply max-height constraint.
	if e.Style != nil && !e.Style.MaxHeight.IsAuto() && e.Style.MaxHeight.Value > 0 {
		maxH := e.Style.MaxHeight.ToPoints(contentHeight, 12)
		if h > maxH {
			w = w * maxH / h
			h = maxH
		}
	}

	return w, h
}

// isSVGData checks if the data looks like SVG content.
func isSVGData(data []byte) bool {
	header := data
	if len(header) > 512 {
		header = header[:512]
	}
	trimmed := bytes.TrimSpace(header)
	return bytes.HasPrefix(trimmed, []byte("<?xml")) ||
		bytes.HasPrefix(trimmed, []byte("<svg")) ||
		bytes.Contains(header, []byte("<svg"))
}

// planSVGLayout parses and renders SVG data as inline vector PDF content.
func (e *ImageElement) planSVGLayout(data []byte, contentWidth, contentHeight float64, bm layout.BoxModel) layout.LayoutPlan {
	root, err := svg.Parse(data)
	if err != nil {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	// Extract native SVG dimensions.
	svgW, svgH := parseSVGDimensions(root)
	if svgW <= 0 || svgH <= 0 {
		svgW, svgH = 300, 150 // default fallback
	}

	// Use shared sizing logic (respects width/height/max-width/max-height/aspect-ratio).
	displayW, displayH := e.computeImageDisplaySize(svgW, svgH, contentWidth, contentHeight)

	// Render SVG to PDF content stream.
	renderer := svg.NewRenderer(svgW, svgH)
	svgContent := renderer.Render(root)

	scaleX := displayW / svgW
	scaleY := displayH / svgH

	// Handle object-fit for SVG: adjust scale to maintain aspect ratio.
	objectFit := ""
	if e.Style != nil {
		objectFit = e.Style.ObjectFit
	}
	// Center offset for cover/contain positioning.
	offsetX, offsetY := 0.0, 0.0
	switch objectFit {
	case "cover":
		// Scale uniformly using the larger factor; clip the overflow.
		s := math.Max(scaleX, scaleY)
		offsetX = (displayW - svgW*s) / 2
		offsetY = (displayH - svgH*s) / 2
		scaleX, scaleY = s, s
	case "contain", "scale-down":
		// Scale uniformly using the smaller factor; center in the box.
		s := math.Min(scaleX, scaleY)
		offsetX = (displayW - svgW*s) / 2
		offsetY = (displayH - svgH*s) / 2
		scaleX, scaleY = s, s
	case "", "fill":
		// Default: stretch to fill (scaleX/scaleY differ). No offset.
		// But if only one CSS dimension was set, aspect ratio is already
		// preserved by computeImageDisplaySize, so this is fine.
	}
	block := layout.PlacedBlock{
		X:      bm.ContentLeft(),
		Y:      bm.ContentTop(),
		Width:  displayW,
		Height: displayH,
		Draw: func(ctx *layout.DrawContext, x, pdfY float64) {
			// Save state, set clipping rect to prevent overflow.
			ctx.WriteString(fmt.Sprintf("q %.2f %.2f %.2f %.2f re W n\n",
				x, pdfY-displayH, displayW, displayH))
			// Place the SVG in one matrix so scaling, Y-flip, and object-fit
			// offsets are applied consistently.
			ctx.WriteString(fmt.Sprintf("%.4f 0 0 %.4f %.4f %.4f cm\n",
				scaleX, -scaleY, x+offsetX, pdfY-offsetY))
			ctx.WriteString(string(svgContent))
			ctx.WriteString("Q\n")
		},
	}

	return layout.LayoutPlan{
		Status:   layout.LayoutFull,
		Consumed: displayH + bm.TotalVertical(),
		Blocks:   []layout.PlacedBlock{block},
	}
}

// parseSVGDimensions extracts width and height from an SVG root node in points.
func parseSVGDimensions(root *svg.SVGNode) (float64, float64) {
	const pxToPt = 0.75

	w := parseSVGLength(root.Attrs["width"], pxToPt)
	h := parseSVGLength(root.Attrs["height"], pxToPt)
	if w > 0 && h > 0 {
		return w, h
	}

	// Fall back to viewBox.
	if vb, ok := root.Attrs["viewBox"]; ok {
		if _, _, vw, vh, ok := parseSVGViewBox(vb); ok && vw > 0 && vh > 0 {
			return vw * pxToPt, vh * pxToPt
		}
	}
	return w, h
}

func parseSVGViewBox(vb string) (minX, minY, width, height float64, ok bool) {
	parts := strings.Fields(strings.ReplaceAll(vb, ",", " "))
	if len(parts) != 4 {
		return 0, 0, 0, 0, false
	}
	values := [4]float64{}
	for i, part := range parts {
		v, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0, 0, 0, 0, false
		}
		values[i] = v
	}
	return values[0], values[1], values[2], values[3], true
}

// parseSVGLength parses a length value like "200", "200px", "10cm", etc. to points.
func parseSVGLength(s string, pxToPt float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Strip known unit suffixes.
	units := map[string]float64{
		"px": pxToPt,
		"pt": 1,
		"in": 72,
		"cm": 28.3465,
		"mm": 2.83465,
		"em": 12,
	}
	for suffix, factor := range units {
		if strings.HasSuffix(s, suffix) {
			s = strings.TrimSuffix(s, suffix)
			v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return 0
			}
			return v * factor
		}
	}
	// No unit: treat as px.
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v * pxToPt
}

// HRElement renders a horizontal rule.
type HRElement struct{ Style *ComputedStyle }

func (e *HRElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	height := 13.0
	block := layout.PlacedBlock{
		X: 0, Y: 0, Width: area.Width, Height: height,
		Draw: func(ctx *layout.DrawContext, x, topY float64) {
			y := topY - 6
			ctx.WriteString(fmt.Sprintf("0.8 0.8 0.8 RG 0.5 w %.2f %.2f m %.2f %.2f l S\n", x, y, x+area.Width, y))
		},
	}
	return layout.LayoutPlan{Status: layout.LayoutFull, Consumed: height, Blocks: []layout.PlacedBlock{block}}
}

type FlexChildElement struct {
	Element layout.Element
	Style   *ComputedStyle
}

// FlexContainerElement renders a flex container.
type FlexContainerElement struct {
	Children []FlexChildElement
	Style    *ComputedStyle
	BoxModel layout.BoxModel
}

func (e *FlexContainerElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	bm := e.BoxModel
	innerWidth := area.Width - bm.TotalHorizontal()
	consumed := bm.ContentTop()
	var blocks []layout.PlacedBlock

	isRow := e.Style == nil || e.Style.FlexDirection == "" || e.Style.FlexDirection == "row" || e.Style.FlexDirection == "row-reverse"

	justify := ""
	if e.Style != nil {
		justify = e.Style.JustifyContent
	}

	if isRow {
		gap := 0.0
		if e.Style != nil && e.Style.Gap > 0 {
			gap = e.Style.Gap
		}
		n := len(e.Children)
		totalGap := gap * math.Max(0, float64(n-1))
		availableWidth := innerWidth - totalGap
		if availableWidth < 0 {
			availableWidth = 0
		}

		widths := make([]float64, n)
		basisTotal := 0.0
		totalGrow := 0.0
		for i, child := range e.Children {
			grow := 0.0
			if child.Style != nil && child.Style.FlexGrow > 0 {
				grow = child.Style.FlexGrow
			}
			basis := 0.0
			if child.Style != nil && !child.Style.FlexBasis.IsAuto() && child.Style.FlexBasis.Value > 0 {
				basis = child.Style.FlexBasis.ToPoints(availableWidth, 12)
			} else if child.Style != nil && !child.Style.Width.IsAuto() && child.Style.Width.Value > 0 {
				// Use explicit width (e.g. w-1/3) as basis
				basis = child.Style.Width.ToPoints(innerWidth, 12)
			} else if grow == 0 {
				basis = estimateIntrinsicWidth(child.Element)
			}
			widths[i] = basis
			basisTotal += basis
			totalGrow += grow
		}
		if basisTotal > availableWidth && basisTotal > 0 {
			scale := availableWidth / basisTotal
			for i := range widths {
				widths[i] *= scale
			}
			basisTotal = availableWidth
		}
		remainingWidth := availableWidth - basisTotal
		if remainingWidth < 0 {
			remainingWidth = 0
		}
		if totalGrow > 0 {
			for i, child := range e.Children {
				grow := 0.0
				if child.Style != nil && child.Style.FlexGrow > 0 {
					grow = child.Style.FlexGrow
				}
				if grow > 0 {
					widths[i] += remainingWidth * (grow / totalGrow)
				}
			}
		}
		// Don't distribute extra space equally when no grow — leave it for justify-content

		// Layout each child and record its consumed height
		type childLayout struct {
			blocks   []layout.PlacedBlock
			width    float64
			consumed float64
		}
		layouts := make([]childLayout, n)
		maxH := 0.0
		usedWidth := 0.0
		for i, child := range e.Children {
			plan := child.Element.PlanLayout(layout.LayoutArea{Width: widths[i], Height: 5000})
			layouts[i] = childLayout{blocks: plan.Blocks, width: widths[i], consumed: plan.Consumed}
			usedWidth += widths[i]
			if plan.Consumed > maxH {
				maxH = plan.Consumed
			}
		}

		// Calculate starting X and per-item gap based on justify-content
		freeSpace := innerWidth - usedWidth - totalGap
		if freeSpace < 0 {
			freeSpace = 0
		}
		startX := bm.ContentLeft()
		extraGap := 0.0
		switch justify {
		case "flex-end", "end":
			startX += freeSpace
		case "center":
			startX += freeSpace / 2
		case "space-between":
			if n > 1 {
				extraGap = freeSpace / float64(n-1)
			}
		case "space-around":
			if n > 0 {
				itemGap := freeSpace / float64(n)
				startX += itemGap / 2
				extraGap = itemGap
			}
		case "space-evenly":
			if n > 0 {
				itemGap := freeSpace / float64(n+1)
				startX += itemGap
				extraGap = itemGap
			}
		default: // flex-start or empty
		}

		xOffset := startX
		for i := range layouts {
			cl := &layouts[i]
			for _, b := range cl.blocks {
				b.X += xOffset
				b.Y += consumed
				blocks = append(blocks, b)
			}
			xOffset += cl.width
			if i < n-1 {
				xOffset += gap + extraGap
			}
		}
		consumed += maxH
	} else {
		// Column layout
		gap := 0.0
		if e.Style != nil && e.Style.Gap > 0 {
			gap = e.Style.Gap
		}
		for i, child := range e.Children {
			plan := child.Element.PlanLayout(layout.LayoutArea{Width: innerWidth, Height: 5000})
			for _, b := range plan.Blocks {
				b.X += bm.ContentLeft()
				b.Y += consumed
				blocks = append(blocks, b)
			}
			consumed += plan.Consumed
			if i < len(e.Children)-1 {
				consumed += gap
			}
		}
	}
	consumed += bm.PaddingBottom + bm.BorderBottomWidth + bm.MarginBottom

	if bm.Background != nil || bm.BorderTopWidth > 0 || bm.BorderBottomWidth > 0 || bm.BorderLeftWidth > 0 || bm.BorderRightWidth > 0 {
		capturedBm := bm
		capturedW := area.Width
		capturedH := consumed
		bgBlock := layout.PlacedBlock{
			X: 0, Y: 0, Width: capturedW, Height: capturedH,
			Draw: func(ctx *layout.DrawContext, x, topY float64) {
				drawBoxModel(ctx, x, topY, capturedW, capturedH, capturedBm)
			},
		}
		blocks = append([]layout.PlacedBlock{bgBlock}, blocks...)
	}

	return layout.LayoutPlan{Status: layout.LayoutFull, Consumed: consumed, Blocks: blocks}
}

// GridContainerElement renders a grid container.
type GridContainerElement struct {
	Children []layout.Element
	Style    *ComputedStyle
	BoxModel layout.BoxModel
	Columns  []GridTrack
	Rows     []GridTrack
}

type GridTrack struct {
	Size CSSLength
	Fr   float64
	Auto bool
}

func (e *GridContainerElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	bm := e.BoxModel
	innerWidth := area.Width - bm.TotalHorizontal()
	cols := len(e.Columns)
	if cols == 0 {
		cols = 1
	}
	colWidth := innerWidth / float64(cols)
	consumed := bm.ContentTop()
	var blocks []layout.PlacedBlock
	col := 0
	rowH := 0.0

	for _, child := range e.Children {
		plan := child.PlanLayout(layout.LayoutArea{Width: colWidth, Height: 5000})
		for _, b := range plan.Blocks {
			b.X += bm.ContentLeft() + float64(col)*colWidth
			b.Y += consumed
			blocks = append(blocks, b)
		}
		if plan.Consumed > rowH {
			rowH = plan.Consumed
		}
		col++
		if col >= cols {
			col = 0
			consumed += rowH
			rowH = 0
		}
	}
	if col > 0 {
		consumed += rowH
	}
	consumed += bm.PaddingBottom + bm.BorderBottomWidth + bm.MarginBottom
	return layout.LayoutPlan{Status: layout.LayoutFull, Consumed: consumed, Blocks: blocks}
}

// ---------------------------------------------------------------------------
// Text wrapping
// ---------------------------------------------------------------------------

type wrappedLine struct {
	runs  []layout.TextRun
	width float64
}

type cr struct {
	ch  rune
	run layout.TextRun
}

func wrapRuns(runs []layout.TextRun, maxWidth, defaultFontSize float64) []wrappedLine {
	if maxWidth <= 0 {
		maxWidth = defaultFontSize
	}

	// Flatten all runs into chars with style info
	var chars []cr
	for _, r := range runs {
		for _, ch := range r.Text {
			chars = append(chars, cr{ch, r})
		}
	}
	if len(chars) == 0 {
		return nil
	}

	// Split into words and explicit whitespace runs so we preserve
	// intentional multi-space gaps from utilities like Tailwind's space-x-*.
	type segment struct {
		chars     []cr
		width     float64
		space     bool
		breakLine bool
	}
	var segments []segment
	var cur []cr
	spaceRun := false
	flush := func() {
		if len(cur) == 0 {
			return
		}
		segments = append(segments, segment{
			chars: append([]cr(nil), cur...),
			width: measureCR(cur),
			space: spaceRun,
		})
		cur = nil
	}
	for _, c := range chars {
		switch c.ch {
		case '\n':
			flush()
			segments = append(segments, segment{breakLine: true})
			spaceRun = false
			continue
		case ' ', '\t':
			if !spaceRun {
				flush()
				spaceRun = true
			}
			if c.ch == '\t' {
				for i := 0; i < 4; i++ {
					cur = append(cur, cr{' ', c.run})
				}
			} else {
				cur = append(cur, c)
			}
			continue
		}
		if spaceRun {
			flush()
			spaceRun = false
		}
		cur = append(cur, c)
	}
	flush()
	if len(segments) == 0 {
		return nil
	}

	var lines []wrappedLine
	var lineChars []cr
	lineW := 0.0

	trimTrailingWhitespace := func() {
		for len(lineChars) > 0 {
			last := lineChars[len(lineChars)-1]
			if last.ch != ' ' && last.ch != '\t' {
				break
			}
			lineW -= charWidth(last.ch, last.run.FontSize, last.run.Bold, last.run.FontName)
			lineChars = lineChars[:len(lineChars)-1]
		}
		if lineW < 0 {
			lineW = 0
		}
	}
	flushLine := func(forceBlank bool) {
		trimTrailingWhitespace()
		if len(lineChars) > 0 {
			lines = append(lines, buildLine(lineChars, lineW))
		} else if forceBlank {
			lines = append(lines, wrappedLine{})
		}
		lineChars = nil
		lineW = 0
	}

	for _, seg := range segments {
		if seg.breakLine {
			flushLine(true)
			continue
		}
		if seg.space {
			if len(lineChars) == 0 {
				continue
			}
			if lineW+seg.width > maxWidth {
				flushLine(false)
				continue
			}
			lineChars = append(lineChars, seg.chars...)
			lineW += seg.width
			continue
		}
		if len(lineChars) > 0 && lineW+seg.width > maxWidth {
			flushLine(false)
		}
		if seg.width > maxWidth {
			wordChars := seg.chars
			for len(wordChars) > 0 {
				chunk, chunkW, rest := takeCharsFitting(wordChars, maxWidth-lineW)
				if len(chunk) == 0 {
					chunk, chunkW, rest = takeCharsFitting(wordChars, maxWidth)
				}
				lineChars = append(lineChars, chunk...)
				lineW += chunkW
				wordChars = rest
				if len(wordChars) > 0 {
					flushLine(false)
				}
			}
			continue
		}
		lineChars = append(lineChars, seg.chars...)
		lineW += seg.width
	}
	if len(lineChars) > 0 {
		flushLine(false)
	}
	return lines
}

// unused type kept for compatibility
type charRun = cr

func measureCR(chars []cr) float64 {
	w := 0.0
	for _, c := range chars {
		w += charWidth(c.ch, c.run.FontSize, c.run.Bold, c.run.FontName)
	}
	return w
}

func takeCharsFitting(chars []cr, maxWidth float64) ([]cr, float64, []cr) {
	if len(chars) == 0 {
		return nil, 0, nil
	}

	width := 0.0
	idx := 0
	for idx < len(chars) {
		chW := charWidth(chars[idx].ch, chars[idx].run.FontSize, chars[idx].run.Bold, chars[idx].run.FontName)
		if idx > 0 && width+chW > maxWidth {
			break
		}
		width += chW
		idx++
	}
	if idx == 0 {
		idx = 1
		width = charWidth(chars[0].ch, chars[0].run.FontSize, chars[0].run.Bold, chars[0].run.FontName)
	}

	return chars[:idx], width, chars[idx:]
}

func tableColumnCount(rows []TableRow) int {
	maxCols := 0
	for _, row := range rows {
		cols := 0
		for _, cell := range row.Cells {
			cols += maxInt(1, cell.Colspan)
		}
		if cols > maxCols {
			maxCols = cols
		}
	}
	return maxCols
}

func resolveTableColumnWidths(tableWidth float64, rows []TableRow, cellPad, defaultFontSize float64) []float64 {
	numCols := tableColumnCount(rows)
	if numCols == 0 {
		return nil
	}

	minWidths := make([]float64, numCols)
	prefWidths := make([]float64, numCols)
	stretchWeights := tableColumnStretchWeights(rows, numCols)

	for _, row := range rows {
		colIdx := 0
		for _, cell := range row.Cells {
			span := maxInt(1, cell.Colspan)
			minW, prefW := measureRunsForColumnSizing(cell.Runs, defaultFontSize)
			minW += cellPad * 2
			prefW += cellPad * 2

			if span == 1 {
				if minW > minWidths[colIdx] {
					minWidths[colIdx] = minW
				}
				if prefW > prefWidths[colIdx] {
					prefWidths[colIdx] = prefW
				}
			} else {
				perMin := minW / float64(span)
				perPref := prefW / float64(span)
				for i := 0; i < span && colIdx+i < numCols; i++ {
					if perMin > minWidths[colIdx+i] {
						minWidths[colIdx+i] = perMin
					}
					if perPref > prefWidths[colIdx+i] {
						prefWidths[colIdx+i] = perPref
					}
				}
			}

			colIdx += span
		}
	}

	for i := range minWidths {
		if prefWidths[i] < minWidths[i] {
			prefWidths[i] = minWidths[i]
		}
	}

	sumMin := sumFloat64s(minWidths)
	sumPref := sumFloat64s(prefWidths)
	if tableWidth <= 0 {
		return make([]float64, numCols)
	}

	widths := make([]float64, numCols)
	switch {
	case sumPref <= tableWidth:
		copy(widths, prefWidths)
		extra := tableWidth - sumPref
		if extra > 0 {
			totalWeight := sumFloat64s(stretchWeights)
			if totalWeight == 0 {
				totalWeight = float64(numCols)
			}
			for i := range widths {
				weight := stretchWeights[i]
				if weight == 0 {
					if sumFloat64s(stretchWeights) == 0 {
						weight = 1
					} else {
						continue
					}
				}
				widths[i] += extra * (weight / totalWeight)
			}
		}
	case sumMin >= tableWidth:
		scale := tableWidth / sumMin
		for i := range widths {
			widths[i] = minWidths[i] * scale
		}
	default:
		copy(widths, minWidths)
		extra := tableWidth - sumMin
		flexTotal := 0.0
		for i := range widths {
			flexTotal += (prefWidths[i] - minWidths[i]) * maxFloat64(stretchWeights[i], 0)
		}
		if flexTotal == 0 {
			flexTotal = float64(numCols)
		}
		for i := range widths {
			flex := (prefWidths[i] - minWidths[i]) * maxFloat64(stretchWeights[i], 0)
			if flex == 0 {
				if flexTotal == float64(numCols) {
					flex = 1
				} else {
					continue
				}
			}
			widths[i] += extra * (flex / flexTotal)
		}
	}

	return widths
}

func tableColumnStretchWeights(rows []TableRow, numCols int) []float64 {
	weights := make([]float64, numCols)
	for i := range weights {
		weights[i] = 1
	}

	colIdxs := make([]int, numCols)
	for _, row := range rows {
		colIdx := 0
		for _, cell := range row.Cells {
			span := maxInt(1, cell.Colspan)
			weight := tableCellStretchWeight(cell)
			for i := 0; i < span && colIdx+i < numCols; i++ {
				if colIdxs[colIdx+i] == 0 || weight > weights[colIdx+i] {
					weights[colIdx+i] = weight
				}
				colIdxs[colIdx+i]++
			}
			colIdx += span
		}
	}
	return weights
}

func tableCellStretchWeight(cell TableCell) float64 {
	align := "left"
	if cell.Style != nil && cell.Style.TextAlign != "" {
		align = cell.Style.TextAlign
	}

	switch align {
	case "left", "justify", "start", "":
		return 1
	case "center":
		return 0.15
	case "right", "end":
		return 0.05
	default:
		return 1
	}
}

func measureRunsForColumnSizing(runs []layout.TextRun, defaultFontSize float64) (minWidth, prefWidth float64) {
	lineWidth := 0.0
	tokenWidth := 0.0

	flushToken := func() {
		if tokenWidth > minWidth {
			minWidth = tokenWidth
		}
		tokenWidth = 0
	}
	flushLine := func() {
		if lineWidth > prefWidth {
			prefWidth = lineWidth
		}
		lineWidth = 0
	}

	for _, run := range runs {
		fontSize := run.FontSize
		if fontSize <= 0 {
			fontSize = defaultFontSize
		}
		for _, ch := range run.Text {
			switch ch {
			case '\n':
				flushToken()
				flushLine()
			case ' ', '\t':
				flushToken()
				lineWidth += fontSize * 0.25
			default:
				chWidth := charWidth(ch, fontSize, run.Bold, run.FontName)
				tokenWidth += chWidth
				lineWidth += chWidth
			}
		}
	}

	flushToken()
	flushLine()
	return minWidth, prefWidth
}

func sumColumnWidths(widths []float64, start, span int) float64 {
	total := 0.0
	for i := 0; i < span && start+i < len(widths); i++ {
		total += widths[start+i]
	}
	return total
}

func sumFloat64s(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func estimateIntrinsicWidth(el layout.Element) float64 {
	switch v := el.(type) {
	case *ParagraphElement:
		return measureRunsForIntrinsicWidth(v.Runs, v.Style)
	case *HeadingElement:
		return measureRunsForIntrinsicWidth(v.Runs, v.Style)
	case *DivElement:
		maxW := 0.0
		for _, child := range v.Children {
			if w := estimateIntrinsicWidth(child); w > maxW {
				maxW = w
			}
		}
		return maxW + v.BoxModel.TotalHorizontal()
	case *ImageElement:
		if v.Style != nil && !v.Style.Width.IsAuto() && v.Style.Width.Value > 0 {
			return v.Style.Width.ToPoints(0, 12) + v.BoxModel.TotalHorizontal()
		}
		return 100 + v.BoxModel.TotalHorizontal()
	case *InlineBoxElement:
		return measureRunsForIntrinsicWidth(v.Runs, v.Style) + v.BoxModel.TotalHorizontal()
	default:
		return 0
	}
}

func measureRunsForIntrinsicWidth(runs []layout.TextRun, style *ComputedStyle) float64 {
	maxLine := 0.0
	lineWidth := 0.0
	defaultFontSize := 12.0
	if style != nil && style.FontSize > 0 {
		defaultFontSize = style.FontSize
	}

	for _, run := range runs {
		fs := run.FontSize
		if fs <= 0 {
			fs = defaultFontSize
		}
		for _, ch := range run.Text {
			if ch == '\n' {
				if lineWidth > maxLine {
					maxLine = lineWidth
				}
				lineWidth = 0
				continue
			}
			lineWidth += charWidth(ch, fs, run.Bold, run.FontName)
		}
	}
	if lineWidth > maxLine {
		maxLine = lineWidth
	}
	if maxLine == 0 {
		return 0
	}
	return maxLine*1.25 + 10
}

func buildLine(chars []cr, width float64) wrappedLine {
	var runs []layout.TextRun
	var cur strings.Builder
	var curRun layout.TextRun
	first := true

	flush := func() {
		if cur.Len() > 0 {
			r := curRun
			r.Text = cur.String()
			runs = append(runs, r)
			cur.Reset()
		}
	}

	for _, c := range chars {
		r := c.run
		r.Text = ""
		if first {
			curRun = r
			first = false
		} else if r.FontName != curRun.FontName || r.FontSize != curRun.FontSize ||
			r.Bold != curRun.Bold || r.Italic != curRun.Italic ||
			r.Color != curRun.Color || r.Underline != curRun.Underline ||
			r.Strike != curRun.Strike || r.Link != curRun.Link {
			flush()
			curRun = r
		}
		cur.WriteRune(c.ch)
	}
	flush()
	return wrappedLine{runs: runs, width: width}
}

// ---------------------------------------------------------------------------
// Character width using Helvetica metrics
// ---------------------------------------------------------------------------

// charWidth returns the width of a single character in points.
func charWidth(ch rune, fontSize float64, bold bool, fontName string) float64 {
	if fontSize <= 0 {
		fontSize = 12
	}
	if fontName == "Courier" || strings.HasPrefix(fontName, "Courier") {
		return fontSize * 0.6
	}
	// Use Helvetica widths (in 1000 units/em)
	var w int
	if bold {
		w = helveticaBoldWidth(ch)
	} else {
		w = helveticaWidth(ch)
	}
	if w == 0 {
		w = 500 // fallback
	}
	return fontSize * float64(w) / 1000.0
}

// helveticaWidth returns the width of a rune in Helvetica (1000 units/em).
func helveticaWidth(r rune) int {
	if r < 256 {
		return helveticaWidths[r]
	}
	// Common Unicode → approximate width
	switch r {
	case 0x2022: // bullet •
		return 350
	case 0x2013: // en dash –
		return 556
	case 0x2014: // em dash —
		return 1000
	case 0x2018, 0x2019: // ' '
		return 222
	case 0x201C, 0x201D: // " "
		return 333
	case 0x2026: // …
		return 1000
	case 0x20AC: // €
		return 556
	}
	return 500
}

func helveticaBoldWidth(r rune) int {
	if r < 256 {
		return helveticaBoldWidths[r]
	}
	w := helveticaWidth(r)
	return w * 107 / 100
}

// Helvetica width table (WinAnsiEncoding, 1000 units/em)
var helveticaWidths = [256]int{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	278, 278, 355, 556, 556, 889, 667, 191,
	333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556,
	278, 278, 584, 584, 584, 556,
	1015,
	667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611,
	278, 278, 278, 469, 556, 333,
	556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556,
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500,
	334, 260, 334, 584, 0,
	556, 0, 222, 556, 333, 1000, 556, 556, 333, 1000, 667, 333, 1000, 0, 611, 0,
	0, 333, 333, 500, 500, 350, 556, 1000, 333, 1000, 500, 333, 944, 0, 500, 667,
	278, 333, 556, 556, 556, 556, 260, 556, 333, 737, 370, 556, 584, 333, 737, 333,
	400, 584, 333, 333, 333, 556, 537, 278, 333, 333, 365, 556, 834, 834, 834, 611,
	667, 667, 667, 667, 667, 667, 1000, 722, 667, 667, 667, 667, 278, 278, 278, 278,
	722, 722, 778, 778, 778, 778, 778, 584, 778, 722, 722, 722, 722, 667, 667, 611,
	556, 556, 556, 556, 556, 556, 889, 500, 556, 556, 556, 556, 278, 278, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 584, 611, 556, 556, 556, 556, 500, 556, 500,
}

var helveticaBoldWidths = [256]int{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	278, 333, 474, 556, 556, 889, 722, 238,
	333, 333, 389, 584, 278, 333, 278, 278,
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556,
	333, 333, 584, 584, 584, 611,
	975,
	722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778,
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611,
	333, 278, 333, 584, 556, 333,
	556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611,
	611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500,
	389, 280, 389, 584, 0,
	556, 0, 278, 556, 500, 1000, 556, 556, 333, 1000, 667, 333, 1000, 0, 611, 0,
	0, 333, 333, 556, 556, 350, 556, 1000, 333, 1000, 556, 333, 944, 0, 500, 667,
	278, 333, 556, 556, 556, 556, 280, 556, 333, 737, 370, 556, 584, 333, 737, 333,
	400, 584, 333, 333, 333, 611, 556, 278, 333, 333, 365, 556, 834, 834, 834, 611,
	722, 722, 722, 722, 722, 722, 1000, 722, 667, 667, 667, 667, 278, 278, 278, 278,
	722, 722, 778, 778, 778, 778, 778, 584, 778, 722, 722, 722, 722, 667, 667, 611,
	556, 556, 556, 556, 556, 556, 889, 556, 556, 556, 556, 556, 278, 278, 278, 278,
	611, 611, 611, 611, 611, 611, 611, 584, 611, 611, 611, 611, 611, 556, 611, 556,
}

// ---------------------------------------------------------------------------
// Rendering helpers
// ---------------------------------------------------------------------------

func makeParagraphBlock(lines []wrappedLine, totalHeight, blockWidth, lineHeight, fontSize float64, bm layout.BoxModel, style *ComputedStyle) layout.PlacedBlock {
	cLines := lines
	cBm := bm
	cLH := lineHeight
	cFS := fontSize
	cStyle := style
	cW := blockWidth
	contentWidth := blockWidth - bm.TotalHorizontal()
	if contentWidth <= 0 {
		contentWidth = blockWidth
	}

	return layout.PlacedBlock{
		X: 0, Y: 0, Width: blockWidth, Height: totalHeight, Tag: "P",
		Draw: func(ctx *layout.DrawContext, x, topY float64) {
			drawBoxModel(ctx, x, topY, cW, totalHeight, cBm)

			textX := x + cBm.ContentLeft()
			textY := topY - cBm.ContentTop() - cFS

			color := [3]float64{0.2, 0.2, 0.2}
			if cStyle != nil {
				color = cStyle.Color
			}

			for _, line := range cLines {
				lineX := textX
				if cStyle != nil {
					switch cStyle.TextAlign {
					case "center":
						lineX = textX + (contentWidth-line.width)/2
					case "right":
						lineX = textX + contentWidth - line.width
					}
					if lineX < textX {
						lineX = textX
					}
				}
				for _, run := range line.runs {
					fs := run.FontSize
					if fs <= 0 {
						fs = cFS
					}
					lineX += drawStyledRun(ctx, run, color, lineX, textY, textY+fs)
				}
				textY -= cLH
			}
		},
	}
}

func drawStyledRun(ctx *layout.DrawContext, run layout.TextRun, defaultColor [3]float64, x, baselineY, lineTopY float64) float64 {
	fs := run.FontSize
	if fs <= 0 {
		fs = 12
	}
	fn := resolveFontName(fs, run.Bold, run.Italic)
	ensureFont(ctx, fn)

	color := defaultColor
	if run.Color != ([3]float64{}) {
		color = run.Color
	}

	text := toWinAnsi(run.Text)
	ctx.WriteString(fmt.Sprintf("BT\n/%s %.1f Tf\n%.3f %.3f %.3f rg\n%.2f %.2f Td\n(%s) Tj\nET\n",
		fn, fs, color[0], color[1], color[2], x, baselineY, escPDF(text)))

	width := measureStr(run.Text, fs, run.Bold, run.FontName)
	if run.Link != "" && width > 0 {
		ctx.AddLink(x, lineTopY-fs, x+width, lineTopY, run.Link)
	}
	return width
}

func drawTextRuns(ctx *layout.DrawContext, runs []layout.TextRun, x, topY, fontSize, lineHeight float64) {
	color := [3]float64{0.2, 0.2, 0.2}
	curX := x
	for _, run := range runs {
		if run.Text == "" {
			continue
		}
		runFS := run.FontSize
		if runFS <= 0 {
			runFS = fontSize
		}
		curX += drawStyledRun(ctx, run, color, curX, topY-runFS, topY)
	}
	_ = lineHeight
}

func resolveFontName(_ float64, bold, italic bool) string {
	switch {
	case bold && italic:
		return "Helvetica-BoldOblique"
	case bold:
		return "Helvetica-Bold"
	case italic:
		return "Helvetica-Oblique"
	default:
		return "Helvetica"
	}
}

func ensureFont(ctx *layout.DrawContext, fontName string) {
	if _, ok := ctx.Fonts[fontName]; !ok {
		ctx.Fonts[fontName] = layout.FontEntry{PDFName: fontName}
	}
}

func measureStr(s string, fontSize float64, bold bool, fontName string) float64 {
	w := 0.0
	for _, ch := range s {
		w += charWidth(ch, fontSize, bold, fontName)
	}
	return w
}

func escPDF(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `(`, `\(`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	return s
}

// toWinAnsi converts UTF-8 string to WinAnsiEncoding-safe string.
// Characters not in WinAnsi are replaced with closest equivalents.
func toWinAnsi(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 128 {
			b.WriteRune(r)
			continue
		}
		if r >= 160 && r <= 255 {
			b.WriteByte(byte(r))
			continue
		}
		// Map common Unicode chars to WinAnsi byte values
		switch r {
		case 0x2022: // bullet •
			b.WriteByte(0x95) // WinAnsi 149
		case 0x2013: // en dash –
			b.WriteByte(0x96) // WinAnsi 150
		case 0x2014: // em dash —
			b.WriteByte(0x97) // WinAnsi 151
		case 0x2018: // left single quote '
			b.WriteByte(0x91) // WinAnsi 145
		case 0x2019: // right single quote '
			b.WriteByte(0x92) // WinAnsi 146
		case 0x201C: // left double quote "
			b.WriteByte(0x93) // WinAnsi 147
		case 0x201D: // right double quote "
			b.WriteByte(0x94) // WinAnsi 148
		case 0x2026: // ellipsis …
			b.WriteByte(0x85) // WinAnsi 133
		case 0x20AC: // euro €
			b.WriteByte(0x80) // WinAnsi 128
		case 0x2122: // trademark ™
			b.WriteByte(0x99) // WinAnsi 153
		case 0x0152: // OE ligature
			b.WriteByte(0x8C)
		case 0x0153: // oe ligature
			b.WriteByte(0x9C)
		case 0x0160: // S caron
			b.WriteByte(0x8A)
		case 0x0161: // s caron
			b.WriteByte(0x9A)
		case 0x0178: // Y diaeresis
			b.WriteByte(0x9F)
		case 0x017D: // Z caron
			b.WriteByte(0x8E)
		case 0x017E: // z caron
			b.WriteByte(0x9E)
		case 0x0192: // f hook
			b.WriteByte(0x83)
		case 0x02C6: // circumflex
			b.WriteByte(0x88)
		case 0x02DC: // tilde
			b.WriteByte(0x98)
		case 0x2020: // dagger
			b.WriteByte(0x86)
		case 0x2021: // double dagger
			b.WriteByte(0x87)
		case 0x2030: // per mille
			b.WriteByte(0x89)
		case 0x2039: // single left angle
			b.WriteByte(0x8B)
		case 0x203A: // single right angle
			b.WriteByte(0x9B)
		case 0x201A: // single low-9 quote
			b.WriteByte(0x82)
		case 0x201E: // double low-9 quote
			b.WriteByte(0x84)
		default:
			b.WriteByte('?') // fallback
		}
	}
	return b.String()
}

// roundedRect returns PDF path operators for a rounded rectangle.
// (x, y) is the bottom-left corner; w and h are width and height; r is the corner radius.
func roundedRect(x, y, w, h, r float64) string {
	return roundedRectCorners(x, y, w, h, r, r, r, r)
}

// roundedRectCorners returns PDF path operators for a rounded rectangle with
// per-corner radii in top-left, top-right, bottom-right, bottom-left order.
func roundedRectCorners(x, y, w, h, tl, tr, br, bl float64) string {
	const kappa = 0.5523
	tl = clampCornerRadius(tl, w, h)
	tr = clampCornerRadius(tr, w, h)
	br = clampCornerRadius(br, w, h)
	bl = clampCornerRadius(bl, w, h)
	tlk := tl * kappa
	trk := tr * kappa
	brk := br * kappa
	blk := bl * kappa
	return fmt.Sprintf(
		"%.2f %.2f m "+
			"%.2f %.2f l "+
			"%.2f %.2f %.2f %.2f %.2f %.2f c "+
			"%.2f %.2f l "+
			"%.2f %.2f %.2f %.2f %.2f %.2f c "+
			"%.2f %.2f l "+
			"%.2f %.2f %.2f %.2f %.2f %.2f c "+
			"%.2f %.2f l "+
			"%.2f %.2f %.2f %.2f %.2f %.2f c "+
			"h ",
		x+bl, y,
		x+w-br, y,
		x+w-br+brk, y, x+w, y+br-brk, x+w, y+br,
		x+w, y+h-tr,
		x+w, y+h-tr+trk, x+w-tr+trk, y+h, x+w-tr, y+h,
		x+tl, y+h,
		x+tl-tlk, y+h, x, y+h-tl+tlk, x, y+h-tl,
		x, y+bl,
		x, y+bl-blk, x+bl-blk, y, x+bl, y,
	)
}

func clampCornerRadius(r, w, h float64) float64 {
	if r < 0 {
		return 0
	}
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	return r
}

// drawBoxModel draws background and borders.
func drawBoxModel(ctx *layout.DrawContext, x, topY, width, height float64, bm layout.BoxModel) {
	bx := x + bm.MarginLeft
	by := topY - height + bm.MarginBottom
	bw := width - bm.MarginLeft - bm.MarginRight
	bh := height - bm.MarginTop - bm.MarginBottom
	if bw <= 0 || bh <= 0 {
		return
	}

	if bm.BoxShadow != "" {
		drawBoxShadow(ctx, bx, by, bw, bh, bm)
	}

	if bm.Background != nil {
		bg := bm.Background
		if hasAnyBoxRadius(bm) {
			ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %sf\n",
				bg[0], bg[1], bg[2], roundedRectCorners(
					bx, by, bw, bh,
					bm.BorderTopLeftRadius,
					bm.BorderTopRightRadius,
					bm.BorderBottomRightRadius,
					bm.BorderBottomLeftRadius,
				)))
		} else {
			ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
				bg[0], bg[1], bg[2], bx, by, bw, bh))
		}
	}
	if bm.BackgroundImage != "" {
		drawBackgroundImage(ctx, bx, by, bw, bh, bm)
	}
	if hasAnyBoxRadius(bm) && hasUniformVisibleBorder(bm) {
		bw := bm.BorderTopWidth
		if bm.BorderBottomWidth > bw {
			bw = bm.BorderBottomWidth
		}
		if bm.BorderLeftWidth > bw {
			bw = bm.BorderLeftWidth
		}
		if bm.BorderRightWidth > bw {
			bw = bm.BorderRightWidth
		}
		if bw > 0 {
			bx := x + bm.MarginLeft
			by := topY - height + bm.MarginBottom
			bWidth := width - bm.MarginLeft - bm.MarginRight
			bHeight := height - bm.MarginTop - bm.MarginBottom
			borderColor := resolvedBorderColor(bm.BorderTopColor, bm.BorderColor)
			ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %sS\n",
				borderColor[0], borderColor[1], borderColor[2],
				bw, roundedRectCorners(
					bx, by, bWidth, bHeight,
					bm.BorderTopLeftRadius,
					bm.BorderTopRightRadius,
					bm.BorderBottomRightRadius,
					bm.BorderBottomLeftRadius,
				)))
		}
		return
	}
	drawBoxSides(ctx, x, topY, width, height, bm)
}

func drawBackgroundImage(ctx *layout.DrawContext, x, y, width, height float64, bm layout.BoxModel) {
	image := strings.TrimSpace(bm.BackgroundImage)
	if image == "" {
		return
	}
	if drawLinearGradientBackground(ctx, x, y, width, height, bm, image) {
		return
	}
}

func drawLinearGradientBackground(ctx *layout.DrawContext, x, y, width, height float64, bm layout.BoxModel, image string) bool {
	layer := strings.TrimSpace(firstTopLevelCSSLayer(image))
	lower := strings.ToLower(layer)
	if !strings.HasPrefix(lower, "linear-gradient(") {
		return false
	}
	inner := strings.TrimSpace(layer[len("linear-gradient("):])
	if strings.HasSuffix(inner, ")") {
		inner = inner[:len(inner)-1]
	}
	args := splitTopLevelCSV(inner)
	if len(args) < 2 {
		return false
	}

	startIdx := 0
	direction := "vertical"
	if looksLikeGradientDirection(args[0]) {
		direction = parseGradientDirection(args[0])
		startIdx = 1
	}
	if len(args[startIdx:]) < 2 {
		return false
	}

	startColor, ok := extractGradientColor(args[startIdx])
	if !ok {
		return false
	}
	endColor, ok := extractGradientColor(args[len(args)-1])
	if !ok {
		return false
	}

	if hasAnyBoxRadius(bm) {
		ctx.WriteString("q\n")
		ctx.WriteString(fmt.Sprintf("%sW n\n", roundedRectCorners(
			x, y, width, height,
			bm.BorderTopLeftRadius,
			bm.BorderTopRightRadius,
			bm.BorderBottomRightRadius,
			bm.BorderBottomLeftRadius,
		)))
		defer ctx.WriteString("Q\n")
	}

	steps := 24
	switch direction {
	case "horizontal":
		stepWidth := width / float64(steps)
		for i := 0; i < steps; i++ {
			t := float64(i) / float64(steps-1)
			c := interpolateColor(startColor, endColor, t)
			segX := x + stepWidth*float64(i)
			segW := stepWidth
			if i == steps-1 {
				segW = x + width - segX
			}
			ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
				c[0], c[1], c[2], segX, y, segW, height))
		}
	default:
		stepHeight := height / float64(steps)
		for i := 0; i < steps; i++ {
			t := float64(i) / float64(steps-1)
			c := interpolateColor(startColor, endColor, t)
			segY := y + stepHeight*float64(i)
			segH := stepHeight
			if i == steps-1 {
				segH = y + height - segY
			}
			ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
				c[0], c[1], c[2], x, segY, width, segH))
		}
	}

	return true
}

func drawBoxShadow(ctx *layout.DrawContext, x, y, width, height float64, bm layout.BoxModel) {
	offsetX, offsetY, blur, spread, color, ok := parseBoxShadow(bm.BoxShadow)
	if !ok {
		return
	}
	expand := spread + blur*0.5
	shadowX := x + offsetX - expand
	shadowY := y - offsetY - expand
	shadowW := width + expand*2
	shadowH := height + expand*2
	if shadowW <= 0 || shadowH <= 0 {
		return
	}
	if hasAnyBoxRadius(bm) {
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %sf\n",
			color[0], color[1], color[2], roundedRectCorners(
				shadowX, shadowY, shadowW, shadowH,
				bm.BorderTopLeftRadius+expand,
				bm.BorderTopRightRadius+expand,
				bm.BorderBottomRightRadius+expand,
				bm.BorderBottomLeftRadius+expand,
			)))
		return
	}
	ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
		color[0], color[1], color[2], shadowX, shadowY, shadowW, shadowH))
}

func drawBoxSides(ctx *layout.DrawContext, x, topY, width, height float64, bm layout.BoxModel) {
	if bm.BorderTopWidth > 0 {
		borderColor := resolvedBorderColor(bm.BorderTopColor, bm.BorderColor)
		y := topY - bm.MarginTop
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
			borderColor[0], borderColor[1], borderColor[2],
			bm.BorderTopWidth, x+bm.MarginLeft, y, x+width-bm.MarginRight, y))
	}
	if bm.BorderBottomWidth > 0 {
		borderColor := resolvedBorderColor(bm.BorderBottomColor, bm.BorderColor)
		y := topY - height + bm.MarginBottom
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
			borderColor[0], borderColor[1], borderColor[2],
			bm.BorderBottomWidth, x+bm.MarginLeft, y, x+width-bm.MarginRight, y))
	}
	if bm.BorderLeftWidth > 0 {
		borderColor := resolvedBorderColor(bm.BorderLeftColor, bm.BorderColor)
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
			borderColor[0], borderColor[1], borderColor[2],
			bm.BorderLeftWidth, x+bm.MarginLeft, topY-bm.MarginTop, x+bm.MarginLeft, topY-height+bm.MarginBottom))
	}
	if bm.BorderRightWidth > 0 {
		borderColor := resolvedBorderColor(bm.BorderRightColor, bm.BorderColor)
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
			borderColor[0], borderColor[1], borderColor[2],
			bm.BorderRightWidth, x+width-bm.MarginRight, topY-bm.MarginTop, x+width-bm.MarginRight, topY-height+bm.MarginBottom))
	}
}

func shouldClipBoxChildren(style *ComputedStyle, bm layout.BoxModel) bool {
	if style == nil || style.Overflow != "hidden" {
		return false
	}
	return hasAnyBoxRadius(bm) || bm.PaddingTop > 0 || bm.PaddingRight > 0 || bm.PaddingBottom > 0 || bm.PaddingLeft > 0
}

func beginBoxClip(ctx *layout.DrawContext, x, topY, width, height float64, bm layout.BoxModel) {
	ctx.WriteString("q\n")
	bx := x + bm.MarginLeft
	by := topY - height + bm.MarginBottom
	bw := width - bm.MarginLeft - bm.MarginRight
	bh := height - bm.MarginTop - bm.MarginBottom
	if bw <= 0 || bh <= 0 {
		return
	}
	if hasAnyBoxRadius(bm) {
		ctx.WriteString(fmt.Sprintf("%sW n\n", roundedRectCorners(
			bx, by, bw, bh,
			bm.BorderTopLeftRadius,
			bm.BorderTopRightRadius,
			bm.BorderBottomRightRadius,
			bm.BorderBottomLeftRadius,
		)))
		return
	}
	ctx.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f re W n\n", bx, by, bw, bh))
}

func hasUniformVisibleBorder(bm layout.BoxModel) bool {
	if bm.BorderTopWidth <= 0 {
		return false
	}
	return bm.BorderTopWidth == bm.BorderRightWidth &&
		bm.BorderTopWidth == bm.BorderBottomWidth &&
		bm.BorderTopWidth == bm.BorderLeftWidth &&
		resolvedBorderColor(bm.BorderTopColor, bm.BorderColor) == resolvedBorderColor(bm.BorderRightColor, bm.BorderColor) &&
		resolvedBorderColor(bm.BorderTopColor, bm.BorderColor) == resolvedBorderColor(bm.BorderBottomColor, bm.BorderColor) &&
		resolvedBorderColor(bm.BorderTopColor, bm.BorderColor) == resolvedBorderColor(bm.BorderLeftColor, bm.BorderColor)
}

func resolvedBorderColor(sideColor, fallback [3]float64) [3]float64 {
	if sideColor != ([3]float64{}) {
		return sideColor
	}
	return fallback
}

func hasAnyBoxRadius(bm layout.BoxModel) bool {
	return bm.BorderRadius > 0 ||
		bm.BorderTopLeftRadius > 0 ||
		bm.BorderTopRightRadius > 0 ||
		bm.BorderBottomRightRadius > 0 ||
		bm.BorderBottomLeftRadius > 0
}

func firstTopLevelCSSLayer(value string) string {
	layers := splitTopLevelCSV(value)
	if len(layers) == 0 {
		return value
	}
	return layers[0]
}

func splitTopLevelCSV(value string) []string {
	var parts []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(value); i++ {
		if quote != 0 {
			if value[i] == '\\' && i+1 < len(value) {
				i++
				continue
			}
			if value[i] == quote {
				quote = 0
			}
			continue
		}
		switch value[i] {
		case '"', '\'':
			quote = value[i]
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				part := strings.TrimSpace(value[start:i])
				if part != "" {
					parts = append(parts, part)
				}
				start = i + 1
			}
		}
	}
	if start < len(value) {
		part := strings.TrimSpace(value[start:])
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func looksLikeGradientDirection(token string) bool {
	token = strings.TrimSpace(strings.ToLower(token))
	return strings.HasPrefix(token, "to ") ||
		strings.HasSuffix(token, "deg") ||
		strings.HasSuffix(token, "turn") ||
		strings.HasSuffix(token, "rad")
}

func parseGradientDirection(token string) string {
	token = strings.TrimSpace(strings.ToLower(token))
	if strings.Contains(token, "right") || strings.Contains(token, "left") {
		return "horizontal"
	}
	if strings.HasSuffix(token, "deg") {
		if degrees, err := strconv.ParseFloat(strings.TrimSuffix(token, "deg"), 64); err == nil {
			degrees = math.Mod(degrees, 360)
			if degrees < 0 {
				degrees += 360
			}
			if (degrees >= 45 && degrees <= 135) || (degrees >= 225 && degrees <= 315) {
				return "horizontal"
			}
		}
	}
	return "vertical"
}

func extractGradientColor(stop string) ([3]float64, bool) {
	parts := splitCSSValues(stop)
	for _, part := range parts {
		if c, ok := parseColor(part); ok {
			return c, true
		}
	}
	return [3]float64{}, false
}

func interpolateColor(start, end [3]float64, t float64) [3]float64 {
	return [3]float64{
		start[0] + (end[0]-start[0])*t,
		start[1] + (end[1]-start[1])*t,
		start[2] + (end[2]-start[2])*t,
	}
}

func parseBoxShadow(value string) (float64, float64, float64, float64, [3]float64, bool) {
	layer := strings.TrimSpace(firstTopLevelCSSLayer(value))
	if layer == "" || strings.EqualFold(layer, "none") {
		return 0, 0, 0, 0, [3]float64{}, false
	}
	parts := splitCSSValues(layer)
	if len(parts) < 2 {
		return 0, 0, 0, 0, [3]float64{}, false
	}
	color := [3]float64{0.75, 0.75, 0.75}
	var lengths []float64
	for _, part := range parts {
		lower := strings.ToLower(part)
		if lower == "inset" {
			return 0, 0, 0, 0, [3]float64{}, false
		}
		if c, ok := parseColor(part); ok {
			color = c
			continue
		}
		length := parseLength(part)
		if length.Unit != "" || length.Value != 0 || part == "0" || strings.HasPrefix(part, ".") || strings.HasPrefix(part, "-") {
			lengths = append(lengths, length.ToPoints(12, 12))
		}
	}
	if len(lengths) < 2 {
		return 0, 0, 0, 0, [3]float64{}, false
	}
	blur := 0.0
	spread := 0.0
	if len(lengths) > 2 {
		blur = lengths[2]
	}
	if len(lengths) > 3 {
		spread = lengths[3]
	}
	return lengths[0], lengths[1], blur, spread, color, true
}
