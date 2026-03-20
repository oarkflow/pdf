package svg

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Renderer walks an SVG tree and emits PDF content stream operators.
type Renderer struct {
	width, height float64
	viewBox       [4]float64
	buf           []byte
	defs          map[string]*SVGNode
	gradients     map[string]*Gradient
}

// NewRenderer creates a new SVG renderer with the given output dimensions.
func NewRenderer(width, height float64) *Renderer {
	return &Renderer{
		width:     width,
		height:    height,
		defs:      make(map[string]*SVGNode),
		gradients: make(map[string]*Gradient),
	}
}

// Render renders the SVG tree and returns PDF content stream operators.
func (r *Renderer) Render(root *SVGNode) []byte {
	r.buf = r.buf[:0]

	// Collect defs
	r.collectDefs(root)

	// Parse SVG root attributes
	if root.Tag == "svg" {
		r.parseSVGRoot(root)
	}

	// Apply viewBox transform
	r.applyViewBoxTransform()

	// Render children
	for _, child := range root.Children {
		r.renderNode(child)
	}

	return r.buf
}

func (r *Renderer) collectDefs(node *SVGNode) {
	if node.Tag == "defs" {
		for _, child := range node.Children {
			if id := child.Attrs["id"]; id != "" {
				r.defs[id] = child
			}
			switch child.Tag {
			case "linearGradient", "radialGradient":
				if g := parseGradient(child); g != nil {
					r.gradients[g.ID] = g
				}
			}
			r.collectDefs(child)
		}
		return
	}
	// Also collect defs that appear outside <defs>
	if id := node.Attrs["id"]; id != "" {
		r.defs[id] = node
	}
	switch node.Tag {
	case "linearGradient", "radialGradient":
		if g := parseGradient(node); g != nil {
			r.gradients[g.ID] = g
		}
	}
	for _, child := range node.Children {
		r.collectDefs(child)
	}
}

func (r *Renderer) parseSVGRoot(node *SVGNode) {
	if w := node.Attrs["width"]; w != "" {
		r.width = parseDimension(w, r.width)
	}
	if h := node.Attrs["height"]; h != "" {
		r.height = parseDimension(h, r.height)
	}
	if vb := node.Attrs["viewBox"]; vb != "" {
		parts := strings.Fields(strings.ReplaceAll(vb, ",", " "))
		if len(parts) == 4 {
			for i := 0; i < 4; i++ {
				r.viewBox[i], _ = strconv.ParseFloat(parts[i], 64)
			}
		}
	} else {
		r.viewBox = [4]float64{0, 0, r.width, r.height}
	}
}

func (r *Renderer) applyViewBoxTransform() {
	vbW := r.viewBox[2]
	vbH := r.viewBox[3]
	if vbW == 0 || vbH == 0 {
		return
	}
	sx := r.width / vbW
	sy := r.height / vbH
	tx := -r.viewBox[0] * sx
	ty := -r.viewBox[1] * sy
	m := Matrix{sx, 0, 0, sy, tx, ty}
	r.buf = append(r.buf, fmt.Sprintf("%s\n", m.ToPDFOperator())...)
}

func (r *Renderer) renderNode(node *SVGNode) {
	switch node.Tag {
	case "defs":
		return // already collected
	case "g":
		r.renderGroup(node)
	case "rect":
		r.renderRect(node)
	case "circle":
		r.renderCircle(node)
	case "ellipse":
		r.renderEllipse(node)
	case "line":
		r.renderLine(node)
	case "polyline":
		r.renderPolyline(node, false)
	case "polygon":
		r.renderPolyline(node, true)
	case "path":
		r.renderPathElement(node)
	case "text":
		r.renderTextElement(node)
	case "image":
		r.renderImage(node)
	case "use":
		r.renderUse(node)
	case "clipPath":
		return // handled when referenced
	case "linearGradient", "radialGradient":
		return // handled via defs
	case "svg":
		// Nested SVG
		for _, child := range node.Children {
			r.renderNode(child)
		}
	default:
		// Unknown element, try rendering children
		for _, child := range node.Children {
			r.renderNode(child)
		}
	}
}

func (r *Renderer) renderGroup(node *SVGNode) {
	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)
	r.applyClipPath(node)
	for _, child := range node.Children {
		r.renderNode(child)
	}
	r.buf = append(r.buf, "Q\n"...)
}

