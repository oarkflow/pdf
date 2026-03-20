package html

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ComputedStyle holds all computed CSS properties for a node.
type ComputedStyle struct {
	// Text
	FontFamily     string
	FontSize       float64 // in points
	FontWeight     int     // 100-900
	FontStyle      string  // normal, italic, oblique
	Color          [3]float64
	TextAlign      string // left, center, right, justify
	TextDecoration string // none, underline, line-through
	LineHeight     float64
	LetterSpacing  float64
	WordSpacing    float64
	TextIndent     float64
	TextTransform  string // none, uppercase, lowercase, capitalize
	WhiteSpace     string // normal, nowrap, pre, pre-wrap
	VerticalAlign  string

	// Box model
	Display       string // block, inline, inline-block, flex, grid, table, none, list-item
	Position      string
	Float         string
	Clear         string
	Width         CSSLength
	Height        CSSLength
	MinWidth      CSSLength
	MaxWidth      CSSLength
	MinHeight     CSSLength
	MaxHeight     CSSLength
	MarginTop     CSSLength
	MarginRight   CSSLength
	MarginBottom  CSSLength
	MarginLeft    CSSLength
	PaddingTop    CSSLength
	PaddingRight  CSSLength
	PaddingBottom CSSLength
	PaddingLeft   CSSLength

	BorderTopWidth    float64
	BorderRightWidth  float64
	BorderBottomWidth float64
	BorderLeftWidth   float64
	BorderTopColor    [3]float64
	BorderRightColor  [3]float64
	BorderBottomColor [3]float64
	BorderLeftColor   [3]float64
	BorderTopStyle    string
	BorderRightStyle  string
	BorderBottomStyle string
	BorderLeftStyle   string
	BorderRadius      float64

	// Background
	BackgroundColor *[3]float64
	BackgroundImage string

	// Flex
	FlexDirection  string
	FlexWrap       string
	JustifyContent string
	AlignItems     string
	FlexGrow       float64
	FlexShrink     float64
	FlexBasis      CSSLength
	Gap            float64

	// Grid
	GridTemplateColumns string
	GridTemplateRows    string
	GridColumn          string
	GridRow             string

	// List
	ListStyleType     string
	ListStylePosition string

	// Table
	BorderCollapse string
	BorderSpacing  float64

	// Visual
	Opacity    float64
	Overflow   string
	Visibility string
	ZIndex     int

	// Page
	PageBreakBefore string
	PageBreakAfter  string
	PageBreakInside string

	// Counters
	CounterReset     string
	CounterIncrement string
	Content          string

	// Custom properties
	CustomProperties map[string]string
}

// CSSLength represents a CSS length value with a unit.
type CSSLength struct {
	Value float64
	Unit  string // px, pt, em, rem, %, cm, mm, in, vw, vh, auto
}

// ToPoints converts a CSSLength to points.
func (l CSSLength) ToPoints(parentSize, rootFontSize float64) float64 {
	if l.Unit == "auto" {
		return 0
	}
	switch l.Unit {
	case "px":
		return l.Value * 0.75
	case "pt":
		return l.Value
	case "em":
		return l.Value * parentSize
	case "rem":
		return l.Value * rootFontSize
	case "%":
		return l.Value / 100 * parentSize
	case "cm":
		return l.Value * 28.3465
	case "mm":
		return l.Value * 2.83465
	case "in":
		return l.Value * 72
	case "vw", "vh":
		return l.Value / 100 * parentSize // approximation
	case "":
		if l.Value == 0 {
			return 0
		}
		return l.Value * 0.75 // treat as px
	default:
		return l.Value * 0.75
	}
}

// IsAuto returns true if the length is "auto".
func (l CSSLength) IsAuto() bool {
	return l.Unit == "auto"
}

