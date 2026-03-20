package barcode

import (
	"errors"
	"fmt"
	"strings"
)

// QRCode represents an encoded QR code.
type QRCode struct {
	Version int
	ECLevel ECLevel
	Modules [][]bool // true = dark module
	Size    int
}

// EncodeQR encodes the given data string as a QR code.
func EncodeQR(data string, ecLevel ECLevel) (*QRCode, error) {
	// 1. Determine encoding mode
	mode := qrPickMode(data)

	// 2. Find minimum version
	version, err := qrPickVersion(data, mode, ecLevel)
	if err != nil {
		return nil, err
	}

	// 3. Encode data bits
	bits := qrEncodeData(data, mode, version, ecLevel)

	// 4. Split into blocks and compute EC
	info := qrECInfo[version-1][ecLevel]
	dataBytes := bitsToBytes(bits)

	blocks, ecBlocks := qrSplitAndEC(dataBytes, info)

	// 5. Interleave
	interleaved := qrInterleave(blocks, ecBlocks, info)

	// 6. Build matrix
	size := version*4 + 17
	modules := make([][]bool, size)
	reserved := make([][]bool, size)
	for i := range modules {
		modules[i] = make([]bool, size)
		reserved[i] = make([]bool, size)
	}

	// Place function patterns
	qrPlaceFinderPatterns(modules, reserved, size)
	qrPlaceAlignmentPatterns(modules, reserved, version, size)
	qrPlaceTimingPatterns(modules, reserved, size)
	qrPlaceDarkModule(modules, reserved, version)
	qrReserveFormatAreas(reserved, size)
	if version >= 7 {
		qrReserveVersionAreas(reserved, size)
	}

	// Place data bits
	qrPlaceDataBits(modules, reserved, interleaved, size)

	// 7. Apply mask patterns, evaluate penalty, pick best
	bestMask := -1
	bestPenalty := int(^uint(0) >> 1)
	var bestModules [][]bool

	for mask := 0; mask < 8; mask++ {
		candidate := qrCopyMatrix(modules, size)
		qrApplyMask(candidate, reserved, mask, size)
		qrPlaceFormatInfo(candidate, ecLevel, mask, size)
		if version >= 7 {
			qrPlaceVersionInfo(candidate, version, size)
		}
		penalty := qrPenalty(candidate, size)
		if penalty < bestPenalty {
			bestPenalty = penalty
			bestMask = mask
			bestModules = candidate
		}
	}
	_ = bestMask

	return &QRCode{
		Version: version,
		ECLevel: ecLevel,
		Modules: bestModules,
		Size:    size,
	}, nil
}

// Render returns the QR code modules grid.
func (qr *QRCode) Render() [][]bool {
	return qr.Modules
}

