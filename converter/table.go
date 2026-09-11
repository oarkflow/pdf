package converter

import (
	"math"
	"slices"
	"sort"
	"strings"
)

// detectTables locates tables using text alignment heuristics, then
// reconstructs each one at the row level so that wrapped/multi-line cells
// (a description that spans two or three PDF lines, a right-aligned number
// column whose start X shifts slightly line to line, ...) still collapse
// into a single logical row instead of derailing detection.
//
// The approach:
//  1. Scan for a line with a "column grid" (2+ distinct span-start X
//     positions) to seed a candidate table block.
//  2. Grow the block through following lines that are visually contiguous
//     (small Y gap) and whose X positions are compatible with the seed grid
//     — including lines that only populate one or two of the columns, which
//     is exactly what a wrapped cell's continuation line looks like.
//  3. Re-group the block's lines into logical rows using the Y gap between
//     them: a small gap (roughly one line height) means "still the same
//     row, wrapped"; a larger gap starts a new row. This distinction is
//     reliable even though individual column X positions are not.
//  4. Assign every span in a row to the nearest seed column using midpoint
//     boundaries between columns (not the raw seed X itself), which
//     tolerates left-, right-, or center-aligned columns whose exact start
//     position drifts a little from line to line.
func detectTables(lines []Line) []DetectedTable {
	if len(lines) < 3 {
		return nil
	}

	var tables []DetectedTable
	i := 0
	for i < len(lines) {
		seeds := spanXPositions(lines[i].Spans)
		if len(seeds) < 2 {
			i++
			continue
		}

		blockEnd := i
		lastY := lines[i].Y
		j := i + 1
		for j < len(lines) {
			xs := spanXPositions(lines[j].Spans)
			if len(xs) == 0 || !xsCompatible(seeds, xs) {
				break
			}
			fontSize := avgFontSize(lines[j].Spans)
			if fontSize == 0 {
				fontSize = 12
			}
			if math.Abs(lastY-lines[j].Y) > fontSize*4.0 {
				break
			}
			blockEnd = j
			lastY = lines[j].Y
			if len(xs) > len(seeds) {
				// Adopt the most complete row seen so far as the column grid.
				seeds = xs
			}
			j++
		}

		if table := buildTableFromBlock(lines, i, blockEnd, seeds); table != nil {
			if !looksLikeList(table.Cells) {
				tables = append(tables, *table)
			}
		}

		if blockEnd == i {
			i++
		} else {
			i = blockEnd + 1
		}
	}

	return tables
}

// xsCompatible reports whether a line's X positions belong to the same
// column grid as seeds. Every position must land near some seed (loose
// tolerance, to absorb left/right/center alignment drift a wrapped or
// right-aligned column can have from line to line within the same cell).
// Beyond that:
//   - A line with only one X position is accepted only on a tight match — a
//     single coincidental loose match (e.g. ordinary prose starting a few
//     points from some column) isn't enough evidence it belongs to the table.
//   - A line with two or more X positions is accepted once at least two of
//     them land near distinct seed columns, since two independent columns
//     lining up by chance is not something ordinary prose does.
func xsCompatible(seeds, xs []float64) bool {
	const tightTol, looseTol = 5.0, 80.0
	distinctMatches := 0
	usedSeeds := make(map[int]bool)
	for _, x := range xs {
		si, d := nearestSeed(seeds, x)
		if d > looseTol {
			return false
		}
		if len(xs) == 1 {
			return d <= tightTol
		}
		if !usedSeeds[si] {
			usedSeeds[si] = true
			distinctMatches++
		}
	}
	return distinctMatches >= 2
}

func nearestSeed(seeds []float64, x float64) (index int, dist float64) {
	best := math.MaxFloat64
	bestIdx := 0
	for i, s := range seeds {
		if d := math.Abs(s - x); d < best {
			best = d
			bestIdx = i
		}
	}
	return bestIdx, best
}

