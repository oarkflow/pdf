package html

import (
	"strings"
)

// PageRule holds parsed @page rule information.
type PageRule struct {
	Size    [2]float64
	Margins [4]float64
	Marks   string
}

// parsePageRules extracts page configuration from a stylesheet's @page rules.
func parsePageRules(stylesheet *Stylesheet) *PageRule {
	if stylesheet == nil || len(stylesheet.Pages) == 0 {
		return nil
	}

	rule := &PageRule{}
	for _, page := range stylesheet.Pages {
		for name, val := range page.Properties {
			v := strings.TrimSpace(val.Value)
			switch name {
			case "size":
				rule.Size = parsePageSize(v)
			case "margin":
				parts := splitCSSValues(v)
				sides := expandFourValues(parts)
				rule.Margins[0] = parseLengthValue(sides[0], 12, 12)
				rule.Margins[1] = parseLengthValue(sides[1], 12, 12)
				rule.Margins[2] = parseLengthValue(sides[2], 12, 12)
				rule.Margins[3] = parseLengthValue(sides[3], 12, 12)
			case "margin-top":
				rule.Margins[0] = parseLengthValue(v, 12, 12)
			case "margin-right":
				rule.Margins[1] = parseLengthValue(v, 12, 12)
			case "margin-bottom":
				rule.Margins[2] = parseLengthValue(v, 12, 12)
			case "margin-left":
				rule.Margins[3] = parseLengthValue(v, 12, 12)
			case "marks":
				rule.Marks = v
			}
		}
	}

	return rule
}

func parsePageSize(s string) [2]float64 {
	s = strings.TrimSpace(strings.ToLower(s))

	// Named sizes
	switch s {
	case "a3":
		return [2]float64{841.89, 1190.55}
	case "a4":
		return [2]float64{595.28, 841.89}
	case "a5":
		return [2]float64{419.53, 595.28}
	case "b4":
		return [2]float64{708.66, 1000.63}
	case "b5":
		return [2]float64{498.90, 708.66}
	case "letter":
		return [2]float64{612, 792}
	case "legal":
		return [2]float64{612, 1008}
	case "ledger":
		return [2]float64{1224, 792}
	}

	// Check for landscape/portrait suffix
	landscape := false
	if strings.HasSuffix(s, " landscape") {
		landscape = true
		s = strings.TrimSuffix(s, " landscape")
	} else if strings.HasSuffix(s, " portrait") {
		s = strings.TrimSuffix(s, " portrait")
	}

	// Try named size after removing orientation
	size := [2]float64{}
	switch s {
	case "a4":
		size = [2]float64{595.28, 841.89}
	case "letter":
		size = [2]float64{612, 792}
	default:
		// Two values: width height
		parts := splitCSSValues(s)
		if len(parts) == 2 {
			size[0] = parseLengthValue(parts[0], 12, 12)
			size[1] = parseLengthValue(parts[1], 12, 12)
		} else if len(parts) == 1 {
			v := parseLengthValue(parts[0], 12, 12)
			size = [2]float64{v, v}
		}
	}

	if landscape && size[0] < size[1] {
		size[0], size[1] = size[1], size[0]
	}

	return size
}