// SVG returns an SVG representation of the QR code.
func (qr *QRCode) SVG(moduleSize float64) string {
	quiet := 4
	totalSize := float64(qr.Size+quiet*2) * moduleSize
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.1f %.1f" width="%.1f" height="%.1f">`, totalSize, totalSize, totalSize, totalSize))
	sb.WriteString(fmt.Sprintf(`<rect width="%.1f" height="%.1f" fill="white"/>`, totalSize, totalSize))
	for y := 0; y < qr.Size; y++ {
		for x := 0; x < qr.Size; x++ {
			if qr.Modules[y][x] {
				px := float64(x+quiet) * moduleSize
				py := float64(y+quiet) * moduleSize
				sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="black"/>`, px, py, moduleSize, moduleSize))
			}
		}
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}

// --- mode detection ---

const (
	qrModeNumeric      = 1
	qrModeAlphanumeric = 2
	qrModeByte         = 4
)

func qrPickMode(data string) int {
	numeric := true
	alpha := true
	for _, c := range data {
		if c < '0' || c > '9' {
			numeric = false
		}
		if c >= 128 || qrAlphanumericMap[c] < 0 {
			alpha = false
		}
	}
	if numeric {
		return qrModeNumeric
	}
	if alpha {
		return qrModeAlphanumeric
	}
	return qrModeByte
}

func qrPickVersion(data string, mode int, ecLevel ECLevel) (int, error) {
	n := len(data)
	var modeIdx int
	switch mode {
	case qrModeNumeric:
		modeIdx = 0
	case qrModeAlphanumeric:
		modeIdx = 1
	default:
		modeIdx = 2
	}
	for v := 1; v <= 40; v++ {
		if qrVersionCapacity[v-1][ecLevel][modeIdx] >= n {
			return v, nil
		}
	}
	return 0, errors.New("qr: data too large for any version")
}

// --- data encoding ---

func qrEncodeData(data string, mode, version int, ecLevel ECLevel) []bool {
	totalDataCW := qrDataCapacity[version-1][ecLevel]
	totalBits := totalDataCW * 8

	var bits []bool
	appendBits := func(val, count int) {
		for i := count - 1; i >= 0; i-- {
			bits = append(bits, (val>>uint(i))&1 == 1)
		}
	}

	// Mode indicator (4 bits)
	appendBits(mode, 4)

	// Character count indicator
	ccBits := qrCharCountBits(version, mode)
	appendBits(len(data), ccBits)

	// Encode data
	switch mode {
	case qrModeNumeric:
		for i := 0; i < len(data); i += 3 {
			rem := len(data) - i
			if rem >= 3 {
				val := int(data[i]-'0')*100 + int(data[i+1]-'0')*10 + int(data[i+2]-'0')
				appendBits(val, 10)
			} else if rem == 2 {
				val := int(data[i]-'0')*10 + int(data[i+1]-'0')
				appendBits(val, 7)
			} else {
				appendBits(int(data[i]-'0'), 4)
			}
		}
	case qrModeAlphanumeric:
		for i := 0; i < len(data); i += 2 {
			if i+1 < len(data) {
				val := qrAlphanumericMap[data[i]]*45 + qrAlphanumericMap[data[i+1]]
				appendBits(val, 11)
			} else {
				appendBits(qrAlphanumericMap[data[i]], 6)
			}
		}
	case qrModeByte:
		for i := 0; i < len(data); i++ {
			appendBits(int(data[i]), 8)
		}
	}

	// Terminator
	termLen := 4
	if totalBits-len(bits) < termLen {
		termLen = totalBits - len(bits)
	}
	for i := 0; i < termLen; i++ {
		bits = append(bits, false)
	}

	// Pad to byte boundary
	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}

	// Pad bytes
	padBytes := []byte{0xEC, 0x11}
	pi := 0
	for len(bits) < totalBits {
		appendBits(int(padBytes[pi]), 8)
		pi = (pi + 1) % 2
	}

	return bits[:totalBits]
}

func qrCharCountBits(version, mode int) int {
	switch {
	case version <= 9:
		switch mode {
		case qrModeNumeric:
			return 10
		case qrModeAlphanumeric:
			return 9
		default:
			return 8
		}
	case version <= 26:
		switch mode {
		case qrModeNumeric:
			return 12
		case qrModeAlphanumeric:
			return 11
		default:
			return 16
		}
	default:
		switch mode {
		case qrModeNumeric:
			return 14
		case qrModeAlphanumeric:
			return 13
		default:
			return 16
		}
	}
}

func bitsToBytes(bits []bool) []byte {
	result := make([]byte, len(bits)/8)
	for i := range result {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i*8+j] {
				b |= 1 << uint(7-j)
			}
		}
		result[i] = b
	}
	return result
}

// --- EC and interleaving ---

func qrSplitAndEC(data []byte, info qrVersionInfo) ([][]byte, [][]byte) {
	blocks := make([][]byte, 0, info.Group1Blocks+info.Group2Blocks)
	ecBlocks := make([][]byte, 0, info.Group1Blocks+info.Group2Blocks)
	idx := 0
	for i := 0; i < info.Group1Blocks; i++ {
		block := make([]byte, info.Group1Data)
		copy(block, data[idx:idx+info.Group1Data])
		idx += info.Group1Data
		blocks = append(blocks, block)
		ecBlocks = append(ecBlocks, RSEncode(block, info.ECPerBlock))
	}
	for i := 0; i < info.Group2Blocks; i++ {
		block := make([]byte, info.Group2Data)
		copy(block, data[idx:idx+info.Group2Data])
		idx += info.Group2Data
		blocks = append(blocks, block)
		ecBlocks = append(ecBlocks, RSEncode(block, info.ECPerBlock))
	}
	return blocks, ecBlocks
}

func qrInterleave(blocks, ecBlocks [][]byte, info qrVersionInfo) []byte {
	var result []byte
	maxData := info.Group1Data
	if info.Group2Data > maxData {
		maxData = info.Group2Data
	}
	for i := 0; i < maxData; i++ {
		for _, b := range blocks {
			if i < len(b) {
				result = append(result, b[i])
			}
		}
	}
	for i := 0; i < info.ECPerBlock; i++ {
		for _, b := range ecBlocks {
			if i < len(b) {
				result = append(result, b[i])
			}
		}
	}
	return result
}

// --- matrix construction ---

func qrPlaceFinderPatterns(modules, reserved [][]bool, size int) {
	placeFinderAt := func(row, col int) {
		for dr := -1; dr <= 7; dr++ {
			for dc := -1; dc <= 7; dc++ {
				r, c := row+dr, col+dc
				if r < 0 || r >= size || c < 0 || c >= size {
					continue
				}
				reserved[r][c] = true
				// Finder pattern: 7x7 with nested squares
				if dr >= 0 && dr <= 6 && dc >= 0 && dc <= 6 {
					if dr == 0 || dr == 6 || dc == 0 || dc == 6 ||
						(dr >= 2 && dr <= 4 && dc >= 2 && dc <= 4) {
						modules[r][c] = true
					}
				}
			}
		}
	}
	placeFinderAt(0, 0)
	placeFinderAt(0, size-7)
	placeFinderAt(size-7, 0)
}

func qrPlaceAlignmentPatterns(modules, reserved [][]bool, version, size int) {
	if version < 2 {
		return
	}
	positions := qrAlignmentPatterns[version]
	for _, cy := range positions {
		for _, cx := range positions {
			// Skip if overlapping with finder patterns
			if (cy <= 8 && cx <= 8) || (cy <= 8 && cx >= size-8) || (cy >= size-8 && cx <= 8) {
				continue
			}
			for dr := -2; dr <= 2; dr++ {
				for dc := -2; dc <= 2; dc++ {
					r, c := cy+dr, cx+dc
					reserved[r][c] = true
					if dr == -2 || dr == 2 || dc == -2 || dc == 2 || (dr == 0 && dc == 0) {
						modules[r][c] = true
					} else {
						modules[r][c] = false
					}
				}
			}
		}
	}
}

func qrPlaceTimingPatterns(modules, reserved [][]bool, size int) {
	for i := 8; i < size-8; i++ {
		reserved[6][i] = true
		modules[6][i] = i%2 == 0
		reserved[i][6] = true
		modules[i][6] = i%2 == 0
	}
}

func qrPlaceDarkModule(modules, reserved [][]bool, version int) {
	r := 4*version + 9
	modules[r][8] = true
	reserved[r][8] = true
}

func qrReserveFormatAreas(reserved [][]bool, size int) {
	for i := 0; i < 9; i++ {
		if i < size {
			reserved[8][i] = true
			reserved[i][8] = true
		}
	}
	for i := 0; i < 8; i++ {
		reserved[8][size-1-i] = true
		reserved[size-1-i][8] = true
	}
}

func qrReserveVersionAreas(reserved [][]bool, size int) {
	for i := 0; i < 6; i++ {
		for j := 0; j < 3; j++ {
			reserved[i][size-11+j] = true
			reserved[size-11+j][i] = true
		}
	}
}

func qrPlaceDataBits(modules, reserved [][]bool, data []byte, size int) {
	bitIdx := 0
	totalBits := len(data) * 8

	getBit := func() bool {
		if bitIdx >= totalBits {
			return false
		}
		b := (data[bitIdx/8] >> uint(7-bitIdx%8)) & 1
		bitIdx++
		return b == 1
	}

	// Traverse columns right to left, in pairs
	col := size - 1
	for col >= 0 {
		if col == 6 { // skip timing column
			col--
		}
		for row := 0; row < size; row++ {
			// Upward or downward based on column pair
			upward := ((size - 1 - col) / 2) % 2 == 0
			r := row
			if upward {
				r = size - 1 - row
			}
			for c := col; c > col-2 && c >= 0; c-- {
				if !reserved[r][c] {
					modules[r][c] = getBit()
				}
			}
		}
		col -= 2
	}
}

func qrCopyMatrix(src [][]bool, size int) [][]bool {
	dst := make([][]bool, size)
	for i := range dst {
		dst[i] = make([]bool, size)
		copy(dst[i], src[i])
	}
	return dst
}

func qrApplyMask(modules, reserved [][]bool, mask, size int) {
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if reserved[r][c] {
				continue
			}
			var invert bool
			switch mask {
			case 0:
				invert = (r+c)%2 == 0
			case 1:
				invert = r%2 == 0
			case 2:
				invert = c%3 == 0
			case 3:
				invert = (r+c)%3 == 0
			case 4:
				invert = (r/2+c/3)%2 == 0
			case 5:
				invert = (r*c)%2+(r*c)%3 == 0
			case 6:
				invert = ((r*c)%2+(r*c)%3)%2 == 0
			case 7:
				invert = ((r+c)%2+(r*c)%3)%2 == 0
			}
			if invert {
				modules[r][c] = !modules[r][c]
			}
		}
	}
}

func qrPlaceFormatInfo(modules [][]bool, ecLevel ECLevel, mask, size int) {
	info := qrFormatInfo[ecLevel][mask]
	// Around top-left finder
	for i := 0; i < 8; i++ {
		bit := (info >> uint(14-i)) & 1 == 1
		// Horizontal
		c := i
		if i >= 6 {
			c = i + 1
		}
		modules[8][c] = bit
		// Vertical
		r := 0
		if i == 0 {
			r = size - 1
		} else if i <= 7 {
			r = size - i
		}
		modules[r][8] = bit
	}
	for i := 8; i < 15; i++ {
		bit := (info >> uint(14-i)) & 1 == 1
		// Horizontal: right side
		modules[8][size-15+i] = bit
		// Vertical: top
		r := 14 - i
		if r <= 5 {
			// row r
		} else {
			r = r + 1 // skip timing
		}
		modules[r][8] = bit
	}
}

func qrPlaceVersionInfo(modules [][]bool, version, size int) {
	if version < 7 {
		return
	}
	info := qrVersionInfoBits[version]
	for i := 0; i < 18; i++ {
		bit := (info >> uint(i)) & 1 == 1
		row := i / 3
		col := size - 11 + i%3
		modules[row][col] = bit
		modules[col][row] = bit
	}
}

// --- penalty ---

func qrPenalty(modules [][]bool, size int) int {
	p := 0
	// Rule 1: runs of same color in row/col
	for r := 0; r < size; r++ {
		count := 1
		for c := 1; c < size; c++ {
			if modules[r][c] == modules[r][c-1] {
				count++
			} else {
				if count >= 5 {
					p += count - 2
				}
				count = 1
			}
		}
		if count >= 5 {
			p += count - 2
		}
	}
	for c := 0; c < size; c++ {
		count := 1
		for r := 1; r < size; r++ {
			if modules[r][c] == modules[r-1][c] {
				count++
			} else {
				if count >= 5 {
					p += count - 2
				}
				count = 1
			}
		}
		if count >= 5 {
			p += count - 2
		}
	}
	// Rule 2: 2x2 blocks of same color
	for r := 0; r < size-1; r++ {
		for c := 0; c < size-1; c++ {
			v := modules[r][c]
			if v == modules[r][c+1] && v == modules[r+1][c] && v == modules[r+1][c+1] {
				p += 3
			}
		}
	}
	// Rule 3: finder-like patterns
	patterns := [2][11]bool{
		{true, false, true, true, true, false, true, false, false, false, false},
		{false, false, false, false, true, false, true, true, true, false, true},
	}
	for r := 0; r < size; r++ {
		for c := 0; c <= size-11; c++ {
			for _, pat := range patterns {
				match := true
				for k := 0; k < 11; k++ {
					if modules[r][c+k] != pat[k] {
						match = false
						break
					}
				}
				if match {
					p += 40
				}
			}
		}
	}
	for c := 0; c < size; c++ {
		for r := 0; r <= size-11; r++ {
			for _, pat := range patterns {
				match := true
				for k := 0; k < 11; k++ {
					if modules[r+k][c] != pat[k] {
						match = false
						break
					}
				}
				if match {
					p += 40
				}
			}
		}
	}
	// Rule 4: proportion of dark modules
	dark := 0
	total := size * size
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if modules[r][c] {
				dark++
			}
		}
	}
	pct := dark * 100 / total
	prev5 := (pct / 5) * 5
	next5 := prev5 + 5
	d1 := prev5 - 50
	if d1 < 0 {
		d1 = -d1
	}
	d2 := next5 - 50
	if d2 < 0 {
		d2 = -d2
	}
	d := d1
	if d2 < d {
		d = d2
	}
	p += (d / 5) * 10

	return p
}
