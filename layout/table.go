package layout

import "fmt"

// Table is a table layout element.
type Table struct {
	Columns     int
	Rows        []TableRow
	ColWidths   []float64 // explicit column widths (0 = auto)
	HeaderRows  int       // number of rows to repeat on each page
	Box         BoxModel
	CellPadding float64
	BorderWidth float64
	BorderColor [3]float64
	HeaderBg    *[3]float64
	StripedBg   *[3]float64
}

// TableRow is a row in a table.
type TableRow struct {
	Cells []TableCell
}

// TableCell is a cell in a table row.
type TableCell struct {
	Content Element
	ColSpan int
	RowSpan int
	Align   Alignment
	VAlign  VAlignment
	Bg      *[3]float64
}

// VAlignment is vertical alignment.
type VAlignment int

const (
	VAlignTop VAlignment = iota
	VAlignMiddle
	VAlignBottom
)

// NewTable creates a new table with the given number of columns.
func NewTable(columns int) *Table {
	return &Table{
		Columns:     columns,
		CellPadding: 4,
		BorderWidth: 0.5,
		BorderColor: [3]float64{0, 0, 0},
	}
}

// AddHeader adds a header row with string cells.
func (t *Table) AddHeader(cells ...string) *Table {
	row := TableRow{}
	for _, text := range cells {
		row.Cells = append(row.Cells, TableCell{
			Content: NewParagraph(text),
			ColSpan: 1,
			RowSpan: 1,
		})
	}
	// Insert at position HeaderRows (append to header section)
	t.Rows = append(t.Rows[:t.HeaderRows], append([]TableRow{row}, t.Rows[t.HeaderRows:]...)...)
	t.HeaderRows++
	return t
}

// AddRow adds a data row with string cells.
func (t *Table) AddRow(cells ...string) *Table {
	row := TableRow{}
	for _, text := range cells {
		row.Cells = append(row.Cells, TableCell{
			Content: NewParagraph(text),
			ColSpan: 1,
			RowSpan: 1,
		})
	}
	t.Rows = append(t.Rows, row)
	return t
}

// AddElementRow adds a row with Element cells.
func (t *Table) AddElementRow(cells ...Element) *Table {
	row := TableRow{}
	for _, el := range cells {
		row.Cells = append(row.Cells, TableCell{
			Content: el,
			ColSpan: 1,
			RowSpan: 1,
		})
	}
	t.Rows = append(t.Rows, row)
	return t
}

// SetColumnWidths sets explicit column widths.
func (t *Table) SetColumnWidths(widths ...float64) *Table {
	t.ColWidths = widths
	return t
}

// resolveColWidths computes column widths for the available width.
func (t *Table) resolveColWidths(availWidth float64) []float64 {
	widths := make([]float64, t.Columns)
	totalFixed := 0.0
	autoCount := 0

	for i := 0; i < t.Columns; i++ {
		if i < len(t.ColWidths) && t.ColWidths[i] > 0 {
			widths[i] = t.ColWidths[i]
			totalFixed += widths[i]
		} else {
			autoCount++
		}
	}

	if autoCount > 0 {
		autoWidth := (availWidth - totalFixed) / float64(autoCount)
		if autoWidth < 0 {
			autoWidth = 0
		}
		for i := 0; i < t.Columns; i++ {
			if widths[i] == 0 {
				widths[i] = autoWidth
			}
		}
	}

	return widths
}

// measureRowHeight measures the height of a row given column widths.
func (t *Table) measureRowHeight(row TableRow, colWidths []float64) float64 {
	maxH := 0.0
	pad := t.CellPadding * 2

	for i, cell := range row.Cells {
		if i >= t.Columns {
			break
		}
		cellW := colWidths[i] - pad
		if cell.ColSpan > 1 {
			for j := 1; j < cell.ColSpan && i+j < len(colWidths); j++ {
				cellW += colWidths[i+j]
			}
		}
		if cellW < 0 {
			cellW = 0
		}
		if cell.Content != nil {
			plan := cell.Content.PlanLayout(LayoutArea{Width: cellW, Height: 1e6})
			h := plan.Consumed + pad
			if h > maxH {
				maxH = h
			}
		}
	}

	if maxH == 0 {
		maxH = t.CellPadding*2 + 12 // minimum row height
	}
	return maxH
}

