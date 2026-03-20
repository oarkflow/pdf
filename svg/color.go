package svg

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// parseColor parses an SVG color string and returns RGB values (0-1), opacity, and success.
func parseColor(s string) ([3]float64, float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" {
		return [3]float64{}, 0, false
	}
	if s == "currentColor" {
		return [3]float64{0, 0, 0}, 1, true
	}

	// Named colors
	if c, ok := namedColors[strings.ToLower(s)]; ok {
		return c, 1, true
	}

	// Hex colors
	if strings.HasPrefix(s, "#") {
		return parseHexColor(s[1:])
	}

	// rgb() / rgba()
	if strings.HasPrefix(s, "rgba(") {
		return parseRGBA(s)
	}
	if strings.HasPrefix(s, "rgb(") {
		return parseRGB(s)
	}

	// hsl() / hsla()
	if strings.HasPrefix(s, "hsla(") || strings.HasPrefix(s, "hsl(") {
		return parseHSL(s)
	}

	return [3]float64{}, 0, false
}

func parseHexColor(hex string) ([3]float64, float64, bool) {
	switch len(hex) {
	case 3:
		r, _ := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		g, _ := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		b, _ := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, 1, true
	case 6:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, 1, true
	case 8:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		a, _ := strconv.ParseUint(hex[6:8], 16, 8)
		return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, float64(a) / 255, true
	}
	return [3]float64{}, 0, false
}

func parseRGB(s string) ([3]float64, float64, bool) {
	s = strings.TrimPrefix(s, "rgb(")
	s = strings.TrimSuffix(s, ")")
	parts := splitColorArgs(s)
	if len(parts) < 3 {
		return [3]float64{}, 0, false
	}
	r := parseColorComponent(parts[0])
	g := parseColorComponent(parts[1])
	b := parseColorComponent(parts[2])
	return [3]float64{r, g, b}, 1, true
}

func parseRGBA(s string) ([3]float64, float64, bool) {
	s = strings.TrimPrefix(s, "rgba(")
	s = strings.TrimSuffix(s, ")")
	parts := splitColorArgs(s)
	if len(parts) < 4 {
		return [3]float64{}, 0, false
	}
	r := parseColorComponent(parts[0])
	g := parseColorComponent(parts[1])
	b := parseColorComponent(parts[2])
	a, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	return [3]float64{r, g, b}, a, true
}

func parseHSL(s string) ([3]float64, float64, bool) {
	alpha := 1.0
	if strings.HasPrefix(s, "hsla(") {
		s = strings.TrimPrefix(s, "hsla(")
	} else {
		s = strings.TrimPrefix(s, "hsl(")
	}
	s = strings.TrimSuffix(s, ")")
	parts := splitColorArgs(s)
	if len(parts) < 3 {
		return [3]float64{}, 0, false
	}
	h, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	sl := strings.TrimSpace(parts[1])
	sl = strings.TrimSuffix(sl, "%")
	sVal, _ := strconv.ParseFloat(sl, 64)
	sVal /= 100
	ll := strings.TrimSpace(parts[2])
	ll = strings.TrimSuffix(ll, "%")
	lVal, _ := strconv.ParseFloat(ll, 64)
	lVal /= 100
	if len(parts) >= 4 {
		alpha, _ = strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	}
	r, g, b := hslToRGB(h/360, sVal, lVal)
	return [3]float64{r, g, b}, alpha, true
}

func splitColorArgs(s string) []string {
	// Handle both comma-separated and space-separated (with optional / for alpha)
	s = strings.ReplaceAll(s, "/", ",")
	parts := strings.Split(s, ",")
	if len(parts) >= 3 {
		return parts
	}
	return strings.Fields(s)
}

func parseColorComponent(s string) float64 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		return v / 100
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v / 255
}