// buildTableFromBlock reconstructs a DetectedTable from a contiguous run of
// lines[start..end] that share the given column grid, grouping wrapped
// physical lines into logical rows first.
func buildTableFromBlock(lines []Line, start, end int, seeds []float64) *DetectedTable {
	if end <= start {
		return nil
	}
	seeds = append([]float64(nil), seeds...)
	sort.Float64s(seeds)
	numCols := len(seeds)

	// Column boundaries sit between what neighboring columns actually use,
	// not at a single line's possibly boundary-hugging seed value: gather
	// every full-width line in the block (one seed line alone can put a
	// right-aligned value a hair on the wrong side of its own midpoint) and
	// put each boundary in the gap between the widest column c-1 gets and
	// the narrowest column c gets, falling back to the seed midpoint where
	// there isn't enough data to do better.
	colMin := make([]float64, numCols)
	colMax := make([]float64, numCols)
	for c := range colMin {
		colMin[c] = math.MaxFloat64
		colMax[c] = -math.MaxFloat64
	}
	for idx := start; idx <= end; idx++ {
		xs := spanXPositions(lines[idx].Spans)
		if len(xs) != numCols {
			continue
		}
		for c, x := range xs {
			colMin[c] = min(colMin[c], x)
			colMax[c] = max(colMax[c], x)
		}
	}

	bounds := make([]float64, numCols+1)
	bounds[0] = -math.MaxFloat64
	bounds[numCols] = math.MaxFloat64
	for c := 1; c < numCols; c++ {
		if colMax[c-1] > -math.MaxFloat64 && colMin[c] < math.MaxFloat64 && colMax[c-1] < colMin[c] {
			bounds[c] = (colMax[c-1] + colMin[c]) / 2
		} else {
			bounds[c] = (seeds[c-1] + seeds[c]) / 2
		}
	}
	colForX := func(x float64) int {
		for c := range numCols {
			if x >= bounds[c] && x < bounds[c+1] {
				return c
			}
		}
		return numCols - 1
	}

	// Group physical lines into logical rows: a small Y gap means the line
	// is a wrapped continuation of the current row, a larger gap starts a
	// new row.
	var rows [][]int
	var current []int
	lastY := lines[start].Y
	for idx := start; idx <= end; idx++ {
		line := lines[idx]
		if len(current) > 0 {
			fontSize := avgFontSize(line.Spans)
			if fontSize == 0 {
				fontSize = 12
			}
			if math.Abs(lastY-line.Y) > fontSize*2.0 {
				rows = append(rows, current)
				current = nil
			}
		}
		current = append(current, idx)
		lastY = line.Y
	}
	if len(current) > 0 {
		rows = append(rows, current)
	}
	if len(rows) < 2 {
		return nil
	}

	cells := make([][]TableCell, len(rows))
	for r, lineIdxs := range rows {
		rowCells := make([]TableCell, numCols)
		for c := range rowCells {
			rowCells[c] = TableCell{Row: r, Col: c, RowSpan: 1, ColSpan: 1}
		}
		for _, li := range lineIdxs {
			for _, span := range lines[li].Spans {
				c := colForX(span.X)
				if rowCells[c].Text != "" {
					rowCells[c].Text += " "
				}
				rowCells[c].Text += span.Text
				rowCells[c].Spans = append(rowCells[c].Spans, span)
			}
		}
		cells[r] = rowCells
	}

	return &DetectedTable{
		Rows:  len(rows),
		Cols:  numCols,
		Cells: cells,
		Rect:  [4]float64{seeds[0], lines[end].Y, seeds[numCols-1] + 100, lines[start].Y},
	}
}

// tableLineIndex maps a line's rounded Y position to the index of the
// DetectedTable it belongs to, so callers can render each table exactly once
// at the position of its first line and skip the rest of its rows instead of
// re-emitting the source lines as plain text.
func tableLineIndex(tables []DetectedTable) map[float64]int {
	if len(tables) == 0 {
		return nil
	}
	idx := make(map[float64]int)
	for ti, table := range tables {
		for _, row := range table.Cells {
			for _, cell := range row {
				for _, span := range cell.Spans {
					idx[math.Round(span.Y*10)/10] = ti
				}
			}
		}
	}
	return idx
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

// looksLikeList reports whether a detected two-column table is really a
// bulleted or numbered list: a marker column ("•", "1.", "2)", ...) followed
// by a text column, repeated down every row.
func looksLikeList(cells [][]TableCell) bool {
	if len(cells) == 0 || len(cells[0]) != 2 {
		return false
	}
	for _, row := range cells {
		if !isListMarker(strings.TrimSpace(row[0].Text)) {
			return false
		}
	}
	return true
}

// isListMarker reports whether s is a bullet glyph or a bare numeric marker
// such as "1", "1.", "1)", or "(1)".
func isListMarker(s string) bool {
	if s == "" {
		return false
	}
	if slices.Contains(bulletGlyphs, s) {
		return true
	}
	digits := strings.TrimSuffix(strings.TrimSuffix(s, ")"), ".")
	digits = strings.TrimPrefix(digits, "(")
	if digits == "" {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