// NewDefaultStyle creates a style with sensible defaults.
func NewDefaultStyle() *ComputedStyle {
	return &ComputedStyle{
		FontFamily:        "Helvetica",
		FontSize:          12,
		FontWeight:        400,
		FontStyle:         "normal",
		Color:             [3]float64{0.2, 0.2, 0.2}, // #333
		TextAlign:         "left",
		TextDecoration:    "none",
		LineHeight:        1.5,
		TextTransform:     "none",
		WhiteSpace:        "normal",
		VerticalAlign:     "baseline",
		Display:           "block",
		Position:          "static",
		Float:             "none",
		Clear:             "none",
		BorderTopStyle:    "none",
		BorderRightStyle:  "none",
		BorderBottomStyle: "none",
		BorderLeftStyle:   "none",
		FlexDirection:     "row",
		FlexWrap:          "nowrap",
		JustifyContent:    "flex-start",
		AlignItems:        "stretch",
		FlexShrink:        1,
		ListStyleType:     "disc",
		ListStylePosition: "outside",
		BorderCollapse:    "separate",
		Opacity:           1,
		Overflow:          "visible",
		Visibility:        "visible",
		PageBreakBefore:   "auto",
		PageBreakAfter:    "auto",
		PageBreakInside:   "auto",
		CustomProperties:  make(map[string]string),
	}
}

// InheritStyle creates a new style inheriting inheritable properties from parent.
func InheritStyle(parent *ComputedStyle) *ComputedStyle {
	s := NewDefaultStyle()
	if parent == nil {
		return s
	}
	// Inherited properties
	s.FontFamily = parent.FontFamily
	s.FontSize = parent.FontSize
	s.FontWeight = parent.FontWeight
	s.FontStyle = parent.FontStyle
	s.Color = parent.Color
	s.TextAlign = parent.TextAlign
	s.TextDecoration = parent.TextDecoration
	s.LineHeight = parent.LineHeight
	s.LetterSpacing = parent.LetterSpacing
	s.WordSpacing = parent.WordSpacing
	s.TextIndent = parent.TextIndent
	s.TextTransform = parent.TextTransform
	s.WhiteSpace = parent.WhiteSpace
	s.VerticalAlign = parent.VerticalAlign
	s.ListStyleType = parent.ListStyleType
	s.ListStylePosition = parent.ListStylePosition
	s.BorderCollapse = parent.BorderCollapse
	s.BorderSpacing = parent.BorderSpacing
	s.Visibility = parent.Visibility
	// Inherit custom properties
	s.CustomProperties = make(map[string]string)
	for k, v := range parent.CustomProperties {
		s.CustomProperties[k] = v
	}
	return s
}

