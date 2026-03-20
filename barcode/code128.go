package barcode

import (
	"errors"
	"fmt"
	"strings"
)

// Code128 represents an encoded Code 128 barcode.
type Code128 struct {
	Bars []bool // true = bar (dark), false = space (light)
	Text string
}

// Code 128 bar patterns: each symbol is 6 elements (3 bars + 3 spaces) = 11 modules wide
// Values: widths of bar, space, bar, space, bar, space
var code128Patterns = [107][6]int{
	{2, 1, 2, 2, 2, 2}, // 0
	{2, 2, 2, 1, 2, 2}, // 1
	{2, 2, 2, 2, 2, 1}, // 2
	{1, 2, 1, 2, 2, 3}, // 3
	{1, 2, 1, 3, 2, 2}, // 4
	{1, 3, 1, 2, 2, 2}, // 5
	{1, 2, 2, 2, 1, 3}, // 6
	{1, 2, 2, 3, 1, 2}, // 7
	{1, 3, 2, 2, 1, 2}, // 8
	{2, 2, 1, 2, 1, 3}, // 9
	{2, 2, 1, 3, 1, 2}, // 10
	{2, 3, 1, 2, 1, 2}, // 11
	{1, 1, 2, 2, 3, 2}, // 12
	{1, 2, 2, 1, 3, 2}, // 13
	{1, 2, 2, 2, 3, 1}, // 14
	{1, 1, 3, 2, 2, 2}, // 15
	{1, 2, 3, 1, 2, 2}, // 16
	{1, 2, 3, 2, 2, 1}, // 17
	{2, 2, 3, 2, 1, 1}, // 18
	{2, 2, 1, 1, 3, 2}, // 19
	{2, 2, 1, 2, 3, 1}, // 20
	{2, 1, 3, 2, 1, 2}, // 21
	{2, 2, 3, 1, 1, 2}, // 22
	{3, 1, 2, 1, 3, 1}, // 23
	{3, 1, 1, 2, 2, 2}, // 24
	{3, 2, 1, 1, 2, 2}, // 25
	{3, 2, 1, 2, 2, 1}, // 26
	{3, 1, 2, 2, 1, 2}, // 27
	{3, 2, 2, 1, 1, 2}, // 28
	{3, 2, 2, 2, 1, 1}, // 29
	{2, 1, 2, 1, 2, 3}, // 30
	{2, 1, 2, 3, 2, 1}, // 31
	{2, 3, 2, 1, 2, 1}, // 32
	{1, 1, 1, 3, 2, 3}, // 33
	{1, 3, 1, 1, 2, 3}, // 34
	{1, 3, 1, 3, 2, 1}, // 35
	{1, 1, 2, 3, 1, 3}, // 36
	{1, 3, 2, 1, 1, 3}, // 37
	{1, 3, 2, 3, 1, 1}, // 38
	{2, 1, 1, 3, 1, 3}, // 39
	{2, 3, 1, 1, 1, 3}, // 40
	{2, 3, 1, 3, 1, 1}, // 41
	{1, 1, 2, 1, 3, 3}, // 42
	{1, 1, 2, 3, 3, 1}, // 43
	{1, 3, 2, 1, 3, 1}, // 44
	{1, 1, 3, 1, 2, 3}, // 45
	{1, 1, 3, 3, 2, 1}, // 46
	{1, 3, 3, 1, 2, 1}, // 47
	{3, 1, 3, 1, 2, 1}, // 48
	{2, 1, 1, 3, 3, 1}, // 49
	{2, 3, 1, 1, 3, 1}, // 50
	{2, 1, 3, 1, 1, 3}, // 51
	{2, 1, 3, 3, 1, 1}, // 52
	{2, 1, 3, 1, 3, 1}, // 53
	{3, 1, 1, 1, 2, 3}, // 54
	{3, 1, 1, 3, 2, 1}, // 55
	{3, 3, 1, 1, 2, 1}, // 56
	{3, 1, 2, 1, 1, 3}, // 57
	{3, 1, 2, 3, 1, 1}, // 58
	{3, 3, 2, 1, 1, 1}, // 59
	{2, 1, 1, 2, 1, 4}, // 60
	{2, 1, 1, 4, 1, 2}, // 61
	{4, 1, 1, 2, 1, 2}, // 62
	{1, 4, 1, 1, 1, 3}, // 63 (was DEL in A, was 63 in B)
	{1, 1, 1, 2, 4, 2}, // 64
	{1, 2, 1, 1, 4, 2}, // 65
	{1, 2, 1, 4, 1, 2}, // 66 (FNC1 in C / 66 in B)
	{4, 2, 1, 1, 1, 2}, // 67
	{4, 2, 1, 2, 1, 1}, // 68
	{2, 1, 4, 1, 1, 2}, // 69
	{2, 4, 1, 1, 1, 2}, // 70 (was 70)
	{1, 1, 4, 1, 2, 2}, // 71
	{1, 2, 4, 1, 1, 2}, // 72
	{1, 2, 4, 2, 1, 1}, // 73
	{4, 1, 1, 1, 1, 3}, // 74
	{4, 1, 1, 1, 3, 1}, // 75
	{1, 1, 1, 4, 2, 2}, // 76
	{1, 2, 1, 4, 2, 1}, // 77
	{1, 4, 1, 2, 2, 1}, // 78 (was 78)
	{2, 4, 1, 2, 1, 1}, // 79
	{1, 1, 4, 2, 2, 1}, // 80
	{1, 2, 4, 1, 2, 1}, // 81
	{2, 1, 4, 2, 1, 1}, // 82
	{4, 1, 2, 2, 1, 1}, // 83
	{2, 4, 2, 1, 1, 1}, // 84
	{1, 1, 1, 1, 4, 3}, // 85
	{1, 1, 1, 3, 4, 1}, // 86
	{1, 3, 1, 1, 4, 1}, // 87
	{1, 1, 4, 1, 1, 3}, // 88
	{1, 1, 4, 3, 1, 1}, // 89
	{4, 1, 1, 1, 1, 3}, // 90 (was 90)
	{4, 1, 1, 3, 1, 1}, // 91
	{1, 1, 3, 1, 4, 1}, // 92
	{1, 1, 4, 1, 3, 1}, // 93
	{3, 1, 1, 1, 4, 1}, // 94
	{4, 1, 1, 1, 3, 1}, // 95
	{2, 1, 1, 4, 1, 2}, // 96
	{2, 1, 1, 2, 4, 1}, // 97 (was 97)
	{2, 1, 1, 2, 1, 4}, // 98 (SHIFT)
	{2, 1, 1, 1, 4, 2}, // 99 (CODE C)
	{4, 3, 1, 1, 1, 1}, // 100 (CODE B / FNC4)
	{2, 1, 1, 1, 1, 4}, // 101 (CODE A / FNC4)
	{4, 1, 1, 1, 1, 4}, // 102 (FNC1)
	{1, 1, 2, 1, 4, 2}, // 103 (START A)
	{1, 1, 2, 2, 4, 1}, // 104 (START B)
	{1, 1, 2, 4, 2, 1}, // 105 (START C)
	{1, 1, 1, 4, 4, 1}, // 106 (STOP - has extra bar making 13 modules)
}