func hslToRGB(h, s, l float64) (float64, float64, float64) {
	if s == 0 {
		return l, l, l
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	r := hueToRGB(p, q, h+1.0/3)
	g := hueToRGB(p, q, h)
	b := hueToRGB(p, q, h-1.0/3)
	return r, g, b
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6 {
		return p + (q-p)*6*t
	}
	if t < 0.5 {
		return q
	}
	if t < 2.0/3 {
		return p + (q-p)*(2.0/3-t)*6
	}
	return p
}

func formatColor(c [3]float64) string {
	return fmt.Sprintf("%.4g %.4g %.4g", c[0], c[1], c[2])
}

// clamp01 clamps a float64 to [0,1].
func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

// namedColors maps CSS named colors to RGB [0-1] values.
var namedColors = map[string][3]float64{
	"aliceblue":            {0.9412, 0.9725, 1},
	"antiquewhite":         {0.9804, 0.9216, 0.8431},
	"aqua":                 {0, 1, 1},
	"aquamarine":           {0.498, 1, 0.8314},
	"azure":                {0.9412, 1, 1},
	"beige":                {0.9608, 0.9608, 0.8627},
	"bisque":               {1, 0.8941, 0.7686},
	"black":                {0, 0, 0},
	"blanchedalmond":       {1, 0.9216, 0.8039},
	"blue":                 {0, 0, 1},
	"blueviolet":           {0.5412, 0.1686, 0.8863},
	"brown":                {0.6471, 0.1647, 0.1647},
	"burlywood":            {0.8706, 0.7216, 0.5294},
	"cadetblue":            {0.3725, 0.6196, 0.6275},
	"chartreuse":           {0.498, 1, 0},
	"chocolate":            {0.8235, 0.4118, 0.1176},
	"coral":                {1, 0.498, 0.3137},
	"cornflowerblue":       {0.3922, 0.5843, 0.9294},
	"cornsilk":             {1, 0.9725, 0.8627},
	"crimson":              {0.8627, 0.0784, 0.2353},
	"cyan":                 {0, 1, 1},
	"darkblue":             {0, 0, 0.5451},
	"darkcyan":             {0, 0.5451, 0.5451},
	"darkgoldenrod":        {0.7216, 0.5255, 0.0431},
	"darkgray":             {0.6627, 0.6627, 0.6627},
	"darkgreen":            {0, 0.3922, 0},
	"darkgrey":             {0.6627, 0.6627, 0.6627},
	"darkkhaki":            {0.7412, 0.7176, 0.4196},
	"darkmagenta":          {0.5451, 0, 0.5451},
	"darkolivegreen":       {0.3333, 0.4196, 0.1843},
	"darkorange":           {1, 0.549, 0},
	"darkorchid":           {0.6, 0.1961, 0.8},
	"darkred":              {0.5451, 0, 0},
	"darksalmon":           {0.9137, 0.5882, 0.4784},
	"darkseagreen":         {0.5608, 0.7373, 0.5608},
	"darkslateblue":        {0.2824, 0.2392, 0.5451},
	"darkslategray":        {0.1843, 0.3098, 0.3098},
	"darkslategrey":        {0.1843, 0.3098, 0.3098},
	"darkturquoise":        {0, 0.8078, 0.8196},
	"darkviolet":           {0.5804, 0, 0.8275},
	"deeppink":             {1, 0.0784, 0.5765},
	"deepskyblue":          {0, 0.749, 1},
	"dimgray":              {0.4118, 0.4118, 0.4118},
	"dimgrey":              {0.4118, 0.4118, 0.4118},
	"dodgerblue":           {0.1176, 0.5647, 1},
	"firebrick":            {0.698, 0.1333, 0.1333},
	"floralwhite":          {1, 0.9804, 0.9412},
	"forestgreen":          {0.1333, 0.5451, 0.1333},
	"fuchsia":              {1, 0, 1},
	"gainsboro":            {0.8627, 0.8627, 0.8627},
	"ghostwhite":           {0.9725, 0.9725, 1},
	"gold":                 {1, 0.8431, 0},
	"goldenrod":            {0.8549, 0.6471, 0.1255},
	"gray":                 {0.502, 0.502, 0.502},
	"green":                {0, 0.502, 0},
	"greenyellow":          {0.6784, 1, 0.1843},
	"grey":                 {0.502, 0.502, 0.502},
	"honeydew":             {0.9412, 1, 0.9412},
	"hotpink":              {1, 0.4118, 0.7059},
	"indianred":            {0.8039, 0.3608, 0.3608},
	"indigo":               {0.2941, 0, 0.5098},
	"ivory":                {1, 1, 0.9412},
	"khaki":                {0.9412, 0.902, 0.549},
	"lavender":             {0.902, 0.902, 0.9608},
	"lavenderblush":        {1, 0.9412, 0.9608},
	"lawngreen":            {0.4863, 0.9882, 0},
	"lemonchiffon":         {1, 0.9804, 0.8039},
	"lightblue":            {0.6784, 0.8471, 0.902},
	"lightcoral":           {0.9412, 0.502, 0.502},
	"lightcyan":            {0.8784, 1, 1},
	"lightgoldenrodyellow": {0.9804, 0.9804, 0.8235},
	"lightgray":            {0.8275, 0.8275, 0.8275},
	"lightgreen":           {0.5647, 0.9333, 0.5647},
	"lightgrey":            {0.8275, 0.8275, 0.8275},
	"lightpink":            {1, 0.7137, 0.7569},
	"lightsalmon":          {1, 0.6275, 0.4784},
	"lightseagreen":        {0.1255, 0.698, 0.6667},
	"lightskyblue":         {0.5294, 0.8078, 0.9804},
	"lightslategray":       {0.4667, 0.5333, 0.6},
	"lightslategrey":       {0.4667, 0.5333, 0.6},
	"lightsteelblue":       {0.6902, 0.7686, 0.8706},
	"lightyellow":          {1, 1, 0.8784},
	"lime":                 {0, 1, 0},
	"limegreen":            {0.1961, 0.8039, 0.1961},
	"linen":                {0.9804, 0.9412, 0.902},
	"magenta":              {1, 0, 1},
	"maroon":               {0.502, 0, 0},
	"mediumaquamarine":     {0.4, 0.8039, 0.6667},
	"mediumblue":           {0, 0, 0.8039},
	"mediumorchid":         {0.7294, 0.3333, 0.8275},
	"mediumpurple":         {0.5765, 0.4392, 0.8588},
	"mediumseagreen":       {0.2353, 0.702, 0.4431},
	"mediumslateblue":      {0.4824, 0.4078, 0.9333},
	"mediumspringgreen":    {0, 0.9804, 0.6039},
	"mediumturquoise":      {0.2824, 0.8196, 0.8},
	"mediumvioletred":      {0.7804, 0.0824, 0.5216},
	"midnightblue":         {0.098, 0.098, 0.4392},
	"mintcream":            {0.9608, 1, 0.9804},
	"mistyrose":            {1, 0.8941, 0.8824},
	"moccasin":             {1, 0.8941, 0.7098},
	"navajowhite":          {1, 0.8706, 0.6784},
	"navy":                 {0, 0, 0.502},
	"oldlace":              {0.9922, 0.9608, 0.902},
	"olive":                {0.502, 0.502, 0},
	"olivedrab":            {0.4196, 0.5569, 0.1373},
	"orange":               {1, 0.6471, 0},
	"orangered":            {1, 0.2706, 0},
	"orchid":               {0.8549, 0.4392, 0.8392},
	"palegoldenrod":        {0.9333, 0.9098, 0.6667},
	"palegreen":            {0.5961, 0.9843, 0.5961},
	"paleturquoise":        {0.6863, 0.9333, 0.9333},
	"palevioletred":        {0.8588, 0.4392, 0.5765},
	"papayawhip":           {1, 0.9373, 0.8353},
	"peachpuff":            {1, 0.8549, 0.7255},
	"peru":                 {0.8039, 0.5216, 0.2471},
	"pink":                 {1, 0.7529, 0.7961},
	"plum":                 {0.8667, 0.6275, 0.8667},
	"powderblue":           {0.6902, 0.8784, 0.902},
	"purple":               {0.502, 0, 0.502},
	"rebeccapurple":        {0.4, 0.2, 0.6},
	"red":                  {1, 0, 0},
	"rosybrown":            {0.7373, 0.5608, 0.5608},
	"royalblue":            {0.2549, 0.4118, 0.8824},
	"saddlebrown":          {0.5451, 0.2706, 0.0745},
	"salmon":               {0.9804, 0.502, 0.4471},
	"sandybrown":           {0.9569, 0.6431, 0.3765},
	"seagreen":             {0.1804, 0.5451, 0.3412},
	"seashell":             {1, 0.9608, 0.9333},
	"sienna":               {0.6275, 0.3216, 0.1765},
	"silver":               {0.7529, 0.7529, 0.7529},
	"skyblue":              {0.5294, 0.8078, 0.9216},
	"slateblue":            {0.4157, 0.3529, 0.8039},
	"slategray":            {0.4392, 0.502, 0.5647},
	"slategrey":            {0.4392, 0.502, 0.5647},
	"snow":                 {1, 0.9804, 0.9804},
	"springgreen":          {0, 1, 0.498},
	"steelblue":            {0.2745, 0.5098, 0.7059},
	"tan":                  {0.8235, 0.7059, 0.549},
	"teal":                 {0, 0.502, 0.502},
	"thistle":              {0.8471, 0.749, 0.8471},
	"tomato":               {1, 0.3882, 0.2784},
	"turquoise":            {0.251, 0.8784, 0.8157},
	"violet":               {0.9333, 0.5098, 0.9333},
	"wheat":                {0.9608, 0.8706, 0.702},
	"white":                {1, 1, 1},
	"whitesmoke":           {0.9608, 0.9608, 0.9608},
	"yellow":               {1, 1, 0},
	"yellowgreen":          {0.6039, 0.8039, 0.1961},
}