func (r *Renderer) applyTransform(node *SVGNode) {
	if t := node.Attrs["transform"]; t != "" {
		m := parseTransform(t)
		r.buf = append(r.buf, fmt.Sprintf("%s\n", m.ToPDFOperator())...)
	}
}

func (r *Renderer) applyClipPath(node *SVGNode) {
	clip := node.Attr("clip-path")
	if clip == "" {
		return
	}
	// Extract url(#id)
	id := extractURLRef(clip)
	if id == "" {
		return
	}
	clipNode, ok := r.defs[id]
	if !ok || clipNode.Tag != "clipPath" {
		return
	}
	for _, child := range clipNode.Children {
		r.emitShapePath(child)
	}
	r.buf = append(r.buf, "W n\n"...)
}

func (r *Renderer) emitShapePath(node *SVGNode) {
	switch node.Tag {
	case "rect":
		x := parseFloatAttr(node, "x", 0)
		y := parseFloatAttr(node, "y", 0)
		w := parseFloatAttr(node, "width", 0)
		h := parseFloatAttr(node, "height", 0)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g re\n", x, y, w, h)...)
	case "circle":
		cx := parseFloatAttr(node, "cx", 0)
		cy := parseFloatAttr(node, "cy", 0)
		rad := parseFloatAttr(node, "r", 0)
		r.emitEllipsePath(cx, cy, rad, rad)
	case "ellipse":
		cx := parseFloatAttr(node, "cx", 0)
		cy := parseFloatAttr(node, "cy", 0)
		rx := parseFloatAttr(node, "rx", 0)
		ry := parseFloatAttr(node, "ry", 0)
		r.emitEllipsePath(cx, cy, rx, ry)
	case "path":
		d := node.Attrs["d"]
		if d != "" {
			cmds := parsePath(d)
			renderPath(cmds, &r.buf)
		}
	}
}

func (r *Renderer) applyStyle(node *SVGNode) (paint string) {
	fillStr := node.Attr("fill")
	strokeStr := node.Attr("stroke")
	strokeWidth := node.Attr("stroke-width")
	opacity := parseFloatDefault(node.Attr("opacity"), 1)
	fillOpacity := parseFloatDefault(node.Attr("fill-opacity"), 1) * opacity
	strokeOpacity := parseFloatDefault(node.Attr("stroke-opacity"), 1) * opacity
	fillRule := node.Attr("fill-rule")

	hasFill := true
	hasStroke := false

	// Fill
	if fillStr == "none" {
		hasFill = false
	} else if fillStr == "" {
		// Default fill is black
		fillStr = "black"
	}

	if hasFill {
		ref := extractURLRef(fillStr)
		if ref != "" {
			// Gradient reference - sample mid color as approximation
			if g, ok := r.gradients[ref]; ok {
				c, _ := g.SampleColor(0.5)
				r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g rg\n", c[0], c[1], c[2])...)
			}
		} else if c, _, ok := parseColor(fillStr); ok {
			if fillOpacity < 1 {
				r.buf = append(r.buf, fmt.Sprintf("/GS0 gs\n")...) // would need ExtGState
			}
			r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g rg\n", c[0], c[1], c[2])...)
		}
	}

	// Stroke
	if strokeStr != "" && strokeStr != "none" {
		hasStroke = true
		ref := extractURLRef(strokeStr)
		if ref != "" {
			if g, ok := r.gradients[ref]; ok {
				c, _ := g.SampleColor(0.5)
				r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g RG\n", c[0], c[1], c[2])...)
			}
		} else if c, _, ok := parseColor(strokeStr); ok {
			_ = strokeOpacity
			r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g RG\n", c[0], c[1], c[2])...)
		}
	}

	// Stroke width
	if strokeWidth != "" {
		if w, err := strconv.ParseFloat(strings.TrimSuffix(strokeWidth, "px"), 64); err == nil {
			r.buf = append(r.buf, fmt.Sprintf("%.4g w\n", w)...)
		}
	}

	// Line cap
	if lc := node.Attr("stroke-linecap"); lc != "" {
		switch lc {
		case "butt":
			r.buf = append(r.buf, "0 J\n"...)
		case "round":
			r.buf = append(r.buf, "1 J\n"...)
		case "square":
			r.buf = append(r.buf, "2 J\n"...)
		}
	}

	// Line join
	if lj := node.Attr("stroke-linejoin"); lj != "" {
		switch lj {
		case "miter":
			r.buf = append(r.buf, "0 j\n"...)
		case "round":
			r.buf = append(r.buf, "1 j\n"...)
		case "bevel":
			r.buf = append(r.buf, "2 j\n"...)
		}
	}

	// Dash array
	if da := node.Attr("stroke-dasharray"); da != "" && da != "none" {
		parts := strings.FieldsFunc(da, func(c rune) bool { return c == ',' || c == ' ' })
		r.buf = append(r.buf, '[')
		for i, p := range parts {
			if i > 0 {
				r.buf = append(r.buf, ' ')
			}
			r.buf = append(r.buf, strings.TrimSpace(p)...)
		}
		do := node.Attr("stroke-dashoffset")
		if do == "" {
			do = "0"
		}
		r.buf = append(r.buf, fmt.Sprintf("] %s d\n", do)...)
	}

	// Determine paint operator
	useEvenOdd := fillRule == "evenodd"
	switch {
	case hasFill && hasStroke:
		if useEvenOdd {
			return "B*"
		}
		return "B"
	case hasFill:
		if useEvenOdd {
			return "f*"
		}
		return "f"
	case hasStroke:
		return "S"
	default:
		return "n"
	}
}