// Apply applies CSS property values to this computed style.
func (s *ComputedStyle) Apply(properties map[string]CSSValue, parentStyle *ComputedStyle, rootFontSize float64) {
	parentFontSize := 12.0
	if parentStyle != nil {
		parentFontSize = parentStyle.FontSize
	}

	// First pass: collect custom properties
	for name, val := range properties {
		if strings.HasPrefix(name, "--") {
			s.CustomProperties[name] = val.Value
		}
	}

	// Second pass: apply all properties
	for name, val := range properties {
		if strings.HasPrefix(name, "--") {
			continue
		}
		value := s.resolveVar(val.Value)
		value = s.resolveCalc(value)

		switch name {
		case "font-family":
			s.FontFamily = stripQuotes(value)
		case "font-size":
			s.FontSize = parseFontSize(value, parentFontSize, rootFontSize)
		case "font-weight":
			s.FontWeight = parseFontWeight(value)
		case "font-style":
			s.FontStyle = value
		case "color":
			if c, ok := parseColor(value); ok {
				s.Color = c
			}
		case "text-align":
			s.TextAlign = value
		case "text-decoration":
			s.TextDecoration = value
		case "line-height":
			s.LineHeight = parseLineHeight(value, s.FontSize)
		case "letter-spacing":
			if value != "normal" {
				s.LetterSpacing = parseLengthValue(value, parentFontSize, rootFontSize)
			}
		case "word-spacing":
			if value != "normal" {
				s.WordSpacing = parseLengthValue(value, parentFontSize, rootFontSize)
			}
		case "text-indent":
			s.TextIndent = parseLengthValue(value, parentFontSize, rootFontSize)
		case "text-transform":
			s.TextTransform = value
		case "white-space":
			s.WhiteSpace = value
		case "vertical-align":
			s.VerticalAlign = value
		case "display":
			s.Display = value
		case "position":
			s.Position = value
		case "float":
			s.Float = value
		case "clear":
			s.Clear = value
		case "width":
			s.Width = parseLength(value)
		case "height":
			s.Height = parseLength(value)
		case "min-width":
			s.MinWidth = parseLength(value)
		case "max-width":
			s.MaxWidth = parseLength(value)
		case "min-height":
			s.MinHeight = parseLength(value)
		case "max-height":
			s.MaxHeight = parseLength(value)
		case "margin-top":
			s.MarginTop = parseLength(value)
		case "margin-right":
			s.MarginRight = parseLength(value)
		case "margin-bottom":
			s.MarginBottom = parseLength(value)
		case "margin-left":
			s.MarginLeft = parseLength(value)
		case "padding-top":
			s.PaddingTop = parseLength(value)
		case "padding-right":
			s.PaddingRight = parseLength(value)
		case "padding-bottom":
			s.PaddingBottom = parseLength(value)
		case "padding-left":
			s.PaddingLeft = parseLength(value)
		case "border-top-width":
			s.BorderTopWidth = parseBorderWidth(value, parentFontSize, rootFontSize)
		case "border-right-width":
			s.BorderRightWidth = parseBorderWidth(value, parentFontSize, rootFontSize)
		case "border-bottom-width":
			s.BorderBottomWidth = parseBorderWidth(value, parentFontSize, rootFontSize)
		case "border-left-width":
			s.BorderLeftWidth = parseBorderWidth(value, parentFontSize, rootFontSize)
		case "border-top-color":
			if c, ok := parseColor(value); ok {
				s.BorderTopColor = c
			}
		case "border-right-color":
			if c, ok := parseColor(value); ok {
				s.BorderRightColor = c
			}
		case "border-bottom-color":
			if c, ok := parseColor(value); ok {
				s.BorderBottomColor = c
			}
		case "border-left-color":
			if c, ok := parseColor(value); ok {
				s.BorderLeftColor = c
			}
		case "border-top-style":
			s.BorderTopStyle = value
		case "border-right-style":
			s.BorderRightStyle = value
		case "border-bottom-style":
			s.BorderBottomStyle = value
		case "border-left-style":
			s.BorderLeftStyle = value
		case "border-radius":
			s.BorderRadius = parseLengthValue(value, parentFontSize, rootFontSize)
		case "background-color":
			if c, ok := parseColor(value); ok {
				bg := c
				s.BackgroundColor = &bg
			}
		case "background-image":
			s.BackgroundImage = value
		case "flex-direction":
			s.FlexDirection = value
		case "flex-wrap":
			s.FlexWrap = value
		case "justify-content":
			s.JustifyContent = value
		case "align-items":
			s.AlignItems = value
		case "flex-grow":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				s.FlexGrow = v
			}
		case "flex-shrink":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				s.FlexShrink = v
			}
		case "flex-basis":
			s.FlexBasis = parseLength(value)
		case "gap":
			s.Gap = parseLengthValue(value, parentFontSize, rootFontSize)
		case "grid-template-columns":
			s.GridTemplateColumns = value
		case "grid-template-rows":
			s.GridTemplateRows = value
		case "grid-column":
			s.GridColumn = value
		case "grid-row":
			s.GridRow = value
		case "list-style-type":
			s.ListStyleType = value
		case "list-style-position":
			s.ListStylePosition = value
		case "border-collapse":
			s.BorderCollapse = value
		case "border-spacing":
			s.BorderSpacing = parseLengthValue(value, parentFontSize, rootFontSize)
		case "opacity":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				s.Opacity = v
			}
		case "overflow":
			s.Overflow = value
		case "visibility":
			s.Visibility = value
		case "z-index":
			if v, err := strconv.Atoi(value); err == nil {
				s.ZIndex = v
			}
		case "page-break-before":
			s.PageBreakBefore = value
		case "page-break-after":
			s.PageBreakAfter = value
		case "page-break-inside":
			s.PageBreakInside = value
		case "counter-reset":
			s.CounterReset = value
		case "counter-increment":
			s.CounterIncrement = value
		case "content":
			s.Content = value
		}
	}
}

