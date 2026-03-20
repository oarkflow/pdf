package svg

import (
	"fmt"
	"strconv"
	"strings"
)

// renderText renders an SVG <text> element to PDF content stream operators.
func renderText(node *SVGNode, buf *[]byte) {
	x := parseFloatAttr(node, "x", 0)
	y := parseFloatAttr(node, "y", 0)
	dx := parseFloatAttr(node, "dx", 0)
	dy := parseFloatAttr(node, "dy", 0)
	x += dx
	y += dy

	fontSize := parseFontSize(node)
	fontFamily := parseFontFamily(node)
	_ = fontFamily // font mapping is external

	// Set font (using a standard name; actual font resource must be set up externally)
	*buf = append(*buf, fmt.Sprintf("BT\n/F1 %.4g Tf\n", fontSize)...)

	// Apply fill color for text
	fillStr := node.Attr("fill")
	if fillStr == "" {
		fillStr = "black"
	}
	if fillStr != "none" {
		if c, _, ok := parseColor(fillStr); ok {
			*buf = append(*buf, fmt.Sprintf("%.4g %.4g %.4g rg\n", c[0], c[1], c[2])...)
		}
	}

	// If the node has direct text content
	if node.Text != "" {
		*buf = append(*buf, fmt.Sprintf("%.4g %.4g Td\n", x, y)...)
		*buf = append(*buf, fmt.Sprintf("(%s) Tj\n", escapePDFString(node.Text))...)
	}

	// Render tspan children
	cx, cy := x, y
	for _, child := range node.Children {
		if child.Tag == "tspan" {
			renderTspan(child, buf, &cx, &cy)
		}
	}

	*buf = append(*buf, "ET\n"...)
}

// renderTspan renders an SVG <tspan> element within a text block.
func renderTspan(node *SVGNode, buf *[]byte, x, y *float64) {
	newX := *x
	newY := *y

	if v, ok := node.Attrs["x"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			newX = f
		}
	}
	if v, ok := node.Attrs["y"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			newY = f
		}
	}
	if v, ok := node.Attrs["dx"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			newX += f
		}
	}
	if v, ok := node.Attrs["dy"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			newY += f
		}
	}

	// Check for font size change
	fs := parseFontSize(node)
	if fs > 0 {
		*buf = append(*buf, fmt.Sprintf("/F1 %.4g Tf\n", fs)...)
	}

	// Fill color
	fillStr := node.Attr("fill")
	if fillStr != "" && fillStr != "none" {
		if c, _, ok := parseColor(fillStr); ok {
			*buf = append(*buf, fmt.Sprintf("%.4g %.4g %.4g rg\n", c[0], c[1], c[2])...)
		}
	}

	// Position delta
	tdx := newX - *x
	tdy := newY - *y
	if tdx != 0 || tdy != 0 {
		*buf = append(*buf, fmt.Sprintf("%.4g %.4g Td\n", tdx, tdy)...)
	}

	if node.Text != "" {
		*buf = append(*buf, fmt.Sprintf("(%s) Tj\n", escapePDFString(node.Text))...)
	}

	*x = newX
	*y = newY

	// Recurse into child tspans
	for _, child := range node.Children {
		if child.Tag == "tspan" {
			renderTspan(child, buf, x, y)
		}
	}
}

func parseFontSize(node *SVGNode) float64 {
	s := node.Attr("font-size")
	if s == "" {
		return 12 // default
	}
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSuffix(s, "pt")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 12
	}
	return v
}

func parseFontFamily(node *SVGNode) string {
	s := node.Attr("font-family")
	if s == "" {
		return "Helvetica"
	}
	// Remove quotes
	s = strings.Trim(s, "'\"")
	return s
}

func escapePDFString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