// PlanLayout implements Element.
func (t *Table) PlanLayout(area LayoutArea) LayoutPlan {
	contentWidth := area.Width - t.Box.TotalHorizontal()
	if contentWidth < 0 {
		contentWidth = 0
	}

	colWidths := t.resolveColWidths(contentWidth)

	// Measure all row heights
	type rowInfo struct {
		height float64
		row    TableRow
		idx    int
	}
	var rows []rowInfo
	for i, row := range t.Rows {
		h := t.measureRowHeight(row, colWidths)
		rows = append(rows, rowInfo{height: h, row: row, idx: i})
	}

	// Measure header height
	headerHeight := 0.0
	for i := 0; i < t.HeaderRows && i < len(rows); i++ {
		headerHeight += rows[i].height
	}

	// Place rows
	var blocks []PlacedBlock
	curY := t.Box.ContentTop()
	remaining := area.Height - t.Box.TotalVertical()

	for ri, rInfo := range rows {
		if rInfo.height > remaining {
			// Can't fit this row — create overflow with remaining rows
			if ri <= t.HeaderRows {
				// Can't even fit headers + one row
				if ri == 0 {
					return LayoutPlan{Status: LayoutNothing}
				}
			}

			consumed := curY + t.Box.PaddingBottom + t.Box.BorderBottomWidth + t.Box.MarginBottom
			overflowTable := &Table{
				Columns:     t.Columns,
				ColWidths:   t.ColWidths,
				HeaderRows:  t.HeaderRows,
				Box:         t.Box,
				CellPadding: t.CellPadding,
				BorderWidth: t.BorderWidth,
				BorderColor: t.BorderColor,
				HeaderBg:    t.HeaderBg,
				StripedBg:   t.StripedBg,
			}
			// Add header rows + remaining data rows
			for i := 0; i < t.HeaderRows && i < len(t.Rows); i++ {
				overflowTable.Rows = append(overflowTable.Rows, t.Rows[i])
			}
			for i := ri; i < len(t.Rows); i++ {
				overflowTable.Rows = append(overflowTable.Rows, t.Rows[i])
			}

			return LayoutPlan{
				Status:   LayoutPartial,
				Consumed: consumed,
				Blocks:   blocks,
				Overflow: overflowTable,
			}
		}

		// Place this row
		rowBlocks := t.placeRow(rInfo.row, rInfo.idx, colWidths, rInfo.height, curY)
		blocks = append(blocks, rowBlocks...)
		curY += rInfo.height
		remaining -= rInfo.height
	}

	consumed := curY + t.Box.PaddingBottom + t.Box.BorderBottomWidth + t.Box.MarginBottom
	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: consumed,
		Blocks:   blocks,
	}
}

func (t *Table) placeRow(row TableRow, rowIdx int, colWidths []float64, rowHeight float64, y float64) []PlacedBlock {
	var blocks []PlacedBlock
	cellX := t.Box.ContentLeft()

	for i, cell := range row.Cells {
		if i >= t.Columns {
			break
		}
		cellW := colWidths[i]
		if cell.ColSpan > 1 {
			for j := 1; j < cell.ColSpan && i+j < len(colWidths); j++ {
				cellW += colWidths[i+j]
			}
		}

		// Determine background
		var bg *[3]float64
		if cell.Bg != nil {
			bg = cell.Bg
		} else if rowIdx < t.HeaderRows && t.HeaderBg != nil {
			bg = t.HeaderBg
		} else if t.StripedBg != nil && (rowIdx-t.HeaderRows)%2 == 1 {
			bg = t.StripedBg
		}

		localCellW := cellW
		localRowH := rowHeight
		localBg := bg
		borderW := t.BorderWidth
		borderC := t.BorderColor
		pad := t.CellPadding

		// Cell background and border block
		cellBlock := PlacedBlock{
			X: cellX, Y: y,
			Width: localCellW, Height: localRowH,
			Tag: "TD",
			Draw: func(ctx *DrawContext, x, pdfY float64) {
				// Background
				if localBg != nil {
					ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", localBg[0], localBg[1], localBg[2]))
					ctx.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f re f\n", x, pdfY-localRowH, localCellW, localRowH))
				}
				// Border
				if borderW > 0 {
					ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f RG\n%.2f w\n", borderC[0], borderC[1], borderC[2], borderW))
					ctx.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f re S\n", x, pdfY-localRowH, localCellW, localRowH))
				}
			},
		}

		// Layout cell content
		if cell.Content != nil {
			contentW := cellW - pad*2
			if contentW < 0 {
				contentW = 0
			}
			plan := cell.Content.PlanLayout(LayoutArea{Width: contentW, Height: rowHeight - pad*2})
			for _, b := range plan.Blocks {
				b.X += cellX + pad
				b.Y += y + pad
				cellBlock.Children = append(cellBlock.Children, b)
			}
		}

		blocks = append(blocks, cellBlock)
		cellX += cellW
	}

	return blocks
}
