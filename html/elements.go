package html

import (
	"fmt"
	"math"
	"strings"

	pdfimage "github.com/oarkflow/pdf/image"
	"github.com/oarkflow/pdf/layout"
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

func (e *DivElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	bm := e.BoxModel
	innerWidth := area.Width - bm.TotalHorizontal()
	if innerWidth <= 0 {
		innerWidth = area.Width
	}

	var childBlocks []layout.PlacedBlock
	consumed := 0.0

	for _, child := range e.Children {
		childPlan := child.PlanLayout(layout.LayoutArea{Width: innerWidth, Height: math.Max(area.Height, 5000)})
		for _, b := range childPlan.Blocks {
			b.X += bm.ContentLeft()
			b.Y += bm.ContentTop() + consumed
			childBlocks = append(childBlocks, b)
		}
		consumed += childPlan.Consumed
	}

	totalHeight := consumed + bm.TotalVertical()
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
	allBlocks = append(allBlocks, childBlocks...)

	return layout.LayoutPlan{
		Status:   layout.LayoutFull,
		Consumed: totalHeight,
		Blocks:   allBlocks,
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

				// Header row background
				if len(cRow.Cells) > 0 && cRow.Cells[0].IsHeader {
					bg := [3]float64{0.243, 0.243, 0.322}
					if cRow.Style != nil && cRow.Style.BackgroundColor != nil {
						bg = *cRow.Style.BackgroundColor
					} else if len(cRow.Cells) > 0 && cRow.Cells[0].Style != nil && cRow.Cells[0].Style.BackgroundColor != nil {
						bg = *cRow.Cells[0].Style.BackgroundColor
					}
					ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
						bg[0], bg[1], bg[2], xOff, topY-cRowHeight, cTableWidth, cRowHeight))
				} else if cRowIdx%2 == 0 {
					bg := [3]float64{0.973, 0.976, 0.984}
					if cRow.Style != nil && cRow.Style.BackgroundColor != nil {
						bg = *cRow.Style.BackgroundColor
					}
					ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
						bg[0], bg[1], bg[2], xOff, topY-cRowHeight, cTableWidth, cRowHeight))
				}

				cellX := xOff
				for ci, cell := range cRow.Cells {
					cellWidth := cRowLayout.cells[ci].width
					fs := cRowLayout.cells[ci].fs
					lh := cRowLayout.cells[ci].lh
					lines := cRowLayout.cells[ci].lines
					contentW := cellWidth - cCellPad*2

					fn := resolveFontName(fs, cell.IsHeader, false)
					ensureFont(ctx, fn)

					// Determine text color
					if cell.IsHeader {
						tc := [3]float64{1, 1, 1}
						if cell.Style != nil && (cell.Style.Color != [3]float64{}) {
							tc = cell.Style.Color
						}
						ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", tc[0], tc[1], tc[2]))
					} else {
						tc := [3]float64{0.235, 0.263, 0.341}
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
							runFn := resolveFontName(runFS, run.Bold || cell.IsHeader, run.Italic)
							ensureFont(ctx, runFn)

							if run.Color != ([3]float64{}) {
								ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", run.Color[0], run.Color[1], run.Color[2]))
							}

							text := toWinAnsi(run.Text)
							ctx.WriteString(fmt.Sprintf("BT\n/%s %.1f Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
								runFn, runFS, lineX, textY, escPDF(text)))
							lineX += measureStr(run.Text, runFS, run.Bold || cell.IsHeader, run.FontName)
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

	if e.Fetcher == nil || e.Src == "" {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	data, err := e.Fetcher.Fetch(e.Src)
	if err != nil {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	decoded, err := pdfimage.Load(data)
	if err != nil {
		return layout.LayoutPlan{Status: layout.LayoutFull}
	}

	img := &layout.ImageElement{
		Source: data,
		Image: layout.ImageEntry{
			Image: decoded,
		},
		OrigW: decoded.Width,
		OrigH: decoded.Height,
		Alt:   e.Alt,
	}
	if e.Style != nil && !e.Style.Width.IsAuto() && e.Style.Width.Value > 0 {
		img.Width = e.Style.Width.ToPoints(contentWidth, 12)
	}
	if e.Style != nil && !e.Style.Height.IsAuto() && e.Style.Height.Value > 0 {
		img.Height = e.Style.Height.ToPoints(contentHeight, 12)
	}

	plan := img.PlanLayout(layout.LayoutArea{Width: contentWidth, Height: contentHeight})
	for i := range plan.Blocks {
		plan.Blocks[i].X += bm.ContentLeft()
		plan.Blocks[i].Y += bm.ContentTop()
	}
	plan.Consumed += bm.TotalVertical()
	return plan
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

	// Split into words
	type word struct {
		chars []cr
		width float64
	}
	var words []word
	var cur []cr
	for _, c := range chars {
		if c.ch == ' ' || c.ch == '\t' || c.ch == '\n' {
			if len(cur) > 0 {
				words = append(words, word{cur, measureCR(cur)})
				cur = nil
			}
			if c.ch == '\n' {
				words = append(words, word{nil, -1}) // line break
			}
			continue
		}
		cur = append(cur, c)
	}
	if len(cur) > 0 {
		words = append(words, word{cur, measureCR(cur)})
	}
	if len(words) == 0 {
		return nil
	}

	var lines []wrappedLine
	var lineChars []cr
	lineW := 0.0
	spaceW := defaultFontSize * 0.25

	for _, w := range words {
		if w.width < 0 { // forced line break
			if len(lineChars) > 0 {
				lines = append(lines, buildLine(lineChars, lineW))
			} else {
				lines = append(lines, wrappedLine{})
			}
			lineChars = nil
			lineW = 0
			continue
		}
		needed := w.width
		if len(lineChars) > 0 {
			needed += spaceW
		}
		if len(lineChars) > 0 && lineW+needed > maxWidth {
			lines = append(lines, buildLine(lineChars, lineW))
			lineChars = nil
			lineW = 0
		}
		if w.width > maxWidth {
			for len(w.chars) > 0 {
				chunk, chunkW, rest := takeCharsFitting(w.chars, maxWidth)
				if len(chunk) == 0 {
					chunk = w.chars[:1]
					chunkW = measureCR(chunk)
					rest = w.chars[1:]
				}
				lineChars = append(lineChars, chunk...)
				lineW += chunkW
				w.chars = rest
				if len(w.chars) > 0 {
					lines = append(lines, buildLine(lineChars, lineW))
					lineChars = nil
					lineW = 0
				}
			}
			continue
		}
		if len(lineChars) > 0 {
			lineChars = append(lineChars, cr{' ', w.chars[0].run})
			lineW += spaceW
		}
		lineChars = append(lineChars, w.chars...)
		lineW += w.width
	}
	if len(lineChars) > 0 {
		lines = append(lines, buildLine(lineChars, lineW))
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
			r.Color != curRun.Color || r.Underline != curRun.Underline {
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
					fn := resolveFontName(fs, run.Bold, run.Italic)
					ensureFont(ctx, fn)

					c := color
					if run.Color != [3]float64{} {
						c = run.Color
					}

					text := toWinAnsi(run.Text)
					ctx.WriteString(fmt.Sprintf("BT\n/%s %.1f Tf\n%.3f %.3f %.3f rg\n%.2f %.2f Td\n(%s) Tj\nET\n",
						fn, fs, c[0], c[1], c[2], lineX, textY, escPDF(text)))
					lineX += measureStr(run.Text, fs, run.Bold, run.FontName)
				}
				textY -= cLH
			}
		},
	}
}

func drawTextRuns(ctx *layout.DrawContext, runs []layout.TextRun, x, topY, fontSize, lineHeight float64) {
	var parts []string
	bold := false
	italic := false
	color := [3]float64{0.2, 0.2, 0.2}
	for _, run := range runs {
		t := strings.TrimSpace(run.Text)
		if t == "" {
			continue
		}
		parts = append(parts, t)
		bold = bold || run.Bold
		italic = italic || run.Italic
		if run.Color != ([3]float64{}) {
			color = run.Color
		}
		if run.FontSize > 0 {
			fontSize = run.FontSize
		}
	}
	if len(parts) == 0 {
		return
	}
	text := toWinAnsi(strings.Join(parts, " "))
	fn := resolveFontName(fontSize, bold, italic)
	ensureFont(ctx, fn)
	textY := topY - fontSize
	ctx.WriteString(fmt.Sprintf("BT\n/%s %.1f Tf\n%.3f %.3f %.3f rg\n%.2f %.2f Td\n(%s) Tj\nET\n",
		fn, fontSize, color[0], color[1], color[2], x, textY, escPDF(text)))
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
			b.WriteRune(r)
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

// drawBoxModel draws background and borders.
func drawBoxModel(ctx *layout.DrawContext, x, topY, width, height float64, bm layout.BoxModel) {
	if bm.Background != nil {
		bg := bm.Background
		bx := x + bm.MarginLeft
		by := topY - height + bm.MarginBottom
		bw := width - bm.MarginLeft - bm.MarginRight
		bh := height - bm.MarginTop - bm.MarginBottom
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg %.2f %.2f %.2f %.2f re f\n",
			bg[0], bg[1], bg[2], bx, by, bw, bh))
	}
	if bm.BorderTopWidth > 0 {
		y := topY - bm.MarginTop
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
			bm.BorderColor[0], bm.BorderColor[1], bm.BorderColor[2],
			bm.BorderTopWidth, x+bm.MarginLeft, y, x+width-bm.MarginRight, y))
	}
	if bm.BorderBottomWidth > 0 {
		y := topY - height + bm.MarginBottom
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
			bm.BorderColor[0], bm.BorderColor[1], bm.BorderColor[2],
			bm.BorderBottomWidth, x+bm.MarginLeft, y, x+width-bm.MarginRight, y))
	}
	if bm.BorderLeftWidth > 0 {
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
			bm.BorderColor[0], bm.BorderColor[1], bm.BorderColor[2],
			bm.BorderLeftWidth, x+bm.MarginLeft, topY-bm.MarginTop, x+bm.MarginLeft, topY-height+bm.MarginBottom))
	}
	if bm.BorderRightWidth > 0 {
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG %.2f w %.2f %.2f m %.2f %.2f l S\n",
			bm.BorderColor[0], bm.BorderColor[1], bm.BorderColor[2],
			bm.BorderRightWidth, x+width-bm.MarginRight, topY-bm.MarginTop, x+width-bm.MarginRight, topY-height+bm.MarginBottom))
	}
}
