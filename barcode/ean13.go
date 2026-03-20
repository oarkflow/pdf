package barcode

import (
	"errors"
	"fmt"
	"strings"
)

// EAN13 represents an encoded EAN-13 barcode.
type EAN13 struct {
	Digits string // 13 digits including check digit
	Bars   []bool // true = bar
}

// L, G, R encoding patterns for digits 0-9.
// Each digit is encoded as 7 modules.
var ean13L = [10][7]bool{
	{false, false, false, true, true, false, true},  // 0
	{false, false, true, true, false, false, true},   // 1
	{false, false, true, false, false, true, true},   // 2
	{false, true, true, true, true, false, true},     // 3
	{false, true, false, false, false, true, true},   // 4
	{false, true, true, false, false, false, true},   // 5
	{false, true, false, true, true, true, true},     // 6
	{false, true, true, true, false, true, true},     // 7
	{false, true, true, false, true, true, true},     // 8
	{false, false, false, true, false, true, true},   // 9
}

var ean13G = [10][7]bool{
	{false, true, false, false, true, true, true},   // 0
	{false, true, true, false, false, true, true},   // 1
	{false, false, true, true, false, true, true},   // 2
	{false, true, false, false, false, false, true},  // 3
	{false, false, true, true, true, false, true},   // 4
	{false, true, true, true, false, false, true},   // 5
	{false, false, false, false, true, false, true},  // 6
	{false, false, true, false, false, false, true},  // 7
	{false, false, false, true, false, false, true},  // 8
	{false, false, true, false, true, true, true},   // 9
}

var ean13R = [10][7]bool{
	{true, true, true, false, false, true, false},  // 0
	{true, true, false, false, true, true, false},  // 1
	{true, true, false, true, true, false, false},  // 2
	{true, false, false, false, false, true, false}, // 3
	{true, false, true, true, true, false, false},  // 4
	{true, false, false, true, true, true, false},  // 5
	{true, false, true, false, false, false, false}, // 6
	{true, false, false, false, true, false, false}, // 7
	{true, false, false, true, false, false, false}, // 8
	{true, true, true, false, true, false, false},  // 9
}

// Parity patterns for first digit (which L/G pattern to use for digits 2-7)
// L=0, G=1
var ean13Parity = [10][6]int{
	{0, 0, 0, 0, 0, 0}, // 0
	{0, 0, 1, 0, 1, 1}, // 1
	{0, 0, 1, 1, 0, 1}, // 2
	{0, 0, 1, 1, 1, 0}, // 3
	{0, 1, 0, 0, 1, 1}, // 4
	{0, 1, 1, 0, 0, 1}, // 5
	{0, 1, 1, 1, 0, 0}, // 6
	{0, 1, 0, 1, 0, 1}, // 7
	{0, 1, 0, 1, 1, 0}, // 8
	{0, 1, 1, 0, 1, 0}, // 9
}

// EncodeEAN13 encodes an EAN-13 barcode. Accepts 12 digits (computes check digit)
// or 13 digits (validates check digit).
func EncodeEAN13(digits string) (*EAN13, error) {
	for _, c := range digits {
		if c < '0' || c > '9' {
			return nil, errors.New("ean13: non-digit character")
		}
	}

	switch len(digits) {
	case 12:
		digits = digits + string(rune('0'+ean13CheckDigit(digits)))
	case 13:
		expected := ean13CheckDigit(digits[:12])
		if int(digits[12]-'0') != expected {
			return nil, fmt.Errorf("ean13: invalid check digit: expected %d", expected)
		}
	default:
		return nil, errors.New("ean13: expected 12 or 13 digits")
	}

	bars := ean13Encode(digits)
	return &EAN13{Digits: digits, Bars: bars}, nil
}

func ean13CheckDigit(digits string) int {
	sum := 0
	for i := 0; i < 12; i++ {
		d := int(digits[i] - '0')
		if i%2 == 0 {
			sum += d
		} else {
			sum += d * 3
		}
	}
	return (10 - sum%10) % 10
}

func ean13Encode(digits string) []bool {
	var bars []bool

	// Start guard: 101
	bars = append(bars, true, false, true)

	firstDigit := int(digits[0] - '0')
	parity := ean13Parity[firstDigit]

	// Left group (digits[1..6])
	for i := 0; i < 6; i++ {
		d := int(digits[i+1] - '0')
		var pat [7]bool
		if parity[i] == 0 {
			pat = ean13L[d]
		} else {
			pat = ean13G[d]
		}
		bars = append(bars, pat[:]...)
	}

	// Center guard: 01010
	bars = append(bars, false, true, false, true, false)

	// Right group (digits[7..12])
	for i := 0; i < 6; i++ {
		d := int(digits[i+7] - '0')
		bars = append(bars, ean13R[d][:]...)
	}

	// End guard: 101
	bars = append(bars, true, false, true)

	return bars
}

// SVG returns an SVG representation of the EAN-13 barcode.
func (e *EAN13) SVG(barWidth, height float64) string {
	totalWidth := float64(len(e.Bars)) * barWidth
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.1f %.1f" width="%.1f" height="%.1f">`, totalWidth, height, totalWidth, height))
	sb.WriteString(fmt.Sprintf(`<rect width="%.1f" height="%.1f" fill="white"/>`, totalWidth, height))
	x := 0.0
	for _, bar := range e.Bars {
		if bar {
			sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="0" width="%.2f" height="%.1f" fill="black"/>`, x, barWidth, height))
		}
		x += barWidth
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}
