// Package tailwind provides a comprehensive Tailwind CSS class parser
// that converts Tailwind utility classes into CSS property maps.
package tailwind

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser holds configuration and parses Tailwind CSS classes.
type Parser struct {
	// CustomTheme allows extending/overriding default theme values
	CustomTheme map[string]map[string]string
}

// New creates a new Parser with optional custom theme overrides.
func New(customTheme ...map[string]map[string]string) *Parser {
	p := &Parser{}
	if len(customTheme) > 0 {
		p.CustomTheme = customTheme[0]
	}
	return p
}

// Parse takes a string of space-separated Tailwind class names and returns
// a map of CSS property names to their values.
func (p *Parser) Parse(classes string) map[string]any {
	result := make(map[string]any)
	for _, class := range strings.Fields(classes) {
		class = strings.TrimSpace(class)
		if class == "" {
			continue
		}
		// Strip variants (responsive, state, etc.) for now — return them annotated
		base, variant := splitVariant(class)
		props := p.parseClass(base)
		if variant != "" {
			// Nest under variant key
			existing, ok := result[":variant:"+variant]
			if !ok {
				existing = make(map[string]any)
			}
			for k, v := range props {
				existing.(map[string]any)[k] = v
			}
			result[":variant:"+variant] = existing
		} else {
			for k, v := range props {
				result[k] = v
			}
		}
	}
	return result
}

// ParseWithVariants returns a structured result including variant groupings.
func (p *Parser) ParseWithVariants(classes string) map[string]any {
	return p.Parse(classes)
}

// splitVariant splits "md:flex" -> ("flex", "md"), "hover:bg-red-500" -> ("bg-red-500", "hover")
// Handles chained variants like "dark:hover:text-white" -> ("text-white", "dark:hover")
func splitVariant(class string) (base, variant string) {
	// Find last colon that separates variant from base class
	// But careful: arbitrary values like bg-[url('a:b')] should not be split
	inBracket := 0
	lastColon := -1
	for i, ch := range class {
		switch ch {
		case '[':
			inBracket++
		case ']':
			inBracket--
		case ':':
			if inBracket == 0 {
				lastColon = i
			}
		}
	}
	if lastColon == -1 {
		return class, ""
	}
	return class[lastColon+1:], class[:lastColon]
}

// parseClass parses a single (non-variant) Tailwind class and returns CSS properties.
func (p *Parser) parseClass(class string) map[string]any {
	result := make(map[string]any)

	// Handle arbitrary CSS: [property:value]
	if strings.HasPrefix(class, "[") && strings.HasSuffix(class, "]") {
		inner := class[1 : len(class)-1]
		if idx := strings.Index(inner, ":"); idx != -1 {
			prop := strings.ReplaceAll(inner[:idx], "_", " ")
			val := strings.ReplaceAll(inner[idx+1:], "_", " ")
			result[prop] = val
		}
		return result
	}

	// Try all category parsers in order
	parsers := []func(string) map[string]any{
		p.parseLayout,
		p.parseFlexbox,
		p.parseGrid,
		p.parseSpacing,
		p.parseSizing,
		p.parseTypography,
		p.parseBackground,
		p.parseBorder,
		p.parseEffects,
		p.parseFilters,
		p.parseTables,
		p.parseTransitions,
		p.parseTransforms,
		p.parseInteractivity,
		p.parseSVG,
		p.parseAccessibility,
		p.parsePosition,
		p.parseDisplay,
		p.parseOverflow,
		p.parseZIndex,
		p.parseOpacity,
		p.parseShadow,
		p.parseRing,
		p.parseOutline,
		p.parseListStyle,
		p.parseAppearance,
		p.parseCursor,
		p.parseAnimation,
		p.parseArbitrary,
	}

	for _, fn := range parsers {
		if props := fn(class); len(props) > 0 {
			return props
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Theme helpers
// ---------------------------------------------------------------------------

var defaultSpacing = map[string]string{
	"0": "0px", "px": "1px", "0.5": "0.125rem", "1": "0.25rem",
	"1.5": "0.375rem", "2": "0.5rem", "2.5": "0.625rem", "3": "0.75rem",
	"3.5": "0.875rem", "4": "1rem", "5": "1.25rem", "6": "1.5rem",
	"7": "1.75rem", "8": "2rem", "9": "2.25rem", "10": "2.5rem",
	"11": "2.75rem", "12": "3rem", "14": "3.5rem", "16": "4rem",
	"20": "5rem", "24": "6rem", "28": "7rem", "32": "8rem",
	"36": "9rem", "40": "10rem", "44": "11rem", "48": "12rem",
	"52": "13rem", "56": "14rem", "60": "15rem", "64": "16rem",
	"72": "18rem", "80": "20rem", "96": "24rem",
	"auto": "auto", "1/2": "50%", "1/3": "33.333333%", "2/3": "66.666667%",
	"1/4": "25%", "2/4": "50%", "3/4": "75%", "1/5": "20%", "2/5": "40%",
	"3/5": "60%", "4/5": "80%", "1/6": "16.666667%", "2/6": "33.333333%",
	"3/6": "50%", "4/6": "66.666667%", "5/6": "83.333333%",
	"full": "100%", "min": "min-content",
	"max": "max-content", "fit": "fit-content",
}

var defaultColors = map[string]map[string]string{
	"inherit":     {"DEFAULT": "inherit"},
	"current":     {"DEFAULT": "currentColor"},
	"transparent": {"DEFAULT": "transparent"},
	"black":       {"DEFAULT": "#000000"},
	"white":       {"DEFAULT": "#ffffff"},
	"slate": {
		"50": "#f8fafc", "100": "#f1f5f9", "200": "#e2e8f0", "300": "#cbd5e1",
		"400": "#94a3b8", "500": "#64748b", "600": "#475569", "700": "#334155",
		"800": "#1e293b", "900": "#0f172a", "950": "#020617",
	},
	"gray": {
		"50": "#f9fafb", "100": "#f3f4f6", "200": "#e5e7eb", "300": "#d1d5db",
		"400": "#9ca3af", "500": "#6b7280", "600": "#4b5563", "700": "#374151",
		"800": "#1f2937", "900": "#111827", "950": "#030712",
	},
	"zinc": {
		"50": "#fafafa", "100": "#f4f4f5", "200": "#e4e4e7", "300": "#d4d4d8",
		"400": "#a1a1aa", "500": "#71717a", "600": "#52525b", "700": "#3f3f46",
		"800": "#27272a", "900": "#18181b", "950": "#09090b",
	},
	"neutral": {
		"50": "#fafafa", "100": "#f5f5f5", "200": "#e5e5e5", "300": "#d4d4d4",
		"400": "#a3a3a3", "500": "#737373", "600": "#525252", "700": "#404040",
		"800": "#262626", "900": "#171717", "950": "#0a0a0a",
	},
	"stone": {
		"50": "#fafaf9", "100": "#f5f5f4", "200": "#e7e5e4", "300": "#d6d3d1",
		"400": "#a8a29e", "500": "#78716c", "600": "#57534e", "700": "#44403c",
		"800": "#292524", "900": "#1c1917", "950": "#0c0a09",
	},
	"red": {
		"50": "#fef2f2", "100": "#fee2e2", "200": "#fecaca", "300": "#fca5a5",
		"400": "#f87171", "500": "#ef4444", "600": "#dc2626", "700": "#b91c1c",
		"800": "#991b1b", "900": "#7f1d1d", "950": "#450a0a",
	},
	"orange": {
		"50": "#fff7ed", "100": "#ffedd5", "200": "#fed7aa", "300": "#fdba74",
		"400": "#fb923c", "500": "#f97316", "600": "#ea580c", "700": "#c2410c",
		"800": "#9a3412", "900": "#7c2d12", "950": "#431407",
	},
	"amber": {
		"50": "#fffbeb", "100": "#fef3c7", "200": "#fde68a", "300": "#fcd34d",
		"400": "#fbbf24", "500": "#f59e0b", "600": "#d97706", "700": "#b45309",
		"800": "#92400e", "900": "#78350f", "950": "#451a03",
	},
	"yellow": {
		"50": "#fefce8", "100": "#fef9c3", "200": "#fef08a", "300": "#fde047",
		"400": "#facc15", "500": "#eab308", "600": "#ca8a04", "700": "#a16207",
		"800": "#854d0e", "900": "#713f12", "950": "#422006",
	},
	"lime": {
		"50": "#f7fee7", "100": "#ecfccb", "200": "#d9f99d", "300": "#bef264",
		"400": "#a3e635", "500": "#84cc16", "600": "#65a30d", "700": "#4d7c0f",
		"800": "#3f6212", "900": "#365314", "950": "#1a2e05",
	},
	"green": {
		"50": "#f0fdf4", "100": "#dcfce7", "200": "#bbf7d0", "300": "#86efac",
		"400": "#4ade80", "500": "#22c55e", "600": "#16a34a", "700": "#15803d",
		"800": "#166534", "900": "#14532d", "950": "#052e16",
	},
	"emerald": {
		"50": "#ecfdf5", "100": "#d1fae5", "200": "#a7f3d0", "300": "#6ee7b7",
		"400": "#34d399", "500": "#10b981", "600": "#059669", "700": "#047857",
		"800": "#065f46", "900": "#064e3b", "950": "#022c22",
	},
	"teal": {
		"50": "#f0fdfa", "100": "#ccfbf1", "200": "#99f6e4", "300": "#5eead4",
		"400": "#2dd4bf", "500": "#14b8a6", "600": "#0d9488", "700": "#0f766e",
		"800": "#115e59", "900": "#134e4a", "950": "#042f2e",
	},
	"cyan": {
		"50": "#ecfeff", "100": "#cffafe", "200": "#a5f3fc", "300": "#67e8f9",
		"400": "#22d3ee", "500": "#06b6d4", "600": "#0891b2", "700": "#0e7490",
		"800": "#155e75", "900": "#164e63", "950": "#083344",
	},
	"sky": {
		"50": "#f0f9ff", "100": "#e0f2fe", "200": "#bae6fd", "300": "#7dd3fc",
		"400": "#38bdf8", "500": "#0ea5e9", "600": "#0284c7", "700": "#0369a1",
		"800": "#075985", "900": "#0c4a6e", "950": "#082f49",
	},
	"blue": {
		"50": "#eff6ff", "100": "#dbeafe", "200": "#bfdbfe", "300": "#93c5fd",
		"400": "#60a5fa", "500": "#3b82f6", "600": "#2563eb", "700": "#1d4ed8",
		"800": "#1e40af", "900": "#1e3a8a", "950": "#172554",
	},
	"indigo": {
		"50": "#eef2ff", "100": "#e0e7ff", "200": "#c7d2fe", "300": "#a5b4fc",
		"400": "#818cf8", "500": "#6366f1", "600": "#4f46e5", "700": "#4338ca",
		"800": "#3730a3", "900": "#312e81", "950": "#1e1b4b",
	},
	"violet": {
		"50": "#f5f3ff", "100": "#ede9fe", "200": "#ddd6fe", "300": "#c4b5fd",
		"400": "#a78bfa", "500": "#8b5cf6", "600": "#7c3aed", "700": "#6d28d9",
		"800": "#5b21b6", "900": "#4c1d95", "950": "#2e1065",
	},
	"purple": {
		"50": "#faf5ff", "100": "#f3e8ff", "200": "#e9d5ff", "300": "#d8b4fe",
		"400": "#c084fc", "500": "#a855f7", "600": "#9333ea", "700": "#7e22ce",
		"800": "#6b21a8", "900": "#581c87", "950": "#3b0764",
	},
	"fuchsia": {
		"50": "#fdf4ff", "100": "#fae8ff", "200": "#f5d0fe", "300": "#f0abfc",
		"400": "#e879f9", "500": "#d946ef", "600": "#c026d3", "700": "#a21caf",
		"800": "#86198f", "900": "#701a75", "950": "#4a044e",
	},
	"pink": {
		"50": "#fdf2f8", "100": "#fce7f3", "200": "#fbcfe8", "300": "#f9a8d4",
		"400": "#f472b6", "500": "#ec4899", "600": "#db2777", "700": "#be185d",
		"800": "#9d174d", "900": "#831843", "950": "#500724",
	},
	"rose": {
		"50": "#fff1f2", "100": "#ffe4e6", "200": "#fecdd3", "300": "#fda4af",
		"400": "#fb7185", "500": "#f43f5e", "600": "#e11d48", "700": "#be123c",
		"800": "#9f1239", "900": "#881337", "950": "#4c0519",
	},
}

// resolveColor returns the CSS color value for a color name + shade like "red-500" or "blue"
func resolveColor(token string) (string, bool) {
	// Arbitrary value [#abc]
	if strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]") {
		return token[1 : len(token)-1], true
	}
	parts := strings.SplitN(token, "-", 2)
	colorName := parts[0]
	shade := "DEFAULT"
	if len(parts) == 2 {
		shade = parts[1]
	}
	if shades, ok := defaultColors[colorName]; ok {
		if val, ok2 := shades[shade]; ok2 {
			return val, true
		}
	}
	return "", false
}

// resolveSpacing returns the CSS spacing value for a token like "4", "px", "1/2"
func resolveSpacing(token string) (string, bool) {
	if strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]") {
		return token[1 : len(token)-1], true
	}
	if val, ok := defaultSpacing[token]; ok {
		return val, true
	}
	return "", false
}