func (r *Renderer) renderRect(node *SVGNode) {
	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)
	paint := r.applyStyle(node)

	x := parseFloatAttr(node, "x", 0)
	y := parseFloatAttr(node, "y", 0)
	w := parseFloatAttr(node, "width", 0)
	h := parseFloatAttr(node, "height", 0)
	rx := parseFloatAttr(node, "rx", 0)
	ry := parseFloatAttr(node, "ry", 0)

	if rx == 0 && ry != 0 {
		rx = ry
	}
	if ry == 0 && rx != 0 {
		ry = rx
	}

	if rx == 0 && ry == 0 {
		// Simple rectangle
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g re\n", x, y, w, h)...)
	} else {
		// Rounded rectangle using curves
		if rx > w/2 {
			rx = w / 2
		}
		if ry > h/2 {
			ry = h / 2
		}
		// kappa for circle approximation
		k := 0.5522847498
		kx := rx * k
		ky := ry * k

		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g m\n", x+rx, y)...)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g l\n", x+w-rx, y)...)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", x+w-rx+kx, y, x+w, y+ry-ky, x+w, y+ry)...)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g l\n", x+w, y+h-ry)...)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", x+w, y+h-ry+ky, x+w-rx+kx, y+h, x+w-rx, y+h)...)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g l\n", x+rx, y+h)...)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", x+rx-kx, y+h, x, y+h-ry+ky, x, y+h-ry)...)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g l\n", x, y+ry)...)
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", x, y+ry-ky, x+rx-kx, y, x+rx, y)...)
		r.buf = append(r.buf, "h\n"...)
	}

	r.buf = append(r.buf, paint...)
	r.buf = append(r.buf, '\n')
	r.buf = append(r.buf, "Q\n"...)
}

func (r *Renderer) renderCircle(node *SVGNode) {
	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)
	paint := r.applyStyle(node)

	cx := parseFloatAttr(node, "cx", 0)
	cy := parseFloatAttr(node, "cy", 0)
	rad := parseFloatAttr(node, "r", 0)

	r.emitEllipsePath(cx, cy, rad, rad)

	r.buf = append(r.buf, paint...)
	r.buf = append(r.buf, '\n')
	r.buf = append(r.buf, "Q\n"...)
}

func (r *Renderer) renderEllipse(node *SVGNode) {
	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)
	paint := r.applyStyle(node)

	cx := parseFloatAttr(node, "cx", 0)
	cy := parseFloatAttr(node, "cy", 0)
	rx := parseFloatAttr(node, "rx", 0)
	ry := parseFloatAttr(node, "ry", 0)

	r.emitEllipsePath(cx, cy, rx, ry)

	r.buf = append(r.buf, paint...)
	r.buf = append(r.buf, '\n')
	r.buf = append(r.buf, "Q\n"...)
}