func (s *ComputedStyle) resolveVar(value string) string {
	for strings.Contains(value, "var(") {
		idx := strings.Index(value, "var(")
		end := findClosingParen(value, idx+3)
		if end < 0 {
			break
		}
		inner := strings.TrimSpace(value[idx+4 : end])
		// Parse var(--name, fallback)
		varName := inner
		fallback := ""
		if commaIdx := strings.IndexByte(inner, ','); commaIdx >= 0 {
			varName = strings.TrimSpace(inner[:commaIdx])
			fallback = strings.TrimSpace(inner[commaIdx+1:])
		}
		resolved, ok := s.CustomProperties[varName]
		if !ok {
			resolved = fallback
		}
		value = value[:idx] + resolved + value[end+1:]
	}
	return value
}

func (s *ComputedStyle) resolveCalc(value string) string {
	if !strings.HasPrefix(value, "calc(") {
		return value
	}
	inner := value[5:]
	if idx := strings.LastIndexByte(inner, ')'); idx >= 0 {
		inner = inner[:idx]
	}
	// Simple calc: support addition and subtraction of same-unit values
	inner = strings.TrimSpace(inner)
	// Try to evaluate simple expressions
	result := evalSimpleCalc(inner)
	if result != "" {
		return result
	}
	return value
}

func evalSimpleCalc(expr string) string {
	expr = strings.TrimSpace(expr)
	// Try simple: value1 + value2 or value1 - value2
	for _, op := range []string{" + ", " - "} {
		if idx := strings.Index(expr, op); idx >= 0 {
			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+len(op):])
			lLen := parseLength(left)
			rLen := parseLength(right)
			if lLen.Unit == rLen.Unit || rLen.Unit == "" || lLen.Unit == "" {
				unit := lLen.Unit
				if unit == "" {
					unit = rLen.Unit
				}
				var result float64
				if op == " + " {
					result = lLen.Value + rLen.Value
				} else {
					result = lLen.Value - rLen.Value
				}
				return fmt.Sprintf("%g%s", result, unit)
			}
		}
	}
	return ""
}

func findClosingParen(s string, openIdx int) int {
	depth := 1
	for i := openIdx + 1; i < len(s); i++ {
		if s[i] == '(' {
			depth++
		} else if s[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func parseLength(s string) CSSLength {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || s == "0" {
		return CSSLength{0, "px"}
	}
	if s == "auto" {
		return CSSLength{0, "auto"}
	}
	if s == "none" {
		return CSSLength{0, ""}
	}

	units := []string{"rem", "em", "px", "pt", "%", "cm", "mm", "in", "vw", "vh"}
	for _, u := range units {
		if strings.HasSuffix(s, u) {
			numStr := strings.TrimSpace(s[:len(s)-len(u)])
			v, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return CSSLength{}
			}
			return CSSLength{v, u}
		}
	}

	// Try as plain number (treat as px)
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return CSSLength{v, "px"}
	}
	return CSSLength{}
}

func parseLengthValue(s string, parentFontSize, rootFontSize float64) float64 {
	l := parseLength(s)
	return l.ToPoints(parentFontSize, rootFontSize)
}

func parseFontSize(s string, parentFontSize, rootFontSize float64) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "xx-small":
		return 6.75
	case "x-small":
		return 7.5
	case "small":
		return 9.75
	case "medium":
		return 12
	case "large":
		return 13.5
	case "x-large":
		return 18
	case "xx-large":
		return 24
	case "smaller":
		return parentFontSize * 0.833
	case "larger":
		return parentFontSize * 1.2
	default:
		l := parseLength(s)
		return l.ToPoints(parentFontSize, rootFontSize)
	}
}

