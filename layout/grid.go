package layout

// Grid is a CSS Grid layout element.
type Grid struct {
	Columns   []GridTrack
	Rows      []GridTrack
	Gap       float64
	ColumnGap float64
	RowGap    float64
	Children  []GridChild
	Box       BoxModel
}

// GridTrack defines a grid track size.
type GridTrack struct {
	Size    float64 // fixed size in points
	Fr      float64 // fractional unit
	MinSize float64
	MaxSize float64
	Auto    bool
}

// GridChild is a child in a grid.
type GridChild struct {
	Element  Element
	ColStart int
	ColEnd   int
	RowStart int
	RowEnd   int
}

// NewGrid creates a grid with column definitions.
func NewGrid(columns []GridTrack, children ...Element) *Grid {
	g := &Grid{Columns: columns}
	for i, el := range children {
		g.Children = append(g.Children, GridChild{
			Element:  el,
			ColStart: i % len(columns),
			ColEnd:   i%len(columns) + 1,
			RowStart: i / len(columns),
			RowEnd:   i/len(columns) + 1,
		})
	}
	return g
}

// Fr creates a fractional grid track.
func Fr(n float64) GridTrack {
	return GridTrack{Fr: n}
}

// Px creates a fixed-size grid track.
func Px(n float64) GridTrack {
	return GridTrack{Size: n}
}

// Auto creates an auto-sized grid track.
func Auto() GridTrack {
	return GridTrack{Auto: true}
}

func (g *Grid) resolveColumnWidths(avail float64) []float64 {
	nCols := len(g.Columns)
	if nCols == 0 {
		return nil
	}

	colGap := g.ColumnGap
	if colGap == 0 {
		colGap = g.Gap
	}
	totalGaps := colGap * float64(nCols-1)
	availForTracks := avail - totalGaps

	widths := make([]float64, nCols)
	totalFixed := 0.0
	totalFr := 0.0
	autoCount := 0

	for i, t := range g.Columns {
		if t.Size > 0 {
			widths[i] = t.Size
			totalFixed += t.Size
		} else if t.Fr > 0 {
			totalFr += t.Fr
		} else if t.Auto {
			autoCount++
		}
	}

	remaining := availForTracks - totalFixed
	if remaining < 0 {
		remaining = 0
	}

	// Auto tracks get equal share of what's left after fr
	autoShare := 0.0
	if autoCount > 0 && totalFr == 0 {
		autoShare = remaining / float64(autoCount)
	} else if autoCount > 0 {
		autoShare = remaining * 0.2 / float64(autoCount) // auto gets 20%
		remaining *= 0.8
	}

	for i, t := range g.Columns {
		if t.Fr > 0 && totalFr > 0 {
			widths[i] = remaining * (t.Fr / totalFr)
		} else if t.Auto {
			widths[i] = autoShare
		}
		if t.MinSize > 0 && widths[i] < t.MinSize {
			widths[i] = t.MinSize
		}
		if t.MaxSize > 0 && widths[i] > t.MaxSize {
			widths[i] = t.MaxSize
		}
	}

	return widths
}

func (g *Grid) maxRow() int {
	maxR := 0
	for _, ch := range g.Children {
		if ch.RowEnd > maxR {
			maxR = ch.RowEnd
		}
	}
	return maxR
}

// PlanLayout implements Element.
func (g *Grid) PlanLayout(area LayoutArea) LayoutPlan {
	contentW := area.Width - g.Box.TotalHorizontal()
	contentH := area.Height - g.Box.TotalVertical()
	if contentW < 0 {
		contentW = 0
	}
	if contentH < 0 {
		return LayoutPlan{Status: LayoutNothing}
	}

	colWidths := g.resolveColumnWidths(contentW)
	nCols := len(colWidths)
	if nCols == 0 {
		return LayoutPlan{Status: LayoutFull, Consumed: g.Box.TotalVertical()}
	}

	colGap := g.ColumnGap
	if colGap == 0 {
		colGap = g.Gap
	}
	rowGap := g.RowGap
	if rowGap == 0 {
		rowGap = g.Gap
	}

	nRows := g.maxRow()
	if nRows == 0 {
		return LayoutPlan{Status: LayoutFull, Consumed: g.Box.TotalVertical()}
	}

	// Compute column X positions
	colX := make([]float64, nCols)
	x := 0.0
	for i := range colWidths {
		colX[i] = x
		x += colWidths[i] + colGap
	}

	// Measure row heights
	rowHeights := make([]float64, nRows)
	for _, ch := range g.Children {
		if ch.Element == nil {
			continue
		}
		w := 0.0
		for c := ch.ColStart; c < ch.ColEnd && c < nCols; c++ {
			w += colWidths[c]
			if c > ch.ColStart {
				w += colGap
			}
		}
		plan := ch.Element.PlanLayout(LayoutArea{Width: w, Height: 1e6})
		spanRows := ch.RowEnd - ch.RowStart
		if spanRows <= 0 {
			spanRows = 1
		}
		perRow := plan.Consumed / float64(spanRows)
		for r := ch.RowStart; r < ch.RowEnd && r < nRows; r++ {
			if perRow > rowHeights[r] {
				rowHeights[r] = perRow
			}
		}
	}

	// Compute row Y positions
	rowY := make([]float64, nRows)
	y := 0.0
	for i := range rowHeights {
		rowY[i] = y
		y += rowHeights[i] + rowGap
	}
	totalHeight := y - rowGap // remove last gap
	if totalHeight < 0 {
		totalHeight = 0
	}

	// Place children
	var blocks []PlacedBlock
	for _, ch := range g.Children {
		if ch.Element == nil {
			continue
		}
		cx := colX[ch.ColStart]
		cy := rowY[ch.RowStart]
		w := 0.0
		for c := ch.ColStart; c < ch.ColEnd && c < nCols; c++ {
			w += colWidths[c]
			if c > ch.ColStart {
				w += colGap
			}
		}
		h := 0.0
		for r := ch.RowStart; r < ch.RowEnd && r < nRows; r++ {
			h += rowHeights[r]
			if r > ch.RowStart {
				h += rowGap
			}
		}

		plan := ch.Element.PlanLayout(LayoutArea{Width: w, Height: h})
		for _, b := range plan.Blocks {
			b.X += g.Box.ContentLeft() + cx
			b.Y += g.Box.ContentTop() + cy
			blocks = append(blocks, b)
		}
	}

	consumed := g.Box.ContentTop() + totalHeight + g.Box.PaddingBottom + g.Box.BorderBottomWidth + g.Box.MarginBottom
	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: consumed,
		Blocks:   blocks,
	}
}
