package svg

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Matrix represents a 2D affine transform [a b c d e f].
// The full matrix is:
//
//	| a c e |
//	| b d f |
//	| 0 0 1 |
type Matrix [6]float64

// Identity returns the identity matrix.
func Identity() Matrix {
	return Matrix{1, 0, 0, 1, 0, 0}
}

// Multiply returns the product m * n.
func (m Matrix) Multiply(n Matrix) Matrix {
	return Matrix{
		m[0]*n[0] + m[2]*n[1],
		m[1]*n[0] + m[3]*n[1],
		m[0]*n[2] + m[2]*n[3],
		m[1]*n[2] + m[3]*n[3],
		m[0]*n[4] + m[2]*n[5] + m[4],
		m[1]*n[4] + m[3]*n[5] + m[5],
	}
}

// ToPDFOperator returns the PDF cm operator string.
func (m Matrix) ToPDFOperator() string {
	return fmt.Sprintf("%.6g %.6g %.6g %.6g %.6g %.6g cm", m[0], m[1], m[2], m[3], m[4], m[5])
}

// TransformPoint applies the matrix to a point.
func (m Matrix) TransformPoint(x, y float64) (float64, float64) {
	return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
}

// parseTransform parses an SVG transform attribute string.
func parseTransform(s string) Matrix {
	s = strings.TrimSpace(s)
	if s == "" {
		return Identity()
	}

	result := Identity()
	for len(s) > 0 {
		s = strings.TrimSpace(s)
		if s == "" {
			break
		}
		idx := strings.Index(s, "(")
		if idx < 0 {
			break
		}
		name := strings.TrimSpace(s[:idx])
		end := strings.Index(s, ")")
		if end < 0 {
			break
		}
		argsStr := s[idx+1 : end]
		s = s[end+1:]

		args := parseTransformArgs(argsStr)
		var m Matrix
		switch name {
		case "matrix":
			if len(args) >= 6 {
				m = Matrix{args[0], args[1], args[2], args[3], args[4], args[5]}
			} else {
				continue
			}
		case "translate":
			tx := 0.0
			ty := 0.0
			if len(args) >= 1 {
				tx = args[0]
			}
			if len(args) >= 2 {
				ty = args[1]
			}
			m = Matrix{1, 0, 0, 1, tx, ty}
		case "scale":
			sx := 1.0
			sy := 1.0
			if len(args) >= 1 {
				sx = args[0]
				sy = sx
			}
			if len(args) >= 2 {
				sy = args[1]
			}
			m = Matrix{sx, 0, 0, sy, 0, 0}
		case "rotate":
			if len(args) < 1 {
				continue
			}
			angle := args[0] * math.Pi / 180
			cos := math.Cos(angle)
			sin := math.Sin(angle)
			if len(args) >= 3 {
				cx, cy := args[1], args[2]
				m = Matrix{1, 0, 0, 1, cx, cy}
				m = m.Multiply(Matrix{cos, sin, -sin, cos, 0, 0})
				m = m.Multiply(Matrix{1, 0, 0, 1, -cx, -cy})
			} else {
				m = Matrix{cos, sin, -sin, cos, 0, 0}
			}
		case "skewX":
			if len(args) < 1 {
				continue
			}
			angle := args[0] * math.Pi / 180
			m = Matrix{1, 0, math.Tan(angle), 1, 0, 0}
		case "skewY":
			if len(args) < 1 {
				continue
			}
			angle := args[0] * math.Pi / 180
			m = Matrix{1, math.Tan(angle), 0, 1, 0, 0}
		default:
			continue
		}
		result = result.Multiply(m)
	}
	return result
}

func parseTransformArgs(s string) []float64 {
	s = strings.ReplaceAll(s, ",", " ")
	fields := strings.Fields(s)
	args := make([]float64, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.ParseFloat(f, 64)
		if err == nil {
			args = append(args, v)
		}
	}
	return args
}
