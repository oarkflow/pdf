package html

import (
	"fmt"
	"strings"
)

// CounterState manages CSS counters.
type CounterState struct {
	counters map[string][]int
}

// NewCounterState creates a new counter state.
func NewCounterState() *CounterState {
	return &CounterState{
		counters: make(map[string][]int),
	}
}

// Reset resets a counter to the given value.
func (cs *CounterState) Reset(name string, value int) {
	cs.counters[name] = []int{value}
}

// Increment increments a counter by the given value.
func (cs *CounterState) Increment(name string, value int) {
	stack := cs.counters[name]
	if len(stack) == 0 {
		cs.counters[name] = []int{value}
		return
	}
	stack[len(stack)-1] += value
}

// Get returns the current value of a counter.
func (cs *CounterState) Get(name string) int {
	stack := cs.counters[name]
	if len(stack) == 0 {
		return 0
	}
	return stack[len(stack)-1]
}

// GetString returns the formatted counter value.
func (cs *CounterState) GetString(name string, style string) string {
	val := cs.Get(name)
	switch strings.ToLower(style) {
	case "decimal", "":
		return fmt.Sprintf("%d", val)
	case "lower-alpha", "lower-latin":
		if val >= 1 && val <= 26 {
			return string(rune('a' + val - 1))
		}
		return fmt.Sprintf("%d", val)
	case "upper-alpha", "upper-latin":
		if val >= 1 && val <= 26 {
			return string(rune('A' + val - 1))
		}
		return fmt.Sprintf("%d", val)
	case "lower-roman":
		return toRoman(val, false)
	case "upper-roman":
		return toRoman(val, true)
	case "disc":
		return "\u2022"
	case "circle":
		return "\u25CB"
	case "square":
		return "\u25A0"
	case "none":
		return ""
	default:
		return fmt.Sprintf("%d", val)
	}
}

// ResolveContent resolves counter() and counters() functions in a content string.
func (cs *CounterState) ResolveContent(content string) string {
	result := content
	// Resolve counter(name) and counter(name, style)
	for strings.Contains(result, "counter(") {
		idx := strings.Index(result, "counter(")
		end := findClosingParen(result, idx+7)
		if end < 0 {
			break
		}
		inner := result[idx+8 : end]
		parts := strings.SplitN(inner, ",", 2)
		name := strings.TrimSpace(parts[0])
		style := "decimal"
		if len(parts) > 1 {
			style = strings.TrimSpace(parts[1])
		}
		resolved := cs.GetString(name, style)
		result = result[:idx] + resolved + result[end+1:]
	}

	// Resolve counters(name, separator) and counters(name, separator, style)
	for strings.Contains(result, "counters(") {
		idx := strings.Index(result, "counters(")
		end := findClosingParen(result, idx+8)
		if end < 0 {
			break
		}
		inner := result[idx+9 : end]
		parts := strings.SplitN(inner, ",", 3)
		name := strings.TrimSpace(parts[0])
		separator := "."
		if len(parts) > 1 {
			separator = strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		}
		style := "decimal"
		if len(parts) > 2 {
			style = strings.TrimSpace(parts[2])
		}

		stack := cs.counters[name]
		var values []string
		for _, v := range stack {
			cs.counters[name] = []int{v}
			values = append(values, cs.GetString(name, style))
		}
		cs.counters[name] = stack
		resolved := strings.Join(values, separator)
		result = result[:idx] + resolved + result[end+1:]
	}

	// Strip quotes from string values
	result = strings.Trim(result, "\"'")
	return result
}

func toRoman(n int, upper bool) string {
	if n <= 0 || n > 3999 {
		return fmt.Sprintf("%d", n)
	}
	values := []struct {
		val int
		sym string
	}{
		{1000, "m"}, {900, "cm"}, {500, "d"}, {400, "cd"},
		{100, "c"}, {90, "xc"}, {50, "l"}, {40, "xl"},
		{10, "x"}, {9, "ix"}, {5, "v"}, {4, "iv"}, {1, "i"},
	}
	var sb strings.Builder
	for _, v := range values {
		for n >= v.val {
			sb.WriteString(v.sym)
			n -= v.val
		}
	}
	result := sb.String()
	if upper {
		return strings.ToUpper(result)
	}
	return result
}