// resolveOpacity returns the opacity value for tokens like "50", "75", etc.
func resolveOpacity(token string) string {
	if strings.HasPrefix(token, "[") && strings.HasSuffix(token, "]") {
		return token[1 : len(token)-1]
	}
	opacities := map[string]string{
		"0": "0", "5": "0.05", "10": "0.1", "15": "0.15", "20": "0.2",
		"25": "0.25", "30": "0.3", "35": "0.35", "40": "0.4", "45": "0.45",
		"50": "0.5", "55": "0.55", "60": "0.6", "65": "0.65", "70": "0.7",
		"75": "0.75", "80": "0.8", "85": "0.85", "90": "0.9", "95": "0.95",
		"100": "1",
	}
	if v, ok := opacities[token]; ok {
		return v
	}
	return token
}

// withColorAndOpacity handles color classes with optional /opacity suffix
// e.g. "red-500" or "red-500/50"
func withColorAndOpacity(token string) (color string, opacity string, ok bool) {
	parts := strings.SplitN(token, "/", 2)
	colorToken := parts[0]
	color, ok = resolveColor(colorToken)
	if !ok {
		return
	}
	if len(parts) == 2 {
		opacity = resolveOpacity(parts[1])
	}
	return
}

// ---------------------------------------------------------------------------
// Display
// ---------------------------------------------------------------------------

