package converter

import (
	"math"
	"sort"
)

// detectTables uses text alignment heuristics to find tables in a set of lines.
func detectTables(lines []Line) []DetectedTable {
	if len(lines) < 3 {
		return nil
	}

	// Step 1: For each line, collect the X positions of span starts.
	type lineColumns struct {
		lineIdx int
		xPositions []float64
	}

	var lineColData []lineColumns
	for i, line := range lines {
		xs := spanXPositions(line.Spans)
		if len(xs) >= 2 {
			lineColData = append(lineColData, lineColumns{lineIdx: i, xPositions: xs})
		}
	}

	if len(lineColData) < 3 {
		return nil
	}

	// Step 2: Find consecutive lines with similar column structure.
	var tables []DetectedTable
	i := 0
	for i < len(lineColData) {
		// Try to start a table at this line.
		refCols := lineColData[i].xPositions
		tableLines := []int{lineColData[i].lineIdx}

		j := i + 1
		for j < len(lineColData) {
			if columnsMatch(refCols, lineColData[j].xPositions, 5.0) {
				tableLines = append(tableLines, lineColData[j].lineIdx)
				j++
			} else {
				break
			}
		}

		if len(tableLines) >= 3 {
			// Found a table!
			table := buildTable(lines, tableLines, refCols)
			tables = append(tables, table)
		}

		if j == i+1 {
			i++
		} else {
			i = j
		}
	}

	return tables
}

// spanXPositions extracts unique sorted X positions from spans.
func spanXPositions(spans []StyledSpan) []float64 {
	seen := make(map[float64]bool)
	var xs []float64
	for _, s := range spans {
		// Round to avoid float precision issues.
		x := math.Round(s.X*10) / 10
		if !seen[x] {
			seen[x] = true
			xs = append(xs, x)
		}
	}
	sort.Float64s(xs)
	return xs
}

// columnsMatch checks if two sets of column positions are similar within tolerance.
func columnsMatch(a, b []float64, tolerance float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > tolerance {
			return false
		}
	}
	return true
}

// buildTable constructs a DetectedTable from the identified lines and columns.
func buildTable(allLines []Line, tableLineIndices []int, columnPositions []float64) DetectedTable {
	numRows := len(tableLineIndices)
	numCols := len(columnPositions)

	cells := make([][]TableCell, numRows)
	for r, lineIdx := range tableLineIndices {
		cells[r] = make([]TableCell, numCols)
		line := allLines[lineIdx]

		for c := 0; c < numCols; c++ {
			cells[r][c] = TableCell{
				Row:     r,
				Col:     c,
				RowSpan: 1,
				ColSpan: 1,
			}

			// Determine column boundaries.
			colStart := columnPositions[c]
			colEnd := math.MaxFloat64
			if c+1 < numCols {
				colEnd = columnPositions[c+1]
			}

			// Assign spans to this cell.
			for _, span := range line.Spans {
				sx := math.Round(span.X*10) / 10
				if sx >= colStart && sx < colEnd {
					cells[r][c].Spans = append(cells[r][c].Spans, span)
					if cells[r][c].Text != "" {
						cells[r][c].Text += " "
					}
					cells[r][c].Text += span.Text
				}
			}
		}
	}

	// Compute bounding rect.
	var rect [4]float64
	if numRows > 0 && numCols > 0 {
		rect[0] = columnPositions[0]
		rect[2] = columnPositions[numCols-1] + 100 // approximate
		rect[1] = allLines[tableLineIndices[0]].Y
		rect[3] = allLines[tableLineIndices[numRows-1]].Y
	}

	return DetectedTable{
		Rows:  numRows,
		Cols:  numCols,
		Cells: cells,
		Rect:  rect,
	}
}
