package svg

import (
	"strconv"
	"strings"
)

// Gradient represents an SVG gradient definition.
type Gradient struct {
	Type string // "linear" or "radial"
	ID   string

	Stops []GradientStop

	// Linear gradient attributes
	X1, Y1, X2, Y2 float64

	// Radial gradient attributes
	CX, CY, R, FX, FY float64

	Transform Matrix
	Units     string // "objectBoundingBox" or "userSpaceOnUse"
}

// GradientStop represents a stop in a gradient.
type GradientStop struct {
	Offset  float64
	Color   [3]float64
	Opacity float64
}

// parseGradient parses a linearGradient or radialGradient SVG node.
func parseGradient(node *SVGNode) *Gradient {
	if node == nil {
		return nil
	}
	g := &Gradient{
		ID:        node.Attrs["id"],
		Transform: Identity(),
		Units:     "objectBoundingBox",
	}

	if u := node.Attrs["gradientUnits"]; u != "" {
		g.Units = u
	}
	if t := node.Attrs["gradientTransform"]; t != "" {
		g.Transform = parseTransform(t)
	}

	switch node.Tag {
	case "linearGradient":
		g.Type = "linear"
		g.X1 = parseGradientCoord(node.Attrs["x1"], 0)
		g.Y1 = parseGradientCoord(node.Attrs["y1"], 0)
		g.X2 = parseGradientCoord(node.Attrs["x2"], 1)
		g.Y2 = parseGradientCoord(node.Attrs["y2"], 0)
	case "radialGradient":
		g.Type = "radial"
		g.CX = parseGradientCoord(node.Attrs["cx"], 0.5)
		g.CY = parseGradientCoord(node.Attrs["cy"], 0.5)
		g.R = parseGradientCoord(node.Attrs["r"], 0.5)
		g.FX = parseGradientCoord(node.Attrs["fx"], g.CX)
		g.FY = parseGradientCoord(node.Attrs["fy"], g.CY)
	default:
		return nil
	}

	// Parse stops
	for _, child := range node.Children {
		if child.Tag == "stop" {
			stop := parseGradientStop(child)
			g.Stops = append(g.Stops, stop)
		}
	}

	return g
}

func parseGradientCoord(s string, def float64) float64 {
	if s == "" {
		return def
	}
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parseGradientStop(node *SVGNode) GradientStop {
	stop := GradientStop{
		Opacity: 1,
		Color:   [3]float64{0, 0, 0},
	}

	// Parse offset
	offsetStr := node.Attr("offset")
	if offsetStr != "" {
		offsetStr = strings.TrimSpace(offsetStr)
		if strings.HasSuffix(offsetStr, "%") {
			v, _ := strconv.ParseFloat(strings.TrimSuffix(offsetStr, "%"), 64)
			stop.Offset = v / 100
		} else {
			stop.Offset, _ = strconv.ParseFloat(offsetStr, 64)
		}
	}

	// Parse color
	colorStr := node.Attr("stop-color")
	if colorStr != "" {
		if c, _, ok := parseColor(colorStr); ok {
			stop.Color = c
		}
	}

	// Parse opacity
	opStr := node.Attr("stop-opacity")
	if opStr != "" {
		if v, err := strconv.ParseFloat(opStr, 64); err == nil {
			stop.Opacity = v
		}
	}

	return stop
}

// SampleColor returns the interpolated color at position t (0-1) along the gradient.
func (g *Gradient) SampleColor(t float64) ([3]float64, float64) {
	if len(g.Stops) == 0 {
		return [3]float64{0, 0, 0}, 1
	}
	if t <= g.Stops[0].Offset || len(g.Stops) == 1 {
		return g.Stops[0].Color, g.Stops[0].Opacity
	}
	last := g.Stops[len(g.Stops)-1]
	if t >= last.Offset {
		return last.Color, last.Opacity
	}

	// Find the two stops to interpolate between
	for i := 1; i < len(g.Stops); i++ {
		if t <= g.Stops[i].Offset {
			s0 := g.Stops[i-1]
			s1 := g.Stops[i]
			d := s1.Offset - s0.Offset
			if d == 0 {
				return s1.Color, s1.Opacity
			}
			f := (t - s0.Offset) / d
			var c [3]float64
			for j := 0; j < 3; j++ {
				c[j] = s0.Color[j] + f*(s1.Color[j]-s0.Color[j])
			}
			o := s0.Opacity + f*(s1.Opacity-s0.Opacity)
			return c, o
		}
	}
	return last.Color, last.Opacity
}
