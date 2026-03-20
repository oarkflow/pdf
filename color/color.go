package color

import (
	"fmt"
	"math"
	"strings"
)

// RGB color with components 0-1
type RGB struct{ R, G, B float64 }

// CMYK color with components 0-1
type CMYK struct{ C, M, Y, K float64 }

// Gray color with component 0-1
type Gray struct{ G float64 }

// HSL color with H 0-360, S and L 0-1
type HSL struct{ H, S, L float64 }

// Color interface
type Color interface {
	ToRGB() RGB
	ToCMYK() CMYK
	ToGray() Gray
}

// ToRGB returns itself.
func (c RGB) ToRGB() RGB { return c }

// ToCMYK converts RGB to CMYK.
func (c RGB) ToCMYK() CMYK {
	k := 1 - math.Max(c.R, math.Max(c.G, c.B))
	if k >= 1 {
		return CMYK{0, 0, 0, 1}
	}
	return CMYK{
		C: (1 - c.R - k) / (1 - k),
		M: (1 - c.G - k) / (1 - k),
		Y: (1 - c.B - k) / (1 - k),
		K: k,
	}
}

// ToGray converts RGB to Gray using luminance weights.
func (c RGB) ToGray() Gray {
	return Gray{0.299*c.R + 0.587*c.G + 0.114*c.B}
}

// ToRGB converts CMYK to RGB.
func (c CMYK) ToRGB() RGB {
	return RGB{
		R: (1 - c.C) * (1 - c.K),
		G: (1 - c.M) * (1 - c.K),
		B: (1 - c.Y) * (1 - c.K),
	}
}

// ToCMYK returns itself.
func (c CMYK) ToCMYK() CMYK { return c }

// ToGray converts CMYK to Gray via RGB.
func (c CMYK) ToGray() Gray { return c.ToRGB().ToGray() }

// ToRGB converts Gray to RGB.
func (c Gray) ToRGB() RGB { return RGB{c.G, c.G, c.G} }

// ToCMYK converts Gray to CMYK.
func (c Gray) ToCMYK() CMYK { return c.ToRGB().ToCMYK() }

// ToGray returns itself.
func (c Gray) ToGray() Gray { return c }

// ToRGB converts HSL to RGB using the standard algorithm.
func (c HSL) ToRGB() RGB {
	if c.S == 0 {
		return RGB{c.L, c.L, c.L}
	}
	var q float64
	if c.L < 0.5 {
		q = c.L * (1 + c.S)
	} else {
		q = c.L + c.S - c.L*c.S
	}
	p := 2*c.L - q
	h := c.H / 360
	return RGB{
		R: hueToRGB(p, q, h+1.0/3),
		G: hueToRGB(p, q, h),
		B: hueToRGB(p, q, h-1.0/3),
	}
}

// ToCMYK converts HSL to CMYK via RGB.
func (c HSL) ToCMYK() CMYK { return c.ToRGB().ToCMYK() }

// ToGray converts HSL to Gray via RGB.
func (c HSL) ToGray() Gray { return c.ToRGB().ToGray() }

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6:
		return p + (q-p)*6*t
	case t < 1.0/2:
		return q
	case t < 2.0/3:
		return p + (q-p)*(2.0/3-t)*6
	default:
		return p
	}
}

// FromHex parses "#RRGGBB", "#RGB", "RRGGBB", or "RGB" into an RGB color.
func FromHex(hex string) (RGB, error) {
	hex = strings.TrimPrefix(hex, "#")
	switch len(hex) {
	case 3:
		r, g, b, err := parseHex3(hex)
		if err != nil {
			return RGB{}, err
		}
		return RGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}, nil
	case 6:
		r, g, b, err := parseHex6(hex)
		if err != nil {
			return RGB{}, err
		}
		return RGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}, nil
	default:
		return RGB{}, fmt.Errorf("color: invalid hex string %q", hex)
	}
}

func parseHex3(s string) (r, g, b uint8, err error) {
	var n int
	_, err = fmt.Sscanf(s, "%1x%1x%1x%n", &r, &g, &b, &n)
	if err != nil || n != 3 {
		return 0, 0, 0, fmt.Errorf("color: invalid hex string %q", s)
	}
	r = r*16 + r
	g = g*16 + g
	b = b*16 + b
	return
}

func parseHex6(s string) (r, g, b uint8, err error) {
	var n int
	_, err = fmt.Sscanf(s, "%2x%2x%2x%n", &r, &g, &b, &n)
	if err != nil || n != 6 {
		return 0, 0, 0, fmt.Errorf("color: invalid hex string %q", s)
	}
	return
}

// FromRGBi creates an RGB color from 0-255 integer components.
func FromRGBi(r, g, b uint8) RGB {
	return RGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}
}