func (r *Renderer) emitEllipsePath(cx, cy, rx, ry float64) {
	// Approximate ellipse with 4 cubic bezier curves
	k := 0.5522847498
	kx := rx * k
	ky := ry * k

	r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g m\n", cx+rx, cy)...)
	r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", cx+rx, cy+ky, cx+kx, cy+ry, cx, cy+ry)...)
	r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", cx-kx, cy+ry, cx-rx, cy+ky, cx-rx, cy)...)
	r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", cx-rx, cy-ky, cx-kx, cy-ry, cx, cy-ry)...)
	r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", cx+kx, cy-ry, cx+rx, cy-ky, cx+rx, cy)...)
	r.buf = append(r.buf, "h\n"...)
}

func (r *Renderer) renderLine(node *SVGNode) {
	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)
	_ = r.applyStyle(node)

	x1 := parseFloatAttr(node, "x1", 0)
	y1 := parseFloatAttr(node, "y1", 0)
	x2 := parseFloatAttr(node, "x2", 0)
	y2 := parseFloatAttr(node, "y2", 0)

	r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g m\n%.4g %.4g l\nS\n", x1, y1, x2, y2)...)
	r.buf = append(r.buf, "Q\n"...)
}

func (r *Renderer) renderPolyline(node *SVGNode, closed bool) {
	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)
	paint := r.applyStyle(node)

	points := parsePoints(node.Attrs["points"])
	if len(points) < 2 {
		r.buf = append(r.buf, "Q\n"...)
		return
	}

	r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g m\n", points[0], points[1])...)
	for i := 2; i+1 < len(points); i += 2 {
		r.buf = append(r.buf, fmt.Sprintf("%.4g %.4g l\n", points[i], points[i+1])...)
	}
	if closed {
		r.buf = append(r.buf, "h\n"...)
	}

	r.buf = append(r.buf, paint...)
	r.buf = append(r.buf, '\n')
	r.buf = append(r.buf, "Q\n"...)
}

func (r *Renderer) renderPathElement(node *SVGNode) {
	d := node.Attrs["d"]
	if d == "" {
		return
	}
	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)
	paint := r.applyStyle(node)

	cmds := parsePath(d)
	renderPath(cmds, &r.buf)

	r.buf = append(r.buf, paint...)
	r.buf = append(r.buf, '\n')
	r.buf = append(r.buf, "Q\n"...)
}

func (r *Renderer) renderTextElement(node *SVGNode) {
	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)
	renderText(node, &r.buf)
	r.buf = append(r.buf, "Q\n"...)
}

func (r *Renderer) renderImage(node *SVGNode) {
	// Stub: image embedding requires external resource handling
	r.buf = append(r.buf, "% image element (stub)\n"...)
}

func (r *Renderer) renderUse(node *SVGNode) {
	href := node.Attrs["xlink:href"]
	if href == "" {
		href = node.Attrs["href"]
	}
	if href == "" {
		return
	}
	id := strings.TrimPrefix(href, "#")
	target, ok := r.defs[id]
	if !ok {
		return
	}

	r.buf = append(r.buf, "q\n"...)
	r.applyTransform(node)

	// Apply x/y as translation
	x := parseFloatAttr(node, "x", 0)
	y := parseFloatAttr(node, "y", 0)
	if x != 0 || y != 0 {
		m := Matrix{1, 0, 0, 1, x, y}
		r.buf = append(r.buf, fmt.Sprintf("%s\n", m.ToPDFOperator())...)
	}

	r.renderNode(target)
	r.buf = append(r.buf, "Q\n"...)
}

// Helper functions

func parseFloatAttr(node *SVGNode, name string, def float64) float64 {
	s := node.Attr(name)
	if s == "" {
		return def
	}
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSuffix(s, "pt")
	s = strings.TrimSuffix(s, "mm")
	s = strings.TrimSuffix(s, "cm")
	s = strings.TrimSuffix(s, "em")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parseFloatDefault(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return v
}

func parseDimension(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSuffix(s, "pt")
	s = strings.TrimSuffix(s, "%")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parsePoints(s string) []float64 {
	s = strings.ReplaceAll(s, ",", " ")
	fields := strings.Fields(s)
	points := make([]float64, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.ParseFloat(f, 64)
		if err == nil {
			points = append(points, v)
		}
	}
	return points
}

func extractURLRef(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "url(") {
		return ""
	}
	s = strings.TrimPrefix(s, "url(")
	s = strings.TrimSuffix(s, ")")
	s = strings.Trim(s, "'\"")
	s = strings.TrimPrefix(s, "#")
	return s
}

// Ensure math import is used
var _ = math.Pi