const (
	code128StartA = 103
	code128StartB = 104
	code128StartC = 105
	code128CodeA  = 101
	code128CodeB  = 100
	code128CodeC  = 99
	code128Stop   = 106
)

// EncodeCode128 encodes data as a Code 128 barcode with auto code set selection.
func EncodeCode128(data string) (*Code128, error) {
	if len(data) == 0 {
		return nil, errors.New("code128: empty data")
	}

	// Simple auto-selection: use Code C for all-numeric even-length, else Code B
	symbols := code128SelectSymbols(data)
	if symbols == nil {
		return nil, errors.New("code128: cannot encode data")
	}

	// Compute checksum
	sum := symbols[0] // start code value
	for i := 1; i < len(symbols); i++ {
		sum += symbols[i] * i
	}
	check := sum % 103
	symbols = append(symbols, check, code128Stop)

	// Convert symbols to bars
	var bars []bool
	for si, sym := range symbols {
		pat := code128Patterns[sym]
		for i, w := range pat {
			isBar := i%2 == 0
			for j := 0; j < w; j++ {
				bars = append(bars, isBar)
			}
		}
		// Stop pattern has an extra 2-width bar
		if si == len(symbols)-1 {
			bars = append(bars, true, true)
		}
	}

	return &Code128{Bars: bars, Text: data}, nil
}

func code128SelectSymbols(data string) []int {
	// Check if all digits and even length -> use Code C
	allDigit := true
	for _, c := range data {
		if c < '0' || c > '9' {
			allDigit = false
			break
		}
	}

	if allDigit && len(data)%2 == 0 && len(data) >= 2 {
		symbols := []int{code128StartC}
		for i := 0; i < len(data); i += 2 {
			val := int(data[i]-'0')*10 + int(data[i+1]-'0')
			symbols = append(symbols, val)
		}
		return symbols
	}

	// Use Code B for general data
	symbols := []int{code128StartB}
	for _, c := range data {
		if c < 32 || c > 127 {
			return nil // unsupported
		}
		symbols = append(symbols, int(c)-32)
	}
	return symbols
}

// Widths returns the bar/space widths for rendering.
func (c *Code128) Widths() []int {
	if len(c.Bars) == 0 {
		return nil
	}
	var widths []int
	current := c.Bars[0]
	w := 1
	for i := 1; i < len(c.Bars); i++ {
		if c.Bars[i] == current {
			w++
		} else {
			widths = append(widths, w)
			current = c.Bars[i]
			w = 1
		}
	}
	widths = append(widths, w)
	return widths
}

// SVG returns an SVG representation of the Code 128 barcode.
func (c *Code128) SVG(barWidth, height float64) string {
	totalWidth := float64(len(c.Bars)) * barWidth
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %.1f %.1f" width="%.1f" height="%.1f">`, totalWidth, height, totalWidth, height))
	sb.WriteString(fmt.Sprintf(`<rect width="%.1f" height="%.1f" fill="white"/>`, totalWidth, height))
	x := 0.0
	for _, bar := range c.Bars {
		if bar {
			sb.WriteString(fmt.Sprintf(`<rect x="%.2f" y="0" width="%.2f" height="%.1f" fill="black"/>`, x, barWidth, height))
		}
		x += barWidth
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}