func parseFontWeight(s string) int {
	switch strings.ToLower(s) {
	case "normal":
		return 400
	case "bold":
		return 700
	case "bolder":
		return 700
	case "lighter":
		return 300
	default:
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
		return 400
	}
}

func parseLineHeight(s string, fontSize float64) float64 {
	s = strings.TrimSpace(s)
	if s == "normal" {
		return 1.5
	}
	// Try as unitless number (multiplier)
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v
	}
	// Try as length
	l := parseLength(s)
	if l.Unit == "%" {
		return l.Value / 100
	}
	pts := l.ToPoints(fontSize, fontSize)
	if pts > 0 && fontSize > 0 {
		return pts / fontSize
	}
	return 1.5
}

func parseBorderWidth(s string, parentFontSize, rootFontSize float64) float64 {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "thin":
		return 0.75
	case "medium":
		return 1.5
	case "thick":
		return 3
	default:
		return parseLengthValue(s, parentFontSize, rootFontSize)
	}
}

// parseColor parses a CSS color value to RGB [0-1].
func parseColor(s string) ([3]float64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))

	if s == "transparent" || s == "none" || s == "" || s == "currentcolor" {
		return [3]float64{}, false
	}

	// Named colors
	if c, ok := namedColors[s]; ok {
		return c, true
	}

	// Hex
	if strings.HasPrefix(s, "#") {
		return parseHexColor(s[1:])
	}

	// rgb/rgba
	if strings.HasPrefix(s, "rgb") {
		return parseRGBFunction(s)
	}

	return [3]float64{}, false
}

func parseHexColor(hex string) ([3]float64, bool) {
	switch len(hex) {
	case 3:
		r, _ := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		g, _ := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		b, _ := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, true
	case 4: // #RGBA
		r, _ := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		g, _ := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		b, _ := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, true
	case 6:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, true
	case 8: // #RRGGBBAA
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, true
	}
	return [3]float64{}, false
}

func parseRGBFunction(s string) ([3]float64, bool) {
	// rgb(r, g, b) or rgba(r, g, b, a)
	start := strings.IndexByte(s, '(')
	end := strings.LastIndexByte(s, ')')
	if start < 0 || end < 0 {
		return [3]float64{}, false
	}
	inner := s[start+1 : end]
	// Handle both comma and space separated
	inner = strings.ReplaceAll(inner, "/", ",")
	parts := strings.FieldsFunc(inner, func(r rune) bool { return r == ',' || r == ' ' })
	if len(parts) < 3 {
		return [3]float64{}, false
	}
	var rgb [3]float64
	for i := 0; i < 3; i++ {
		p := strings.TrimSpace(parts[i])
		if strings.HasSuffix(p, "%") {
			v, _ := strconv.ParseFloat(p[:len(p)-1], 64)
			rgb[i] = v / 100
		} else {
			v, _ := strconv.ParseFloat(p, 64)
			rgb[i] = v / 255
		}
		rgb[i] = math.Max(0, math.Min(1, rgb[i]))
	}
	return rgb, true
}