func (p *Parser) parseDisplay(class string) map[string]any {
	displays := map[string]string{
		"block": "block", "inline-block": "inline-block", "inline": "inline",
		"flex": "flex", "inline-flex": "inline-flex", "table": "table",
		"inline-table": "inline-table", "table-caption": "table-caption",
		"table-cell": "table-cell", "table-column": "table-column",
		"table-column-group": "table-column-group", "table-footer-group": "table-footer-group",
		"table-header-group": "table-header-group", "table-row-group": "table-row-group",
		"table-row": "table-row", "flow-root": "flow-root", "grid": "grid",
		"inline-grid": "inline-grid", "contents": "contents", "list-item": "list-item",
		"hidden": "none",
	}
	if v, ok := displays[class]; ok {
		return map[string]any{"display": v}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Layout
// ---------------------------------------------------------------------------

func (p *Parser) parseLayout(class string) map[string]any {
	// Container
	if class == "container" {
		return map[string]any{"width": "100%", "max-width": "100%"}
	}

	// Box sizing
	if class == "box-border" {
		return map[string]any{"box-sizing": "border-box"}
	}
	if class == "box-content" {
		return map[string]any{"box-sizing": "content-box"}
	}

	// Float
	floats := map[string]string{
		"float-right": "right", "float-left": "left", "float-none": "none",
		"float-start": "inline-start", "float-end": "inline-end",
	}
	if v, ok := floats[class]; ok {
		return map[string]any{"float": v}
	}

	// Clear
	clears := map[string]string{
		"clear-left": "left", "clear-right": "right", "clear-both": "both",
		"clear-none": "none", "clear-start": "inline-start", "clear-end": "inline-end",
	}
	if v, ok := clears[class]; ok {
		return map[string]any{"clear": v}
	}

	// Isolation
	if class == "isolate" {
		return map[string]any{"isolation": "isolate"}
	}
	if class == "isolation-auto" {
		return map[string]any{"isolation": "auto"}
	}

	// Object fit
	objFit := map[string]string{
		"object-contain":    "contain",
		"object-cover":      "cover",
		"object-fill":       "fill",
		"object-none":       "none",
		"object-scale-down": "scale-down",
	}
	if v, ok := objFit[class]; ok {
		return map[string]any{"object-fit": v}
	}

	// Object position
	objPos := map[string]string{
		"object-bottom":       "bottom",
		"object-center":       "center",
		"object-left":         "left",
		"object-left-bottom":  "left bottom",
		"object-left-top":     "left top",
		"object-right":        "right",
		"object-right-bottom": "right bottom",
		"object-right-top":    "right top",
		"object-top":          "top",
	}
	if v, ok := objPos[class]; ok {
		return map[string]any{"object-position": v}
	}

	// Visibility
	if class == "visible" {
		return map[string]any{"visibility": "visible"}
	}
	if class == "invisible" {
		return map[string]any{"visibility": "hidden"}
	}
	if class == "collapse" {
		return map[string]any{"visibility": "collapse"}
	}

	// Aspect ratio
	aspectRatios := map[string]string{
		"aspect-auto":   "auto",
		"aspect-square": "1 / 1",
		"aspect-video":  "16 / 9",
	}
	if v, ok := aspectRatios[class]; ok {
		return map[string]any{"aspect-ratio": v}
	}
	if rest, ok := stripPrefix(class, "aspect-"); ok {
		if num, den, ok := parseFraction(rest); ok && den != "0" {
			return map[string]any{"aspect-ratio": fmt.Sprintf("%s / %s", num, den)}
		}
	}
	if strings.HasPrefix(class, "aspect-[") {
		val := extractArbitrary(class, "aspect-")
		return map[string]any{"aspect-ratio": val}
	}

	// Columns
	if rest, ok := stripPrefix(class, "columns-"); ok {
		colMap := map[string]string{
			"auto": "auto", "1": "1", "2": "2", "3": "3", "4": "4",
			"5": "5", "6": "6", "7": "7", "8": "8", "9": "9", "10": "10",
			"11": "11", "12": "12", "3xs": "16rem", "2xs": "18rem",
			"xs": "20rem", "sm": "24rem", "md": "28rem", "lg": "32rem",
			"xl": "36rem", "2xl": "42rem", "3xl": "48rem", "4xl": "56rem",
			"5xl": "64rem", "6xl": "72rem", "7xl": "80rem",
		}
		if v, ok := colMap[rest]; ok {
			return map[string]any{"columns": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"columns": arb}
		}
	}

	// Break
	breakBefore := map[string]string{
		"break-before-auto":         "auto",
		"break-before-avoid":        "avoid",
		"break-before-all":          "all",
		"break-before-avoid-page":   "avoid-page",
		"break-before-page":         "page",
		"break-before-left":         "left",
		"break-before-right":        "right",
		"break-before-column":       "column",
	}
	if v, ok := breakBefore[class]; ok {
		return map[string]any{"break-before": v}
	}
	breakAfter := map[string]string{
		"break-after-auto":       "auto",
		"break-after-avoid":      "avoid",
		"break-after-all":        "all",
		"break-after-avoid-page": "avoid-page",
		"break-after-page":       "page",
		"break-after-left":       "left",
		"break-after-right":      "right",
		"break-after-column":     "column",
	}
	if v, ok := breakAfter[class]; ok {
		return map[string]any{"break-after": v}
	}
	breakInside := map[string]string{
		"break-inside-auto":         "auto",
		"break-inside-avoid":        "avoid",
		"break-inside-avoid-page":   "avoid-page",
		"break-inside-avoid-column": "avoid-column",
	}
	if v, ok := breakInside[class]; ok {
		return map[string]any{"break-inside": v}
	}

	// Break word/all
	if class == "break-normal" {
		return map[string]any{"overflow-wrap": "normal", "word-break": "normal"}
	}
	if class == "break-words" {
		return map[string]any{"overflow-wrap": "break-word"}
	}
	if class == "break-all" {
		return map[string]any{"word-break": "break-all"}
	}
	if class == "break-keep" {
		return map[string]any{"word-break": "keep-all"}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Position
// ---------------------------------------------------------------------------

func (p *Parser) parsePosition(class string) map[string]any {
	positions := map[string]string{
		"static": "static", "fixed": "fixed", "absolute": "absolute",
		"relative": "relative", "sticky": "sticky",
	}
	if v, ok := positions[class]; ok {
		return map[string]any{"position": v}
	}

	// Inset / top / right / bottom / left
	type insetDef struct {
		prefix string
		props  []string
	}
	insets := []insetDef{
		{"inset-", []string{"top", "right", "bottom", "left"}},
		{"inset-x-", []string{"left", "right"}},
		{"inset-y-", []string{"top", "bottom"}},
		{"start-", []string{"inset-inline-start"}},
		{"end-", []string{"inset-inline-end"}},
		{"top-", []string{"top"}},
		{"right-", []string{"right"}},
		{"bottom-", []string{"bottom"}},
		{"left-", []string{"left"}},
	}

	for _, id := range insets {
		if rest, ok := stripPrefix(class, id.prefix); ok {
			neg := false
			if strings.HasPrefix(rest, "-") {
				neg = true
				rest = rest[1:]
			}
			val, ok2 := resolveSpacing(rest)
			if !ok2 {
				val = rest
			}
			if neg && val != "auto" {
				val = negate(val)
			}
			result := make(map[string]any)
			for _, prop := range id.props {
				result[prop] = val
			}
			return result
		}
		// negative prefix: -inset-
		negPrefix := "-" + id.prefix
		if rest, ok := stripPrefix(class, negPrefix); ok {
			val, ok2 := resolveSpacing(rest)
			if !ok2 {
				val = rest
			}
			result := make(map[string]any)
			for _, prop := range id.props {
				result[prop] = negate(val)
			}
			return result
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Overflow
// ---------------------------------------------------------------------------

func (p *Parser) parseOverflow(class string) map[string]any {
	overflowMap := map[string]string{
		"overflow-auto":    "auto",
		"overflow-hidden":  "hidden",
		"overflow-clip":    "clip",
		"overflow-visible": "visible",
		"overflow-scroll":  "scroll",
		"overflow-x-auto":    "auto",
		"overflow-x-hidden":  "hidden",
		"overflow-x-clip":    "clip",
		"overflow-x-visible": "visible",
		"overflow-x-scroll":  "scroll",
		"overflow-y-auto":    "auto",
		"overflow-y-hidden":  "hidden",
		"overflow-y-clip":    "clip",
		"overflow-y-visible": "visible",
		"overflow-y-scroll":  "scroll",
	}
	if v, ok := overflowMap[class]; ok {
		if strings.Contains(class, "-x-") {
			return map[string]any{"overflow-x": v}
		}
		if strings.Contains(class, "-y-") {
			return map[string]any{"overflow-y": v}
		}
		return map[string]any{"overflow": v}
	}

	// Overscroll
	overscrollMap := map[string]string{
		"overscroll-auto":     "auto",
		"overscroll-contain":  "contain",
		"overscroll-none":     "none",
		"overscroll-x-auto":    "auto",
		"overscroll-x-contain": "contain",
		"overscroll-x-none":    "none",
		"overscroll-y-auto":    "auto",
		"overscroll-y-contain": "contain",
		"overscroll-y-none":    "none",
	}
	if v, ok := overscrollMap[class]; ok {
		if strings.Contains(class, "-x-") {
			return map[string]any{"overscroll-behavior-x": v}
		}
		if strings.Contains(class, "-y-") {
			return map[string]any{"overscroll-behavior-y": v}
		}
		return map[string]any{"overscroll-behavior": v}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Z-Index
// ---------------------------------------------------------------------------

func (p *Parser) parseZIndex(class string) map[string]any {
	zIndexes := map[string]string{
		"z-auto": "auto", "z-0": "0", "z-10": "10", "z-20": "20",
		"z-30": "30", "z-40": "40", "z-50": "50",
	}
	if v, ok := zIndexes[class]; ok {
		return map[string]any{"z-index": v}
	}
	if rest, ok := stripPrefix(class, "z-"); ok {
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"z-index": arb}
		}
		if _, err := strconv.Atoi(rest); err == nil {
			return map[string]any{"z-index": rest}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Flexbox
// ---------------------------------------------------------------------------

func (p *Parser) parseFlexbox(class string) map[string]any {
	// Flex direction
	flexDir := map[string]string{
		"flex-row": "row", "flex-row-reverse": "row-reverse",
		"flex-col": "column", "flex-col-reverse": "column-reverse",
	}
	if v, ok := flexDir[class]; ok {
		return map[string]any{"flex-direction": v}
	}

	// Flex wrap
	flexWrap := map[string]string{
		"flex-wrap": "wrap", "flex-wrap-reverse": "wrap-reverse", "flex-nowrap": "nowrap",
	}
	if v, ok := flexWrap[class]; ok {
		return map[string]any{"flex-wrap": v}
	}

	// Flex grow / shrink
	if class == "grow" || class == "flex-grow" {
		return map[string]any{"flex-grow": "1"}
	}
	if class == "grow-0" || class == "flex-grow-0" {
		return map[string]any{"flex-grow": "0"}
	}
	if class == "shrink" || class == "flex-shrink" {
		return map[string]any{"flex-shrink": "1"}
	}
	if class == "shrink-0" || class == "flex-shrink-0" {
		return map[string]any{"flex-shrink": "0"}
	}

	// Flex shorthand
	flexShort := map[string]string{
		"flex-1":    "1 1 0%",
		"flex-auto": "1 1 auto",
		"flex-initial": "0 1 auto",
		"flex-none": "none",
	}
	if v, ok := flexShort[class]; ok {
		return map[string]any{"flex": v}
	}
	if rest, ok := stripPrefix(class, "flex-["); ok {
		rest = strings.TrimSuffix(rest, "]")
		return map[string]any{"flex": rest}
	}

	// Order
	if rest, ok := stripPrefix(class, "order-"); ok {
		orders := map[string]string{
			"first": "-9999", "last": "9999", "none": "0",
		}
		if v, ok2 := orders[rest]; ok2 {
			return map[string]any{"order": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"order": arb}
		}
		if _, err := strconv.Atoi(rest); err == nil {
			return map[string]any{"order": rest}
		}
	}

	// Basis
	if rest, ok := stripPrefix(class, "basis-"); ok {
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"flex-basis": arb}
		}
		if val, ok2 := resolveSpacing(rest); ok2 {
			return map[string]any{"flex-basis": val}
		}
	}

	// Justify content
	justifyContent := map[string]string{
		"justify-normal":        "normal",
		"justify-start":         "flex-start",
		"justify-end":           "flex-end",
		"justify-center":        "center",
		"justify-between":       "space-between",
		"justify-around":        "space-around",
		"justify-evenly":        "space-evenly",
		"justify-stretch":       "stretch",
	}
	if v, ok := justifyContent[class]; ok {
		return map[string]any{"justify-content": v}
	}

	// Justify items
	justifyItems := map[string]string{
		"justify-items-start":   "start",
		"justify-items-end":     "end",
		"justify-items-center":  "center",
		"justify-items-stretch": "stretch",
	}
	if v, ok := justifyItems[class]; ok {
		return map[string]any{"justify-items": v}
	}

	// Justify self
	justifySelf := map[string]string{
		"justify-self-auto":    "auto",
		"justify-self-start":   "start",
		"justify-self-end":     "end",
		"justify-self-center":  "center",
		"justify-self-stretch": "stretch",
	}
	if v, ok := justifySelf[class]; ok {
		return map[string]any{"justify-self": v}
	}

	// Align content
	alignContent := map[string]string{
		"content-normal":  "normal",
		"content-center":  "center",
		"content-start":   "flex-start",
		"content-end":     "flex-end",
		"content-between": "space-between",
		"content-around":  "space-around",
		"content-evenly":  "space-evenly",
		"content-baseline": "baseline",
		"content-stretch": "stretch",
	}
	if v, ok := alignContent[class]; ok {
		return map[string]any{"align-content": v}
	}

	// Align items
	alignItems := map[string]string{
		"items-start":    "flex-start",
		"items-end":      "flex-end",
		"items-center":   "center",
		"items-baseline": "baseline",
		"items-stretch":  "stretch",
	}
	if v, ok := alignItems[class]; ok {
		return map[string]any{"align-items": v}
	}

	// Align self
	alignSelf := map[string]string{
		"self-auto":     "auto",
		"self-start":    "flex-start",
		"self-end":      "flex-end",
		"self-center":   "center",
		"self-stretch":  "stretch",
		"self-baseline": "baseline",
	}
	if v, ok := alignSelf[class]; ok {
		return map[string]any{"align-self": v}
	}

	// Place content
	placeContent := map[string]string{
		"place-content-center":  "center",
		"place-content-start":   "start",
		"place-content-end":     "end",
		"place-content-between": "space-between",
		"place-content-around":  "space-around",
		"place-content-evenly":  "space-evenly",
		"place-content-baseline": "baseline",
		"place-content-stretch": "stretch",
	}
	if v, ok := placeContent[class]; ok {
		return map[string]any{"place-content": v}
	}

	// Place items
	placeItems := map[string]string{
		"place-items-start":   "start",
		"place-items-end":     "end",
		"place-items-center":  "center",
		"place-items-baseline": "baseline",
		"place-items-stretch": "stretch",
	}
	if v, ok := placeItems[class]; ok {
		return map[string]any{"place-items": v}
	}

	// Place self
	placeSelf := map[string]string{
		"place-self-auto":    "auto",
		"place-self-start":   "start",
		"place-self-end":     "end",
		"place-self-center":  "center",
		"place-self-stretch": "stretch",
	}
	if v, ok := placeSelf[class]; ok {
		return map[string]any{"place-self": v}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Grid
// ---------------------------------------------------------------------------

func (p *Parser) parseGrid(class string) map[string]any {
	// Grid template columns
	if rest, ok := stripPrefix(class, "grid-cols-"); ok {
		cols := map[string]string{
			"1": "repeat(1, minmax(0, 1fr))",
			"2": "repeat(2, minmax(0, 1fr))",
			"3": "repeat(3, minmax(0, 1fr))",
			"4": "repeat(4, minmax(0, 1fr))",
			"5": "repeat(5, minmax(0, 1fr))",
			"6": "repeat(6, minmax(0, 1fr))",
			"7": "repeat(7, minmax(0, 1fr))",
			"8": "repeat(8, minmax(0, 1fr))",
			"9": "repeat(9, minmax(0, 1fr))",
			"10": "repeat(10, minmax(0, 1fr))",
			"11": "repeat(11, minmax(0, 1fr))",
			"12": "repeat(12, minmax(0, 1fr))",
			"none": "none",
			"subgrid": "subgrid",
		}
		if v, ok2 := cols[rest]; ok2 {
			return map[string]any{"grid-template-columns": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-template-columns": arb}
		}
	}

	// Grid template rows
	if rest, ok := stripPrefix(class, "grid-rows-"); ok {
		rows := map[string]string{
			"1": "repeat(1, minmax(0, 1fr))", "2": "repeat(2, minmax(0, 1fr))",
			"3": "repeat(3, minmax(0, 1fr))", "4": "repeat(4, minmax(0, 1fr))",
			"5": "repeat(5, minmax(0, 1fr))", "6": "repeat(6, minmax(0, 1fr))",
			"none": "none", "subgrid": "subgrid",
		}
		if v, ok2 := rows[rest]; ok2 {
			return map[string]any{"grid-template-rows": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-template-rows": arb}
		}
	}

	// Grid column span
	if rest, ok := stripPrefix(class, "col-span-"); ok {
		if rest == "full" {
			return map[string]any{"grid-column": "1 / -1"}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-column": arb}
		}
		return map[string]any{"grid-column": fmt.Sprintf("span %s / span %s", rest, rest)}
	}
	if rest, ok := stripPrefix(class, "col-start-"); ok {
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-column-start": arb}
		}
		return map[string]any{"grid-column-start": rest}
	}
	if rest, ok := stripPrefix(class, "col-end-"); ok {
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-column-end": arb}
		}
		return map[string]any{"grid-column-end": rest}
	}
	if class == "col-auto" {
		return map[string]any{"grid-column": "auto"}
	}

	// Grid row span
	if rest, ok := stripPrefix(class, "row-span-"); ok {
		if rest == "full" {
			return map[string]any{"grid-row": "1 / -1"}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-row": arb}
		}
		return map[string]any{"grid-row": fmt.Sprintf("span %s / span %s", rest, rest)}
	}
	if rest, ok := stripPrefix(class, "row-start-"); ok {
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-row-start": arb}
		}
		return map[string]any{"grid-row-start": rest}
	}
	if rest, ok := stripPrefix(class, "row-end-"); ok {
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-row-end": arb}
		}
		return map[string]any{"grid-row-end": rest}
	}
	if class == "row-auto" {
		return map[string]any{"grid-row": "auto"}
	}

	// Grid auto flow
	gridAutoFlow := map[string]string{
		"grid-flow-row":         "row",
		"grid-flow-col":         "column",
		"grid-flow-dense":       "dense",
		"grid-flow-row-dense":   "row dense",
		"grid-flow-col-dense":   "column dense",
	}
	if v, ok := gridAutoFlow[class]; ok {
		return map[string]any{"grid-auto-flow": v}
	}

	// Grid auto columns
	if rest, ok := stripPrefix(class, "auto-cols-"); ok {
		autoCols := map[string]string{
			"auto": "auto", "min": "min-content",
			"max": "max-content", "fr": "minmax(0, 1fr)",
		}
		if v, ok2 := autoCols[rest]; ok2 {
			return map[string]any{"grid-auto-columns": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-auto-columns": arb}
		}
	}

	// Grid auto rows
	if rest, ok := stripPrefix(class, "auto-rows-"); ok {
		autoRows := map[string]string{
			"auto": "auto", "min": "min-content",
			"max": "max-content", "fr": "minmax(0, 1fr)",
		}
		if v, ok2 := autoRows[rest]; ok2 {
			return map[string]any{"grid-auto-rows": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"grid-auto-rows": arb}
		}
	}

	// Gap
	if rest, ok := stripPrefix(class, "gap-x-"); ok {
		if val, ok2 := resolveSpacing(rest); ok2 {
			return map[string]any{"column-gap": val}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"column-gap": arb}
		}
	}
	if rest, ok := stripPrefix(class, "gap-y-"); ok {
		if val, ok2 := resolveSpacing(rest); ok2 {
			return map[string]any{"row-gap": val}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"row-gap": arb}
		}
	}
	if rest, ok := stripPrefix(class, "gap-"); ok {
		if val, ok2 := resolveSpacing(rest); ok2 {
			return map[string]any{"gap": val}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"gap": arb}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Spacing (margin, padding)
// ---------------------------------------------------------------------------

func (p *Parser) parseSpacing(class string) map[string]any {
	type spaceDef struct {
		prefix string
		props  []string
	}

	spaceDefs := []spaceDef{
		{"p-", []string{"padding"}},
		{"px-", []string{"padding-left", "padding-right"}},
		{"py-", []string{"padding-top", "padding-bottom"}},
		{"ps-", []string{"padding-inline-start"}},
		{"pe-", []string{"padding-inline-end"}},
		{"pt-", []string{"padding-top"}},
		{"pr-", []string{"padding-right"}},
		{"pb-", []string{"padding-bottom"}},
		{"pl-", []string{"padding-left"}},
		{"m-", []string{"margin"}},
		{"mx-", []string{"margin-left", "margin-right"}},
		{"my-", []string{"margin-top", "margin-bottom"}},
		{"ms-", []string{"margin-inline-start"}},
		{"me-", []string{"margin-inline-end"}},
		{"mt-", []string{"margin-top"}},
		{"mr-", []string{"margin-right"}},
		{"mb-", []string{"margin-bottom"}},
		{"ml-", []string{"margin-left"}},
	}

	for _, sd := range spaceDefs {
		if rest, ok := stripPrefix(class, sd.prefix); ok {
			neg := false
			if strings.HasPrefix(rest, "-") {
				neg = true
				rest = rest[1:]
			}
			var val string
			if v, ok2 := resolveSpacing(rest); ok2 {
				val = v
			} else if arb := extractArbitraryValue(rest); arb != "" {
				val = arb
			} else {
				continue
			}
			if neg && val != "auto" {
				val = negate(val)
			}
			result := make(map[string]any)
			for _, prop := range sd.props {
				result[prop] = val
			}
			return result
		}
		// Negative margin shorthand: -m-, -mx-, etc.
		negPrefix := "-" + sd.prefix
		if rest, ok := stripPrefix(class, negPrefix); ok {
			var val string
			if v, ok2 := resolveSpacing(rest); ok2 {
				val = negate(v)
			} else if arb := extractArbitraryValue(rest); arb != "" {
				val = negate(arb)
			} else {
				continue
			}
			result := make(map[string]any)
			for _, prop := range sd.props {
				result[prop] = val
			}
			return result
		}
	}

	// Space between (uses CSS gap or margin trick — we output the CSS var approach)
	if rest, ok := stripPrefix(class, "space-x-"); ok {
		val, _ := resolveSpacing(rest)
		if arb := extractArbitraryValue(rest); arb != "" {
			val = arb
		}
		if val != "" {
			return map[string]any{
				"--tw-space-x-reverse": "0",
				"margin-right":         fmt.Sprintf("calc(%s * var(--tw-space-x-reverse))", val),
				"margin-left":          fmt.Sprintf("calc(%s * calc(1 - var(--tw-space-x-reverse)))", val),
			}
		}
	}
	if rest, ok := stripPrefix(class, "space-y-"); ok {
		val, _ := resolveSpacing(rest)
		if arb := extractArbitraryValue(rest); arb != "" {
			val = arb
		}
		if val != "" {
			return map[string]any{
				"--tw-space-y-reverse": "0",
				"margin-bottom":        fmt.Sprintf("calc(%s * var(--tw-space-y-reverse))", val),
				"margin-top":           fmt.Sprintf("calc(%s * calc(1 - var(--tw-space-y-reverse)))", val),
			}
		}
	}
	if class == "space-x-reverse" {
		return map[string]any{"--tw-space-x-reverse": "1"}
	}
	if class == "space-y-reverse" {
		return map[string]any{"--tw-space-y-reverse": "1"}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Sizing
// ---------------------------------------------------------------------------

func (p *Parser) parseSizing(class string) map[string]any {
	sizingPrefixes := []struct {
		prefix string
		prop   string
	}{
		{"w-", "width"}, {"min-w-", "min-width"}, {"max-w-", "max-width"},
		{"h-", "height"}, {"min-h-", "min-height"}, {"max-h-", "max-height"},
		{"size-", ""},
	}

	extraW := map[string]string{
		"screen": "100vw", "svw": "100svw", "lvw": "100lvw", "dvw": "100dvw",
		"min":    "min-content", "max": "max-content", "fit": "fit-content",
		"auto":   "auto",
	}
	extraH := map[string]string{
		"screen": "100vh", "svh": "100svh", "lvh": "100lvh", "dvh": "100dvh",
		"min":    "min-content", "max": "max-content", "fit": "fit-content",
		"auto":   "auto",
	}

	// Named max-width sizes
	maxWNamed := map[string]string{
		"xs": "20rem", "sm": "24rem", "md": "28rem", "lg": "32rem",
		"xl": "36rem", "2xl": "42rem", "3xl": "48rem", "4xl": "56rem",
		"5xl": "64rem", "6xl": "72rem", "7xl": "80rem",
		"prose": "65ch", "screen-sm": "640px", "screen-md": "768px",
		"screen-lg": "1024px", "screen-xl": "1280px", "screen-2xl": "1536px",
		"none": "none", "full": "100%", "min": "min-content", "max": "max-content",
		"fit": "fit-content",
	}

	for _, s := range sizingPrefixes {
		if rest, ok := stripPrefix(class, s.prefix); ok {
			var val string
			if arb := extractArbitraryValue(rest); arb != "" {
				val = arb
			} else {
				// Check extra keywords first (screen differs for w vs h)
				extras := extraW
				if strings.HasPrefix(s.prop, "h") || strings.HasPrefix(s.prop, "min-h") || strings.HasPrefix(s.prop, "max-h") {
					extras = extraH
				}
				if v2, ok3 := extras[rest]; ok3 {
					val = v2
				} else if s.prop == "max-width" {
					if v2, ok3 := maxWNamed[rest]; ok3 {
						val = v2
					}
				}
				if val == "" {
					if v, ok2 := resolveSpacing(rest); ok2 {
						val = v
					}
				}
				if val == "" {
					continue
				}
			}
			if s.prop == "" {
				// size-* sets both width and height
				return map[string]any{"width": val, "height": val}
			}
			return map[string]any{s.prop: val}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Typography
// ---------------------------------------------------------------------------

func (p *Parser) parseTypography(class string) map[string]any {
	// Font family
	fontFamily := map[string]string{
		"font-sans":  `ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"`,
		"font-serif": `ui-serif, Georgia, Cambria, "Times New Roman", Times, serif`,
		"font-mono":  `ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace`,
	}
	if v, ok := fontFamily[class]; ok {
		return map[string]any{"font-family": v}
	}

	// Font size
	if rest, ok := stripPrefix(class, "text-"); ok {
		fontSizes := map[string][2]string{
			"xs":   {"0.75rem", "1rem"},
			"sm":   {"0.875rem", "1.25rem"},
			"base": {"1rem", "1.5rem"},
			"lg":   {"1.125rem", "1.75rem"},
			"xl":   {"1.25rem", "1.75rem"},
			"2xl":  {"1.5rem", "2rem"},
			"3xl":  {"1.875rem", "2.25rem"},
			"4xl":  {"2.25rem", "2.5rem"},
			"5xl":  {"3rem", "1"},
			"6xl":  {"3.75rem", "1"},
			"7xl":  {"4.5rem", "1"},
			"8xl":  {"6rem", "1"},
			"9xl":  {"8rem", "1"},
		}
		if v, ok2 := fontSizes[rest]; ok2 {
			return map[string]any{"font-size": v[0], "line-height": v[1]}
		}
		// Text color
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			result := map[string]any{"color": color}
			if opacity != "" {
				result["--tw-text-opacity"] = opacity
				result["color"] = applyOpacity(color, opacity)
			}
			return result
		}
		// Arbitrary font size
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"font-size": arb}
		}
		// Text alignment
		alignments := map[string]string{
			"left": "left", "center": "center", "right": "right",
			"justify": "justify", "start": "start", "end": "end",
		}
		if v, ok2 := alignments[rest]; ok2 {
			return map[string]any{"text-align": v}
		}
		// Text decoration
		decorations := map[string]string{
			"inherit": "inherit", "current": "currentColor",
			"transparent": "transparent",
		}
		if v, ok2 := decorations[rest]; ok2 {
			return map[string]any{"text-decoration-color": v}
		}
		// Text wrap
		wraps := map[string]string{
			"wrap": "wrap", "nowrap": "nowrap", "balance": "balance",
			"pretty": "pretty", "stable": "stable",
		}
		if v, ok2 := wraps[rest]; ok2 {
			return map[string]any{"text-wrap": v}
		}
		// Text overflow
		if rest == "ellipsis" {
			return map[string]any{"text-overflow": "ellipsis"}
		}
		if rest == "clip" {
			return map[string]any{"text-overflow": "clip"}
		}
	}

	// Font weight
	if rest, ok := stripPrefix(class, "font-"); ok {
		weights := map[string]string{
			"thin": "100", "extralight": "200", "light": "300",
			"normal": "400", "medium": "500", "semibold": "600",
			"bold": "700", "extrabold": "800", "black": "900",
		}
		if v, ok2 := weights[rest]; ok2 {
			return map[string]any{"font-weight": v}
		}
		fontStyles := map[string]string{
			"italic": "italic", "not-italic": "normal",
		}
		if v, ok2 := fontStyles[rest]; ok2 {
			return map[string]any{"font-style": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"font-weight": arb}
		}
	}

	// Font style (top-level)
	if class == "italic" {
		return map[string]any{"font-style": "italic"}
	}
	if class == "not-italic" {
		return map[string]any{"font-style": "normal"}
	}

	// Font variant numeric
	numericVariants := map[string]string{
		"normal-nums":        "normal", "ordinal": "ordinal",
		"slashed-zero":       "slashed-zero", "lining-nums": "lining-nums",
		"oldstyle-nums":      "oldstyle-nums", "proportional-nums": "proportional-nums",
		"tabular-nums":       "tabular-nums", "diagonal-fractions": "diagonal-fractions",
		"stacked-fractions":  "stacked-fractions",
	}
	if v, ok := numericVariants[class]; ok {
		return map[string]any{"font-variant-numeric": v}
	}

	// Leading (line-height)
	if rest, ok := stripPrefix(class, "leading-"); ok {
		leadings := map[string]string{
			"none": "1", "tight": "1.25", "snug": "1.375",
			"normal": "1.5", "relaxed": "1.625", "loose": "2",
			"3": ".75rem", "4": "1rem", "5": "1.25rem", "6": "1.5rem",
			"7": "1.75rem", "8": "2rem", "9": "2.25rem", "10": "2.5rem",
		}
		if v, ok2 := leadings[rest]; ok2 {
			return map[string]any{"line-height": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"line-height": arb}
		}
	}

	// Tracking (letter-spacing)
	if rest, ok := stripPrefix(class, "tracking-"); ok {
		trackings := map[string]string{
			"tighter": "-0.05em", "tight": "-0.025em", "normal": "0em",
			"wide": "0.025em", "wider": "0.05em", "widest": "0.1em",
		}
		if v, ok2 := trackings[rest]; ok2 {
			return map[string]any{"letter-spacing": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"letter-spacing": arb}
		}
	}

	// Decoration
	decorationClasses := map[string]map[string]any{
		"underline":       {"text-decoration-line": "underline"},
		"overline":        {"text-decoration-line": "overline"},
		"line-through":    {"text-decoration-line": "line-through"},
		"no-underline":    {"text-decoration-line": "none"},
	}
	if v, ok := decorationClasses[class]; ok {
		return v
	}

	// Decoration style
	if rest, ok := stripPrefix(class, "decoration-"); ok {
		styles := map[string]string{
			"solid": "solid", "double": "double", "dotted": "dotted",
			"dashed": "dashed", "wavy": "wavy",
		}
		if v, ok2 := styles[rest]; ok2 {
			return map[string]any{"text-decoration-style": v}
		}
		// Decoration thickness
		thicknesses := map[string]string{
			"auto": "auto", "from-font": "from-font",
			"0": "0px", "1": "1px", "2": "2px", "4": "4px", "8": "8px",
		}
		if v, ok2 := thicknesses[rest]; ok2 {
			return map[string]any{"text-decoration-thickness": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"text-decoration-thickness": arb}
		}
		// Decoration color
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			result := map[string]any{"text-decoration-color": color}
			if opacity != "" {
				result["text-decoration-color"] = applyOpacity(color, opacity)
			}
			return result
		}
	}

	// Text transform
	transforms := map[string]string{
		"uppercase": "uppercase", "lowercase": "lowercase",
		"capitalize": "capitalize", "normal-case": "none",
	}
	if v, ok := transforms[class]; ok {
		return map[string]any{"text-transform": v}
	}

	// Text overflow
	if class == "truncate" {
		return map[string]any{
			"overflow":      "hidden",
			"text-overflow": "ellipsis",
			"white-space":   "nowrap",
		}
	}
	if class == "text-ellipsis" {
		return map[string]any{"text-overflow": "ellipsis"}
	}
	if class == "text-clip" {
		return map[string]any{"text-overflow": "clip"}
	}

	// Whitespace
	if rest, ok := stripPrefix(class, "whitespace-"); ok {
		ws := map[string]string{
			"normal": "normal", "nowrap": "nowrap", "pre": "pre",
			"pre-line": "pre-line", "pre-wrap": "pre-wrap", "break-spaces": "break-spaces",
		}
		if v, ok2 := ws[rest]; ok2 {
			return map[string]any{"white-space": v}
		}
	}

	// Antialiased
	if class == "antialiased" {
		return map[string]any{
			"-webkit-font-smoothing": "antialiased",
			"-moz-osx-font-smoothing": "grayscale",
		}
	}
	if class == "subpixel-antialiased" {
		return map[string]any{
			"-webkit-font-smoothing": "auto",
			"-moz-osx-font-smoothing": "auto",
		}
	}

	// Hyphens
	if rest, ok := stripPrefix(class, "hyphens-"); ok {
		hyph := map[string]string{"none": "none", "manual": "manual", "auto": "auto"}
		if v, ok2 := hyph[rest]; ok2 {
			return map[string]any{"hyphens": v}
		}
	}

	// Text indent
	if rest, ok := stripPrefix(class, "indent-"); ok {
		if val, ok2 := resolveSpacing(rest); ok2 {
			return map[string]any{"text-indent": val}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"text-indent": arb}
		}
	}

	// Vertical align
	if rest, ok := stripPrefix(class, "align-"); ok {
		aligns := map[string]string{
			"baseline": "baseline", "top": "top", "middle": "middle",
			"bottom": "bottom", "text-top": "text-top", "text-bottom": "text-bottom",
			"sub": "sub", "super": "super",
		}
		if v, ok2 := aligns[rest]; ok2 {
			return map[string]any{"vertical-align": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"vertical-align": arb}
		}
	}

	// Underline offset
	if rest, ok := stripPrefix(class, "underline-offset-"); ok {
		offsets := map[string]string{"auto": "auto", "0": "0px", "1": "1px", "2": "2px", "4": "4px", "8": "8px"}
		if v, ok2 := offsets[rest]; ok2 {
			return map[string]any{"text-underline-offset": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"text-underline-offset": arb}
		}
	}

	// Line clamp
	if rest, ok := stripPrefix(class, "line-clamp-"); ok {
		if rest == "none" {
			return map[string]any{
				"-webkit-line-clamp": "unset",
				"overflow":            "visible",
			}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{
				"overflow":            "hidden",
				"display":             "-webkit-box",
				"-webkit-box-orient": "vertical",
				"-webkit-line-clamp": arb,
			}
		}
		return map[string]any{
			"overflow":            "hidden",
			"display":             "-webkit-box",
			"-webkit-box-orient": "vertical",
			"-webkit-line-clamp": rest,
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Background
// ---------------------------------------------------------------------------

func (p *Parser) parseBackground(class string) map[string]any {
	// Background color
	if rest, ok := stripPrefix(class, "bg-"); ok {
		// Gradient positions
		gradFrom := map[string]string{
			"from-inherit":     "inherit",
			"from-current":     "currentColor",
			"from-transparent": "transparent",
		}
		if v, ok2 := gradFrom[class]; ok2 {
			return map[string]any{"--tw-gradient-from": v}
		}

		// Special bg values
		special := map[string]string{
			"inherit":     "inherit",
			"current":     "currentColor",
			"transparent": "transparent",
			"none":        "none",
		}
		if v, ok2 := special[rest]; ok2 {
			return map[string]any{"background-color": v}
		}

		// Arbitrary background value (before color, so bg-[#ff0000] uses background shorthand)
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"background": arb}
		}

		// Color with optional opacity
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			result := map[string]any{"background-color": color}
			if opacity != "" {
				result["--tw-bg-opacity"] = opacity
				result["background-color"] = applyOpacity(color, opacity)
			}
			return result
		}

		// Background attachment
		attachments := map[string]string{
			"fixed": "fixed", "local": "local", "scroll": "scroll",
		}
		if v, ok2 := attachments[rest]; ok2 {
			return map[string]any{"background-attachment": v}
		}

		// Background clip
		clips := map[string]string{
			"clip-border": "border-box", "clip-padding": "padding-box",
			"clip-content": "content-box", "clip-text": "text",
		}
		if v, ok2 := clips[rest]; ok2 {
			return map[string]any{"background-clip": v}
		}

		// Background origin
		origins := map[string]string{
			"origin-border": "border-box", "origin-padding": "padding-box",
			"origin-content": "content-box",
		}
		if v, ok2 := origins[rest]; ok2 {
			return map[string]any{"background-origin": v}
		}

		// Background position
		positions := map[string]string{
			"bottom": "bottom", "center": "center", "left": "left",
			"left-bottom": "left bottom", "left-top": "left top",
			"right": "right", "right-bottom": "right bottom", "right-top": "right top",
			"top": "top",
		}
		if v, ok2 := positions[rest]; ok2 {
			return map[string]any{"background-position": v}
		}

		// Background repeat
		repeats := map[string]string{
			"repeat": "repeat", "no-repeat": "no-repeat",
			"repeat-x": "repeat-x", "repeat-y": "repeat-y",
			"repeat-round": "round", "repeat-space": "space",
		}
		if v, ok2 := repeats[rest]; ok2 {
			return map[string]any{"background-repeat": v}
		}

		// Background size
		sizes := map[string]string{
			"auto": "auto", "cover": "cover", "contain": "contain",
		}
		if v, ok2 := sizes[rest]; ok2 {
			return map[string]any{"background-size": v}
		}

		// Gradient
		gradients := map[string]string{
			"none":       "none",
			"gradient-to-t":  "linear-gradient(to top, var(--tw-gradient-stops))",
			"gradient-to-tr": "linear-gradient(to top right, var(--tw-gradient-stops))",
			"gradient-to-r":  "linear-gradient(to right, var(--tw-gradient-stops))",
			"gradient-to-br": "linear-gradient(to bottom right, var(--tw-gradient-stops))",
			"gradient-to-b":  "linear-gradient(to bottom, var(--tw-gradient-stops))",
			"gradient-to-bl": "linear-gradient(to bottom left, var(--tw-gradient-stops))",
			"gradient-to-l":  "linear-gradient(to left, var(--tw-gradient-stops))",
			"gradient-to-tl": "linear-gradient(to top left, var(--tw-gradient-stops))",
		}
		if v, ok2 := gradients[rest]; ok2 {
			return map[string]any{"background-image": v}
		}

	}

	// Gradient from/via/to
	if rest, ok := stripPrefix(class, "from-"); ok {
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			result := map[string]any{
				"--tw-gradient-from":  color,
				"--tw-gradient-stops": fmt.Sprintf("var(--tw-gradient-from), var(--tw-gradient-to, %s)", transparentize(color)),
			}
			if opacity != "" {
				result["--tw-gradient-from"] = applyOpacity(color, opacity)
			}
			return result
		}
	}
	if rest, ok := stripPrefix(class, "via-"); ok {
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			result := map[string]any{
				"--tw-gradient-via":   color,
				"--tw-gradient-stops": "var(--tw-gradient-from), var(--tw-gradient-via), var(--tw-gradient-to, transparent)",
			}
			if opacity != "" {
				result["--tw-gradient-via"] = applyOpacity(color, opacity)
			}
			return result
		}
	}
	if rest, ok := stripPrefix(class, "to-"); ok {
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			result := map[string]any{"--tw-gradient-to": color}
			if opacity != "" {
				result["--tw-gradient-to"] = applyOpacity(color, opacity)
			}
			return result
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Border
// ---------------------------------------------------------------------------

func (p *Parser) parseBorder(class string) map[string]any {
	// Border radius
	if rest, ok := stripPrefix(class, "rounded-"); ok {
		radii := map[string]string{
			"none": "0px", "sm": "0.125rem", "": "0.25rem",
			"md": "0.375rem", "lg": "0.5rem", "xl": "0.75rem",
			"2xl": "1rem", "3xl": "1.5rem", "full": "9999px",
		}
		// Corner-specific
		cornerPrefixes := map[string][]string{
			"t-":  {"border-top-left-radius", "border-top-right-radius"},
			"r-":  {"border-top-right-radius", "border-bottom-right-radius"},
			"b-":  {"border-bottom-right-radius", "border-bottom-left-radius"},
			"l-":  {"border-top-left-radius", "border-bottom-left-radius"},
			"tl-": {"border-top-left-radius"},
			"tr-": {"border-top-right-radius"},
			"br-": {"border-bottom-right-radius"},
			"bl-": {"border-bottom-left-radius"},
			"ss-": {"border-start-start-radius"},
			"se-": {"border-start-end-radius"},
			"ee-": {"border-end-end-radius"},
			"es-": {"border-end-start-radius"},
		}
		for cp, props := range cornerPrefixes {
			if strings.HasPrefix(rest, cp) {
				size := rest[len(cp):]
				if size == "" {
					size = ""
				}
				val, ok2 := radii[size]
				if !ok2 {
					if arb := extractArbitraryValue(size); arb != "" {
						val = arb
					} else {
						continue
					}
				}
				result := make(map[string]any)
				for _, prop := range props {
					result[prop] = val
				}
				return result
			}
		}
		// Standard
		if v, ok2 := radii[rest]; ok2 {
			return map[string]any{"border-radius": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"border-radius": arb}
		}
	}
	// rounded (no suffix)
	if class == "rounded" {
		return map[string]any{"border-radius": "0.25rem"}
	}

	// Border width
	borderWidthProps := []struct {
		prefix string
		props  []string
	}{
		{"border-x-", []string{"border-left-width", "border-right-width"}},
		{"border-y-", []string{"border-top-width", "border-bottom-width"}},
		{"border-s-", []string{"border-inline-start-width"}},
		{"border-e-", []string{"border-inline-end-width"}},
		{"border-t-", []string{"border-top-width"}},
		{"border-r-", []string{"border-right-width"}},
		{"border-b-", []string{"border-bottom-width"}},
		{"border-l-", []string{"border-left-width"}},
		{"border-", []string{"border-width"}},
	}

	for _, bw := range borderWidthProps {
		if rest, ok := stripPrefix(class, bw.prefix); ok {
			widths := map[string]string{
				"0": "0px", "2": "2px", "4": "4px", "8": "8px",
			}
			if v, ok2 := widths[rest]; ok2 {
				result := make(map[string]any)
				for _, prop := range bw.props {
					result[prop] = v
				}
				return result
			}
			if arb := extractArbitraryValue(rest); arb != "" {
				result := make(map[string]any)
				for _, prop := range bw.props {
					result[prop] = arb
				}
				return result
			}
			// Color
			if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
				props2 := []string{"border-color"}
				if len(bw.props) > 1 {
					props2 = make([]string, len(bw.props))
					for i, p2 := range bw.props {
						props2[i] = strings.Replace(p2, "width", "color", 1)
					}
				} else if bw.props[0] != "border-width" {
					props2 = []string{strings.Replace(bw.props[0], "width", "color", 1)}
				}
				result := make(map[string]any)
				for _, prop := range props2 {
					if opacity != "" {
						result[prop] = applyOpacity(color, opacity)
					} else {
						result[prop] = color
					}
				}
				return result
			}
		}
	}

	// Plain "border"
	if class == "border" {
		return map[string]any{"border-width": "1px"}
	}

	// Plain side borders without width: border-t, border-r, border-b, border-l
	plainBorders := map[string][]string{
		"border-t": {"border-top-width"},
		"border-r": {"border-right-width"},
		"border-b": {"border-bottom-width"},
		"border-l": {"border-left-width"},
		"border-x": {"border-left-width", "border-right-width"},
		"border-y": {"border-top-width", "border-bottom-width"},
	}
	if props, ok := plainBorders[class]; ok {
		result := make(map[string]any)
		for _, prop := range props {
			result[prop] = "1px"
		}
		return result
	}

	// Border style
	borderStyles := map[string]string{
		"border-solid": "solid", "border-dashed": "dashed",
		"border-dotted": "dotted", "border-double": "double",
		"border-hidden": "hidden", "border-none": "none",
	}
	if v, ok := borderStyles[class]; ok {
		return map[string]any{"border-style": v}
	}

	// Divide (shorthand for children)
	if rest, ok := stripPrefix(class, "divide-x-"); ok {
		if rest == "reverse" {
			return map[string]any{"--tw-divide-x-reverse": "1"}
		}
		w := "1px"
		if widths := map[string]string{"0": "0px", "2": "2px", "4": "4px", "8": "8px"}; widths[rest] != "" {
			w = widths[rest]
		} else if arb := extractArbitraryValue(rest); arb != "" {
			w = arb
		}
		return map[string]any{
			"--tw-divide-x-reverse":   "0",
			"border-right-width": fmt.Sprintf("calc(%s * var(--tw-divide-x-reverse))", w),
			"border-left-width":  fmt.Sprintf("calc(%s * calc(1 - var(--tw-divide-x-reverse)))", w),
		}
	}
	if rest, ok := stripPrefix(class, "divide-y-"); ok {
		if rest == "reverse" {
			return map[string]any{"--tw-divide-y-reverse": "1"}
		}
		w := "1px"
		if widths := map[string]string{"0": "0px", "2": "2px", "4": "4px", "8": "8px"}; widths[rest] != "" {
			w = widths[rest]
		} else if arb := extractArbitraryValue(rest); arb != "" {
			w = arb
		}
		return map[string]any{
			"--tw-divide-y-reverse":    "0",
			"border-bottom-width": fmt.Sprintf("calc(%s * var(--tw-divide-y-reverse))", w),
			"border-top-width":    fmt.Sprintf("calc(%s * calc(1 - var(--tw-divide-y-reverse)))", w),
		}
	}
	if rest, ok := stripPrefix(class, "divide-"); ok {
		// Divide color
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			if opacity != "" {
				return map[string]any{"border-color": applyOpacity(color, opacity)}
			}
			return map[string]any{"border-color": color}
		}
		// Divide style
		styles := map[string]string{
			"solid": "solid", "dashed": "dashed", "dotted": "dotted",
			"double": "double", "none": "none",
		}
		if v, ok2 := styles[rest]; ok2 {
			return map[string]any{"border-style": v}
		}
	}

	// Outline (handled in parseOutline)

	return nil
}

// ---------------------------------------------------------------------------
// Effects (opacity, box-shadow, mix-blend, background-blend)
// ---------------------------------------------------------------------------

func (p *Parser) parseEffects(class string) map[string]any {
	// Opacity is handled in parseOpacity
	// Mix blend mode
	if rest, ok := stripPrefix(class, "mix-blend-"); ok {
		return map[string]any{"mix-blend-mode": rest}
	}

	// Background blend mode
	if rest, ok := stripPrefix(class, "bg-blend-"); ok {
		return map[string]any{"background-blend-mode": rest}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Opacity
// ---------------------------------------------------------------------------

func (p *Parser) parseOpacity(class string) map[string]any {
	if rest, ok := stripPrefix(class, "opacity-"); ok {
		val := resolveOpacity(rest)
		return map[string]any{"opacity": val}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Shadow
// ---------------------------------------------------------------------------

func (p *Parser) parseShadow(class string) map[string]any {
	if rest, ok := stripPrefix(class, "shadow-"); ok {
		shadows := map[string]string{
			"sm":  "0 1px 2px 0 rgb(0 0 0 / 0.05)",
			"":    "0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)",
			"md":  "0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)",
			"lg":  "0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)",
			"xl":  "0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1)",
			"2xl": "0 25px 50px -12px rgb(0 0 0 / 0.25)",
			"inner": "inset 0 2px 4px 0 rgb(0 0 0 / 0.05)",
			"none": "0 0 #0000",
		}
		if v, ok2 := shadows[rest]; ok2 {
			return map[string]any{"box-shadow": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"box-shadow": arb}
		}
		// Shadow color
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			if opacity != "" {
				return map[string]any{"--tw-shadow-color": applyOpacity(color, opacity)}
			}
			return map[string]any{"--tw-shadow-color": color}
		}
	}
	if class == "shadow" {
		return map[string]any{"box-shadow": "0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)"}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Ring
// ---------------------------------------------------------------------------

func (p *Parser) parseRing(class string) map[string]any {
	ringWidths := map[string]string{
		"ring":    "3px",
		"ring-0":  "0px",
		"ring-1":  "1px",
		"ring-2":  "2px",
		"ring-4":  "4px",
		"ring-8":  "8px",
		"ring-inset": "inset",
	}
	if v, ok := ringWidths[class]; ok {
		if v == "inset" {
			return map[string]any{"--tw-ring-inset": "inset"}
		}
		return map[string]any{
			"--tw-ring-shadow": fmt.Sprintf("var(--tw-ring-inset,) 0 0 0 calc(%s + var(--tw-ring-offset-width,0px)) var(--tw-ring-color,rgb(59 130 246 / 0.5))", v),
			"box-shadow":       "var(--tw-ring-offset-shadow,0 0 #0000), var(--tw-ring-shadow,0 0 #0000), var(--tw-shadow,0 0 #0000)",
		}
	}

	if rest, ok := stripPrefix(class, "ring-"); ok {
		// Ring color
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			c := color
			if opacity != "" {
				c = applyOpacity(color, opacity)
			}
			return map[string]any{"--tw-ring-color": c}
		}
		// Ring offset width
		if strings.HasPrefix(rest, "offset-") {
			inner := rest[len("offset-"):]
			offsets := map[string]string{"0": "0px", "1": "1px", "2": "2px", "4": "4px", "8": "8px"}
			if v, ok2 := offsets[inner]; ok2 {
				return map[string]any{"--tw-ring-offset-width": v}
			}
			if arb := extractArbitraryValue(inner); arb != "" {
				return map[string]any{"--tw-ring-offset-width": arb}
			}
			// ring-offset color
			if color, opacity, ok2 := withColorAndOpacity(inner); ok2 {
				c := color
				if opacity != "" {
					c = applyOpacity(color, opacity)
				}
				return map[string]any{"--tw-ring-offset-color": c}
			}
		}
		// Arbitrary ring width
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{
				"--tw-ring-shadow": fmt.Sprintf("var(--tw-ring-inset,) 0 0 0 calc(%s + var(--tw-ring-offset-width,0px)) var(--tw-ring-color,rgb(59 130 246 / 0.5))", arb),
				"box-shadow":       "var(--tw-ring-offset-shadow,0 0 #0000), var(--tw-ring-shadow,0 0 #0000), var(--tw-shadow,0 0 #0000)",
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Outline
// ---------------------------------------------------------------------------

func (p *Parser) parseOutline(class string) map[string]any {
	if class == "outline-none" {
		return map[string]any{"outline": "2px solid transparent", "outline-offset": "2px"}
	}
	if class == "outline" {
		return map[string]any{"outline-style": "solid"}
	}
	outlineStyles := map[string]string{
		"outline-dashed": "dashed", "outline-dotted": "dotted",
		"outline-double": "double",
	}
	if v, ok := outlineStyles[class]; ok {
		return map[string]any{"outline-style": v}
	}

	if rest, ok := stripPrefix(class, "outline-"); ok {
		widths := map[string]string{
			"0": "0px", "1": "1px", "2": "2px", "4": "4px", "8": "8px",
		}
		if v, ok2 := widths[rest]; ok2 {
			return map[string]any{"outline-width": v}
		}
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			c := color
			if opacity != "" {
				c = applyOpacity(color, opacity)
			}
			return map[string]any{"outline-color": c}
		}
		if strings.HasPrefix(rest, "offset-") {
			inner := rest[len("offset-"):]
			offsets := map[string]string{"0": "0px", "1": "1px", "2": "2px", "4": "4px", "8": "8px"}
			if v, ok2 := offsets[inner]; ok2 {
				return map[string]any{"outline-offset": v}
			}
			if arb := extractArbitraryValue(inner); arb != "" {
				return map[string]any{"outline-offset": arb}
			}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"outline-width": arb}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Filters
// ---------------------------------------------------------------------------

func (p *Parser) parseFilters(class string) map[string]any {
	// Blur
	if rest, ok := stripPrefix(class, "blur-"); ok {
		blurs := map[string]string{
			"none": "blur(0)", "sm": "blur(4px)", "": "blur(8px)",
			"md": "blur(12px)", "lg": "blur(16px)", "xl": "blur(24px)",
			"2xl": "blur(40px)", "3xl": "blur(64px)",
		}
		if v, ok2 := blurs[rest]; ok2 {
			return map[string]any{"filter": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"filter": fmt.Sprintf("blur(%s)", arb)}
		}
	}
	if class == "blur" {
		return map[string]any{"filter": "blur(8px)"}
	}

	// Brightness
	if rest, ok := stripPrefix(class, "brightness-"); ok {
		return map[string]any{"filter": fmt.Sprintf("brightness(%s%%)", rest)}
	}

	// Contrast
	if rest, ok := stripPrefix(class, "contrast-"); ok {
		return map[string]any{"filter": fmt.Sprintf("contrast(%s%%)", rest)}
	}

	// Drop shadow
	if rest, ok := stripPrefix(class, "drop-shadow-"); ok {
		dropShadows := map[string]string{
			"sm":  "drop-shadow(0 1px 1px rgb(0 0 0 / 0.05))",
			"":    "drop-shadow(0 1px 2px rgb(0 0 0 / 0.1)) drop-shadow(0 1px 1px rgb(0 0 0 / 0.06))",
			"md":  "drop-shadow(0 4px 3px rgb(0 0 0 / 0.07)) drop-shadow(0 2px 2px rgb(0 0 0 / 0.06))",
			"lg":  "drop-shadow(0 10px 8px rgb(0 0 0 / 0.04)) drop-shadow(0 4px 3px rgb(0 0 0 / 0.1))",
			"xl":  "drop-shadow(0 20px 13px rgb(0 0 0 / 0.03)) drop-shadow(0 8px 5px rgb(0 0 0 / 0.08))",
			"2xl": "drop-shadow(0 25px 25px rgb(0 0 0 / 0.15))",
			"none": "drop-shadow(0 0 #0000)",
		}
		if v, ok2 := dropShadows[rest]; ok2 {
			return map[string]any{"filter": v}
		}
	}

	// Grayscale
	if class == "grayscale" {
		return map[string]any{"filter": "grayscale(100%)"}
	}
	if class == "grayscale-0" {
		return map[string]any{"filter": "grayscale(0)"}
	}

	// Hue rotate
	if rest, ok := stripPrefix(class, "hue-rotate-"); ok {
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"filter": fmt.Sprintf("hue-rotate(%s)", arb)}
		}
		return map[string]any{"filter": fmt.Sprintf("hue-rotate(%sdeg)", rest)}
	}

	// Invert
	if class == "invert" {
		return map[string]any{"filter": "invert(100%)"}
	}
	if class == "invert-0" {
		return map[string]any{"filter": "invert(0)"}
	}

	// Saturate
	if rest, ok := stripPrefix(class, "saturate-"); ok {
		return map[string]any{"filter": fmt.Sprintf("saturate(%s%%)", rest)}
	}

	// Sepia
	if class == "sepia" {
		return map[string]any{"filter": "sepia(100%)"}
	}
	if class == "sepia-0" {
		return map[string]any{"filter": "sepia(0)"}
	}

	// Backdrop filters
	if rest, ok := stripPrefix(class, "backdrop-"); ok {
		if blurRest, ok2 := stripPrefix(rest, "blur-"); ok2 {
			blurs := map[string]string{
				"none": "blur(0)", "sm": "blur(4px)", "md": "blur(12px)",
				"lg": "blur(16px)", "xl": "blur(24px)", "2xl": "blur(40px)", "3xl": "blur(64px)",
			}
			if v, ok3 := blurs[blurRest]; ok3 {
				return map[string]any{"backdrop-filter": v}
			}
			return map[string]any{"backdrop-filter": fmt.Sprintf("blur(%s)", blurRest)}
		}
		if strings.HasPrefix(rest, "brightness-") {
			v := rest[len("brightness-"):]
			return map[string]any{"backdrop-filter": fmt.Sprintf("brightness(%s%%)", v)}
		}
		if strings.HasPrefix(rest, "contrast-") {
			v := rest[len("contrast-"):]
			return map[string]any{"backdrop-filter": fmt.Sprintf("contrast(%s%%)", v)}
		}
		if rest == "grayscale" {
			return map[string]any{"backdrop-filter": "grayscale(100%)"}
		}
		if rest == "grayscale-0" {
			return map[string]any{"backdrop-filter": "grayscale(0)"}
		}
		if strings.HasPrefix(rest, "hue-rotate-") {
			v := rest[len("hue-rotate-"):]
			return map[string]any{"backdrop-filter": fmt.Sprintf("hue-rotate(%sdeg)", v)}
		}
		if rest == "invert" {
			return map[string]any{"backdrop-filter": "invert(100%)"}
		}
		if rest == "invert-0" {
			return map[string]any{"backdrop-filter": "invert(0)"}
		}
		if rest == "opacity" {
			return map[string]any{"backdrop-filter": "opacity(100%)"}
		}
		if strings.HasPrefix(rest, "opacity-") {
			v := rest[len("opacity-"):]
			return map[string]any{"backdrop-filter": fmt.Sprintf("opacity(%s%%)", v)}
		}
		if strings.HasPrefix(rest, "saturate-") {
			v := rest[len("saturate-"):]
			return map[string]any{"backdrop-filter": fmt.Sprintf("saturate(%s%%)", v)}
		}
		if rest == "sepia" {
			return map[string]any{"backdrop-filter": "sepia(100%)"}
		}
		if rest == "sepia-0" {
			return map[string]any{"backdrop-filter": "sepia(0)"}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"backdrop-filter": arb}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Tables
// ---------------------------------------------------------------------------

func (p *Parser) parseTables(class string) map[string]any {
	tableBorders := map[string]string{
		"border-collapse": "collapse",
		"border-separate": "separate",
	}
	if v, ok := tableBorders[class]; ok {
		return map[string]any{"border-collapse": v}
	}

	tableLayouts := map[string]string{
		"table-auto":  "auto",
		"table-fixed": "fixed",
	}
	if v, ok := tableLayouts[class]; ok {
		return map[string]any{"table-layout": v}
	}

	captionSides := map[string]string{
		"caption-top":    "top",
		"caption-bottom": "bottom",
	}
	if v, ok := captionSides[class]; ok {
		return map[string]any{"caption-side": v}
	}

	if rest, ok := stripPrefix(class, "border-spacing-x-"); ok {
		if val, ok2 := resolveSpacing(rest); ok2 {
			return map[string]any{"--tw-border-spacing-x": val, "border-spacing": "var(--tw-border-spacing-x) var(--tw-border-spacing-y)"}
		}
	}
	if rest, ok := stripPrefix(class, "border-spacing-y-"); ok {
		if val, ok2 := resolveSpacing(rest); ok2 {
			return map[string]any{"--tw-border-spacing-y": val, "border-spacing": "var(--tw-border-spacing-x) var(--tw-border-spacing-y)"}
		}
	}
	if rest, ok := stripPrefix(class, "border-spacing-"); ok {
		if val, ok2 := resolveSpacing(rest); ok2 {
			return map[string]any{"--tw-border-spacing-x": val, "--tw-border-spacing-y": val, "border-spacing": "var(--tw-border-spacing-x) var(--tw-border-spacing-y)"}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Transitions & Animation
// ---------------------------------------------------------------------------

func (p *Parser) parseTransitions(class string) map[string]any {
	// Transition property
	if rest, ok := stripPrefix(class, "transition-"); ok {
		transitions := map[string]string{
			"none":    "none",
			"all":     "all 150ms cubic-bezier(0.4,0,0.2,1)",
			"colors":  "color, background-color, border-color, text-decoration-color, fill, stroke 150ms cubic-bezier(0.4,0,0.2,1)",
			"opacity": "opacity 150ms cubic-bezier(0.4,0,0.2,1)",
			"shadow":  "box-shadow 150ms cubic-bezier(0.4,0,0.2,1)",
			"transform": "transform 150ms cubic-bezier(0.4,0,0.2,1)",
		}
		if v, ok2 := transitions[rest]; ok2 {
			return map[string]any{"transition": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"transition-property": arb}
		}
	}
	if class == "transition" {
		return map[string]any{"transition": "color, background-color, border-color, text-decoration-color, fill, stroke, opacity, box-shadow, transform, filter, backdrop-filter 150ms cubic-bezier(0.4,0,0.2,1)"}
	}

	// Duration
	if rest, ok := stripPrefix(class, "duration-"); ok {
		durations := map[string]string{
			"0": "0s", "75": "75ms", "100": "100ms", "150": "150ms",
			"200": "200ms", "300": "300ms", "500": "500ms",
			"700": "700ms", "1000": "1000ms",
		}
		if v, ok2 := durations[rest]; ok2 {
			return map[string]any{"transition-duration": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"transition-duration": arb}
		}
	}

	// Timing function (ease)
	if rest, ok := stripPrefix(class, "ease-"); ok {
		easings := map[string]string{
			"linear":  "linear",
			"in":      "cubic-bezier(0.4, 0, 1, 1)",
			"out":     "cubic-bezier(0, 0, 0.2, 1)",
			"in-out":  "cubic-bezier(0.4, 0, 0.2, 1)",
		}
		if v, ok2 := easings[rest]; ok2 {
			return map[string]any{"transition-timing-function": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"transition-timing-function": arb}
		}
	}

	// Delay
	if rest, ok := stripPrefix(class, "delay-"); ok {
		delays := map[string]string{
			"0": "0s", "75": "75ms", "100": "100ms", "150": "150ms",
			"200": "200ms", "300": "300ms", "500": "500ms",
			"700": "700ms", "1000": "1000ms",
		}
		if v, ok2 := delays[rest]; ok2 {
			return map[string]any{"transition-delay": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"transition-delay": arb}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Animation
// ---------------------------------------------------------------------------

func (p *Parser) parseAnimation(class string) map[string]any {
	if rest, ok := stripPrefix(class, "animate-"); ok {
		animations := map[string]map[string]any{
			"none": {"animation": "none"},
			"spin": {"animation": "spin 1s linear infinite"},
			"ping": {"animation": "ping 1s cubic-bezier(0, 0, 0.2, 1) infinite"},
			"pulse": {"animation": "pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite"},
			"bounce": {"animation": "bounce 1s infinite"},
		}
		if v, ok2 := animations[rest]; ok2 {
			return v
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"animation": arb}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Transforms
// ---------------------------------------------------------------------------

func (p *Parser) parseTransforms(class string) map[string]any {
	if class == "transform" {
		return map[string]any{"transform": "translateX(var(--tw-translate-x,0)) translateY(var(--tw-translate-y,0)) rotate(var(--tw-rotate,0)) skewX(var(--tw-skew-x,0)) skewY(var(--tw-skew-y,0)) scaleX(var(--tw-scale-x,1)) scaleY(var(--tw-scale-y,1))"}
	}
	if class == "transform-none" {
		return map[string]any{"transform": "none"}
	}
	if class == "transform-gpu" {
		return map[string]any{"transform": "translateZ(0)"}
	}

	// Scale
	if rest, ok := stripPrefix(class, "scale-x-"); ok {
		val := scaleValue(rest)
		return map[string]any{"--tw-scale-x": val, "transform": buildTransform()}
	}
	if rest, ok := stripPrefix(class, "scale-y-"); ok {
		val := scaleValue(rest)
		return map[string]any{"--tw-scale-y": val, "transform": buildTransform()}
	}
	if rest, ok := stripPrefix(class, "scale-"); ok {
		val := scaleValue(rest)
		return map[string]any{"--tw-scale-x": val, "--tw-scale-y": val, "transform": buildTransform()}
	}

	// Rotate
	if rest, ok := stripPrefix(class, "rotate-"); ok {
		neg := strings.HasPrefix(rest, "-")
		if neg {
			rest = rest[1:]
		}
		val := rest
		if arb := extractArbitraryValue(rest); arb != "" {
			val = arb
		} else {
			val = rest + "deg"
		}
		if neg {
			val = "-" + val
		}
		return map[string]any{"--tw-rotate": val, "transform": buildTransform()}
	}
	if rest, ok := stripPrefix(class, "-rotate-"); ok {
		val := rest + "deg"
		return map[string]any{"--tw-rotate": "-" + val, "transform": buildTransform()}
	}

	// Translate
	if rest, ok := stripPrefix(class, "translate-x-"); ok {
		neg := false
		if strings.HasPrefix(rest, "-") {
			neg = true
			rest = rest[1:]
		}
		val, ok2 := resolveSpacing(rest)
		if !ok2 {
			if arb := extractArbitraryValue(rest); arb != "" {
				val = arb
			} else {
				val = rest
			}
		}
		if neg {
			val = negate(val)
		}
		return map[string]any{"--tw-translate-x": val, "transform": buildTransform()}
	}
	if rest, ok := stripPrefix(class, "translate-y-"); ok {
		neg := false
		if strings.HasPrefix(rest, "-") {
			neg = true
			rest = rest[1:]
		}
		val, ok2 := resolveSpacing(rest)
		if !ok2 {
			if arb := extractArbitraryValue(rest); arb != "" {
				val = arb
			} else {
				val = rest
			}
		}
		if neg {
			val = negate(val)
		}
		return map[string]any{"--tw-translate-y": val, "transform": buildTransform()}
	}
	if rest, ok := stripPrefix(class, "-translate-x-"); ok {
		val, ok2 := resolveSpacing(rest)
		if !ok2 {
			val = rest
		}
		return map[string]any{"--tw-translate-x": negate(val), "transform": buildTransform()}
	}
	if rest, ok := stripPrefix(class, "-translate-y-"); ok {
		val, ok2 := resolveSpacing(rest)
		if !ok2 {
			val = rest
		}
		return map[string]any{"--tw-translate-y": negate(val), "transform": buildTransform()}
	}

	// Skew
	if rest, ok := stripPrefix(class, "skew-x-"); ok {
		val := rest + "deg"
		if arb := extractArbitraryValue(rest); arb != "" {
			val = arb
		}
		return map[string]any{"--tw-skew-x": val, "transform": buildTransform()}
	}
	if rest, ok := stripPrefix(class, "skew-y-"); ok {
		val := rest + "deg"
		if arb := extractArbitraryValue(rest); arb != "" {
			val = arb
		}
		return map[string]any{"--tw-skew-y": val, "transform": buildTransform()}
	}

	// Transform origin
	if rest, ok := stripPrefix(class, "origin-"); ok {
		origins := map[string]string{
			"center":        "center",
			"top":           "top",
			"top-right":     "top right",
			"right":         "right",
			"bottom-right":  "bottom right",
			"bottom":        "bottom",
			"bottom-left":   "bottom left",
			"left":          "left",
			"top-left":      "top left",
		}
		if v, ok2 := origins[rest]; ok2 {
			return map[string]any{"transform-origin": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"transform-origin": arb}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Interactivity
// ---------------------------------------------------------------------------

func (p *Parser) parseInteractivity(class string) map[string]any {
	// Scroll snap
	snapAligns := map[string]string{
		"snap-start":  "start", "snap-end": "end",
		"snap-center": "center", "snap-align-none": "none",
	}
	if v, ok := snapAligns[class]; ok {
		return map[string]any{"scroll-snap-align": v}
	}

	snapTypes := map[string]string{
		"snap-none":      "none",
		"snap-x":         "x var(--tw-scroll-snap-strictness)",
		"snap-y":         "y var(--tw-scroll-snap-strictness)",
		"snap-both":      "both var(--tw-scroll-snap-strictness)",
	}
	if v, ok := snapTypes[class]; ok {
		return map[string]any{"scroll-snap-type": v}
	}
	if class == "snap-mandatory" {
		return map[string]any{"--tw-scroll-snap-strictness": "mandatory"}
	}
	if class == "snap-proximity" {
		return map[string]any{"--tw-scroll-snap-strictness": "proximity"}
	}
	if class == "snap-normal" {
		return map[string]any{"scroll-snap-stop": "normal"}
	}
	if class == "snap-always" {
		return map[string]any{"scroll-snap-stop": "always"}
	}

	// Scroll margin/padding
	for _, dir := range []struct{ prefix, prop string }{
		{"scroll-m-", "scroll-margin"},
		{"scroll-mx-", "scroll-margin"},
		{"scroll-my-", "scroll-margin"},
		{"scroll-mt-", "scroll-margin-top"},
		{"scroll-mr-", "scroll-margin-right"},
		{"scroll-mb-", "scroll-margin-bottom"},
		{"scroll-ml-", "scroll-margin-left"},
		{"scroll-p-", "scroll-padding"},
		{"scroll-px-", "scroll-padding"},
		{"scroll-py-", "scroll-padding"},
		{"scroll-pt-", "scroll-padding-top"},
		{"scroll-pr-", "scroll-padding-right"},
		{"scroll-pb-", "scroll-padding-bottom"},
		{"scroll-pl-", "scroll-padding-left"},
	} {
		if rest, ok := stripPrefix(class, dir.prefix); ok {
			if val, ok2 := resolveSpacing(rest); ok2 {
				return map[string]any{dir.prop: val}
			}
		}
	}

	// Touch action
	if rest, ok := stripPrefix(class, "touch-"); ok {
		touches := map[string]string{
			"auto": "auto", "none": "none", "pan-x": "pan-x",
			"pan-left": "pan-left", "pan-right": "pan-right",
			"pan-y": "pan-y", "pan-up": "pan-up", "pan-down": "pan-down",
			"pinch-zoom": "pinch-zoom", "manipulation": "manipulation",
		}
		if v, ok2 := touches[rest]; ok2 {
			return map[string]any{"touch-action": v}
		}
	}

	// User select
	if rest, ok := stripPrefix(class, "select-"); ok {
		selects := map[string]string{
			"none": "none", "text": "text", "all": "all", "auto": "auto",
		}
		if v, ok2 := selects[rest]; ok2 {
			return map[string]any{"user-select": v}
		}
	}

	// Will change
	if rest, ok := stripPrefix(class, "will-change-"); ok {
		willChanges := map[string]string{
			"auto": "auto", "scroll": "scroll-position",
			"contents": "contents", "transform": "transform",
		}
		if v, ok2 := willChanges[rest]; ok2 {
			return map[string]any{"will-change": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"will-change": arb}
		}
	}

	// Resize
	if rest, ok := stripPrefix(class, "resize-"); ok {
		resizes := map[string]string{"none": "none", "y": "vertical", "x": "horizontal"}
		if v, ok2 := resizes[rest]; ok2 {
			return map[string]any{"resize": v}
		}
	}
	if class == "resize" {
		return map[string]any{"resize": "both"}
	}

	// Pointer events
	if class == "pointer-events-none" {
		return map[string]any{"pointer-events": "none"}
	}
	if class == "pointer-events-auto" {
		return map[string]any{"pointer-events": "auto"}
	}

	return nil
}

// ---------------------------------------------------------------------------
// SVG
// ---------------------------------------------------------------------------

func (p *Parser) parseSVG(class string) map[string]any {
	// Fill
	if rest, ok := stripPrefix(class, "fill-"); ok {
		if rest == "none" {
			return map[string]any{"fill": "none"}
		}
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			if opacity != "" {
				return map[string]any{"fill": applyOpacity(color, opacity)}
			}
			return map[string]any{"fill": color}
		}
	}

	// Stroke
	if rest, ok := stripPrefix(class, "stroke-"); ok {
		// Stroke width
		widths := map[string]string{"0": "0", "1": "1", "2": "2"}
		if v, ok2 := widths[rest]; ok2 {
			return map[string]any{"stroke-width": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"stroke-width": arb}
		}
		// Stroke color
		if rest == "none" {
			return map[string]any{"stroke": "none"}
		}
		if color, opacity, ok2 := withColorAndOpacity(rest); ok2 {
			if opacity != "" {
				return map[string]any{"stroke": applyOpacity(color, opacity)}
			}
			return map[string]any{"stroke": color}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Accessibility
// ---------------------------------------------------------------------------

func (p *Parser) parseAccessibility(class string) map[string]any {
	if class == "sr-only" {
		return map[string]any{
			"position": "absolute",
			"width":    "1px",
			"height":   "1px",
			"padding":  "0",
			"margin":   "-1px",
			"overflow": "hidden",
			"clip":     "rect(0, 0, 0, 0)",
			"white-space": "nowrap",
			"border-width": "0",
		}
	}
	if class == "not-sr-only" {
		return map[string]any{
			"position": "static",
			"width":    "auto",
			"height":   "auto",
			"padding":  "0",
			"margin":   "0",
			"overflow": "visible",
			"clip":     "auto",
			"white-space": "normal",
		}
	}
	if class == "forced-color-adjust-auto" {
		return map[string]any{"forced-color-adjust": "auto"}
	}
	if class == "forced-color-adjust-none" {
		return map[string]any{"forced-color-adjust": "none"}
	}
	return nil
}

// ---------------------------------------------------------------------------
// List style
// ---------------------------------------------------------------------------

func (p *Parser) parseListStyle(class string) map[string]any {
	listStyles := map[string]string{
		"list-none":     "none",
		"list-disc":     "disc",
		"list-decimal":  "decimal",
	}
	if v, ok := listStyles[class]; ok {
		return map[string]any{"list-style-type": v}
	}
	if rest, ok := stripPrefix(class, "list-"); ok {
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"list-style-type": arb}
		}
	}

	listPositions := map[string]string{
		"list-inside":  "inside",
		"list-outside": "outside",
	}
	if v, ok := listPositions[class]; ok {
		return map[string]any{"list-style-position": v}
	}

	// List image
	if rest, ok := stripPrefix(class, "list-image-"); ok {
		if rest == "none" {
			return map[string]any{"list-style-image": "none"}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"list-style-image": fmt.Sprintf("url(%s)", arb)}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Appearance
// ---------------------------------------------------------------------------

func (p *Parser) parseAppearance(class string) map[string]any {
	if class == "appearance-none" {
		return map[string]any{"appearance": "none"}
	}
	if class == "appearance-auto" {
		return map[string]any{"appearance": "auto"}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Cursor
// ---------------------------------------------------------------------------

func (p *Parser) parseCursor(class string) map[string]any {
	if rest, ok := stripPrefix(class, "cursor-"); ok {
		cursors := map[string]string{
			"auto": "auto", "default": "default", "pointer": "pointer",
			"wait": "wait", "text": "text", "move": "move",
			"help": "help", "not-allowed": "not-allowed", "none": "none",
			"context-menu": "context-menu", "progress": "progress",
			"cell": "cell", "crosshair": "crosshair", "vertical-text": "vertical-text",
			"alias": "alias", "copy": "copy", "no-drop": "no-drop",
			"grab": "grab", "grabbing": "grabbing", "all-scroll": "all-scroll",
			"col-resize": "col-resize", "row-resize": "row-resize",
			"n-resize": "n-resize", "e-resize": "e-resize",
			"s-resize": "s-resize", "w-resize": "w-resize",
			"ne-resize": "ne-resize", "nw-resize": "nw-resize",
			"se-resize": "se-resize", "sw-resize": "sw-resize",
			"ew-resize": "ew-resize", "ns-resize": "ns-resize",
			"nesw-resize": "nesw-resize", "nwse-resize": "nwse-resize",
			"zoom-in": "zoom-in", "zoom-out": "zoom-out",
		}
		if v, ok2 := cursors[rest]; ok2 {
			return map[string]any{"cursor": v}
		}
		if arb := extractArbitraryValue(rest); arb != "" {
			return map[string]any{"cursor": arb}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Arbitrary property handler
// ---------------------------------------------------------------------------

func (p *Parser) parseArbitrary(class string) map[string]any {
	// [property:value] format at top level
	if strings.HasPrefix(class, "[") && strings.HasSuffix(class, "]") {
		inner := class[1 : len(class)-1]
		if idx := strings.Index(inner, ":"); idx != -1 {
			prop := strings.ReplaceAll(inner[:idx], "_", " ")
			val := strings.ReplaceAll(inner[idx+1:], "_", " ")
			return map[string]any{prop: val}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func stripPrefix(s, prefix string) (string, bool) {
	if strings.HasPrefix(s, prefix) {
		return s[len(prefix):], true
	}
	return "", false
}

func extractArbitraryValue(s string) string {
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s[1 : len(s)-1]
	}
	return ""
}

func extractArbitrary(class, prefix string) string {
	rest := class[len(prefix):]
	return extractArbitraryValue(rest)
}

func negate(val string) string {
	if val == "0" || val == "0px" || val == "auto" {
		return val
	}
	if strings.HasPrefix(val, "-") {
		return val[1:]
	}
	return "-" + val
}

func parseFraction(s string) (string, string, bool) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	num := strings.TrimSpace(parts[0])
	den := strings.TrimSpace(parts[1])
	if num == "" || den == "" {
		return "", "", false
	}
	if _, err := strconv.ParseFloat(num, 64); err != nil {
		return "", "", false
	}
	if _, err := strconv.ParseFloat(den, 64); err != nil {
		return "", "", false
	}
	return num, den, true
}

func applyOpacity(color, opacity string) string {
	// Return the color as-is for now; real implementations would convert hex to rgb
	// and apply opacity. For now we use CSS color-mix or the raw value.
	return fmt.Sprintf("color-mix(in srgb, %s %s%%, transparent)", color, opacityToPercent(opacity))
}

func opacityToPercent(opacity string) string {
	f, err := strconv.ParseFloat(opacity, 64)
	if err != nil {
		return opacity
	}
	return strconv.FormatFloat(f*100, 'f', -1, 64)
}

func transparentize(color string) string {
	return fmt.Sprintf("color-mix(in srgb, %s 0%%, transparent)", color)
}

func scaleValue(s string) string {
	if arb := extractArbitraryValue(s); arb != "" {
		return arb
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return strconv.FormatFloat(f/100, 'f', -1, 64)
}

func buildTransform() string {
	return "translateX(var(--tw-translate-x,0)) translateY(var(--tw-translate-y,0)) rotate(var(--tw-rotate,0)) skewX(var(--tw-skew-x,0)) skewY(var(--tw-skew-y,0)) scaleX(var(--tw-scale-x,1)) scaleY(var(--tw-scale-y,1))"
}