// namedColors maps CSS named colors to RGB [0-1].
var namedColors = map[string][3]float64{
	"black":       {0, 0, 0},
	"white":       {1, 1, 1},
	"red":         {1, 0, 0},
	"green":       {0, 0.502, 0},
	"blue":        {0, 0, 1},
	"yellow":      {1, 1, 0},
	"cyan":        {0, 1, 1},
	"magenta":     {1, 0, 1},
	"orange":      {1, 0.647, 0},
	"purple":      {0.502, 0, 0.502},
	"pink":        {1, 0.753, 0.796},
	"gray":        {0.502, 0.502, 0.502},
	"grey":        {0.502, 0.502, 0.502},
	"silver":      {0.753, 0.753, 0.753},
	"navy":        {0, 0, 0.502},
	"teal":        {0, 0.502, 0.502},
	"maroon":      {0.502, 0, 0},
	"olive":       {0.502, 0.502, 0},
	"lime":        {0, 1, 0},
	"aqua":        {0, 1, 1},
	"fuchsia":     {1, 0, 1},
	"brown":       {0.647, 0.165, 0.165},
	"coral":       {1, 0.498, 0.314},
	"crimson":     {0.863, 0.078, 0.235},
	"darkblue":    {0, 0, 0.545},
	"darkgray":    {0.663, 0.663, 0.663},
	"darkgrey":    {0.663, 0.663, 0.663},
	"darkgreen":   {0, 0.392, 0},
	"darkred":     {0.545, 0, 0},
	"gold":        {1, 0.843, 0},
	"indigo":      {0.294, 0, 0.510},
	"ivory":       {1, 1, 0.941},
	"khaki":       {0.941, 0.902, 0.549},
	"lavender":    {0.902, 0.902, 0.980},
	"lightblue":   {0.678, 0.847, 0.902},
	"lightgray":   {0.827, 0.827, 0.827},
	"lightgrey":   {0.827, 0.827, 0.827},
	"lightgreen":  {0.565, 0.933, 0.565},
	"lightyellow": {1, 1, 0.878},
	"tomato":      {1, 0.388, 0.278},
	"turquoise":   {0.251, 0.878, 0.816},
	"violet":      {0.933, 0.510, 0.933},
	"wheat":       {0.961, 0.871, 0.702},
	"whitesmoke":  {0.961, 0.961, 0.961},
	"yellowgreen": {0.604, 0.804, 0.196},
	"steelblue":   {0.275, 0.510, 0.706},
	"slategray":   {0.439, 0.502, 0.565},
	"slategrey":   {0.439, 0.502, 0.565},
	"skyblue":     {0.529, 0.808, 0.922},
	"salmon":      {0.980, 0.502, 0.447},
	"royalblue":   {0.255, 0.412, 0.882},
	"plum":        {0.867, 0.627, 0.867},
	"peru":        {0.804, 0.522, 0.247},
	"orchid":      {0.855, 0.439, 0.839},
	"linen":       {0.980, 0.941, 0.902},
	"chocolate":   {0.824, 0.412, 0.118},
	"beige":       {0.961, 0.961, 0.863},
	"azure":       {0.941, 1, 1},
	"aliceblue":   {0.941, 0.973, 1},
	"antiquewhite": {0.980, 0.922, 0.843},
	"bisque":       {1, 0.894, 0.769},
	"blanchedalmond": {1, 0.922, 0.804},
	"burlywood":     {0.871, 0.722, 0.529},
	"cadetblue":     {0.373, 0.620, 0.627},
	"chartreuse":    {0.498, 1, 0},
	"cornflowerblue": {0.392, 0.584, 0.929},
	"cornsilk":       {1, 0.973, 0.863},
	"darkcyan":       {0, 0.545, 0.545},
	"darkgoldenrod":  {0.722, 0.525, 0.043},
	"darkkhaki":      {0.741, 0.718, 0.420},
	"darkmagenta":    {0.545, 0, 0.545},
	"darkolivegreen": {0.333, 0.420, 0.184},
	"darkorange":     {1, 0.549, 0},
	"darkorchid":     {0.600, 0.196, 0.800},
	"darksalmon":     {0.914, 0.588, 0.478},
	"darkseagreen":   {0.561, 0.737, 0.561},
	"darkslateblue":  {0.282, 0.239, 0.545},
	"darkslategray":  {0.184, 0.310, 0.310},
	"darkslategrey":  {0.184, 0.310, 0.310},
	"darkturquoise":  {0, 0.808, 0.820},
	"darkviolet":     {0.580, 0, 0.827},
	"deeppink":       {1, 0.078, 0.576},
	"deepskyblue":    {0, 0.749, 1},
	"dimgray":        {0.412, 0.412, 0.412},
	"dimgrey":        {0.412, 0.412, 0.412},
	"dodgerblue":     {0.118, 0.565, 1},
	"firebrick":      {0.698, 0.133, 0.133},
	"floralwhite":    {1, 0.980, 0.941},
	"forestgreen":    {0.133, 0.545, 0.133},
	"gainsboro":      {0.863, 0.863, 0.863},
	"ghostwhite":     {0.973, 0.973, 1},
	"goldenrod":      {0.855, 0.647, 0.125},
	"greenyellow":    {0.678, 1, 0.184},
	"honeydew":       {0.941, 1, 0.941},
	"hotpink":        {1, 0.412, 0.706},
	"indianred":      {0.804, 0.361, 0.361},
	"lawngreen":      {0.486, 0.988, 0},
	"lemonchiffon":   {1, 0.980, 0.804},
	"lightcoral":     {0.941, 0.502, 0.502},
	"lightcyan":      {0.878, 1, 1},
	"lightpink":      {1, 0.714, 0.757},
	"lightsalmon":    {1, 0.627, 0.478},
	"lightseagreen":  {0.125, 0.698, 0.667},
	"lightskyblue":   {0.529, 0.808, 0.980},
	"lightslategray": {0.467, 0.533, 0.600},
	"lightslategrey": {0.467, 0.533, 0.600},
	"lightsteelblue": {0.690, 0.769, 0.871},
	"mediumaquamarine": {0.400, 0.804, 0.667},
	"mediumblue":       {0, 0, 0.804},
	"mediumorchid":     {0.729, 0.333, 0.827},
	"mediumpurple":     {0.576, 0.439, 0.859},
	"mediumseagreen":   {0.235, 0.702, 0.443},
	"mediumslateblue":  {0.482, 0.408, 0.933},
	"mediumspringgreen": {0, 0.980, 0.604},
	"mediumturquoise":   {0.282, 0.820, 0.800},
	"mediumvioletred":   {0.780, 0.082, 0.522},
	"midnightblue":      {0.098, 0.098, 0.439},
	"mintcream":          {0.961, 1, 0.980},
	"mistyrose":          {1, 0.894, 0.882},
	"moccasin":           {1, 0.894, 0.710},
	"navajowhite":        {1, 0.871, 0.678},
	"oldlace":            {0.992, 0.961, 0.902},
	"olivedrab":          {0.420, 0.557, 0.137},
	"orangered":          {1, 0.271, 0},
	"palegoldenrod":      {0.933, 0.910, 0.667},
	"palegreen":          {0.596, 0.984, 0.596},
	"paleturquoise":      {0.686, 0.933, 0.933},
	"palevioletred":      {0.859, 0.439, 0.576},
	"papayawhip":         {1, 0.937, 0.835},
	"peachpuff":          {1, 0.855, 0.725},
	"powderblue":         {0.690, 0.878, 0.902},
	"rebeccapurple":      {0.400, 0.200, 0.600},
	"rosybrown":          {0.737, 0.561, 0.561},
	"saddlebrown":        {0.545, 0.271, 0.075},
	"sandybrown":         {0.957, 0.643, 0.376},
	"seagreen":           {0.180, 0.545, 0.341},
	"seashell":           {1, 0.961, 0.933},
	"sienna":             {0.627, 0.322, 0.176},
	"snow":               {1, 0.980, 0.980},
	"springgreen":        {0, 1, 0.498},
	"tan":                {0.824, 0.706, 0.549},
	"thistle":            {0.847, 0.749, 0.847},
}
