package svg

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

type pathCommand struct {
	cmd  byte
	args []float64
}

// parsePath parses an SVG path data string into a sequence of path commands.
func parsePath(d string) []pathCommand {
	var cmds []pathCommand
	d = strings.TrimSpace(d)
	i := 0
	n := len(d)

	for i < n {
		// Skip whitespace and commas
		for i < n && (d[i] == ' ' || d[i] == ',' || d[i] == '\t' || d[i] == '\n' || d[i] == '\r') {
			i++
		}
		if i >= n {
			break
		}

		ch := d[i]
		if isPathCommand(ch) {
			cmd := ch
			i++
			argCount := pathArgCount(cmd)
			if argCount == 0 {
				cmds = append(cmds, pathCommand{cmd: cmd})
				continue
			}
			// Parse repeated argument groups
			for {
				// Skip whitespace and commas
				for i < n && (d[i] == ' ' || d[i] == ',' || d[i] == '\t' || d[i] == '\n' || d[i] == '\r') {
					i++
				}
				if i >= n {
					break
				}
				if isPathCommand(d[i]) {
					break
				}
				args := make([]float64, 0, argCount)
				for j := 0; j < argCount && i < n; j++ {
					for i < n && (d[i] == ' ' || d[i] == ',' || d[i] == '\t' || d[i] == '\n' || d[i] == '\r') {
						i++
					}
					if i >= n {
						break
					}
					v, adv := parsePathNumber(d[i:])
					if adv == 0 {
						break
					}
					args = append(args, v)
					i += adv
				}
				if len(args) == argCount {
					cmds = append(cmds, pathCommand{cmd: cmd, args: args})
					// For M, subsequent groups become L; for m, subsequent become l
					if cmd == 'M' {
						cmd = 'L'
					} else if cmd == 'm' {
						cmd = 'l'
					}
				} else {
					break
				}
			}
		} else {
			i++ // skip unknown
		}
	}
	return cmds
}

func isPathCommand(ch byte) bool {
	switch ch {
	case 'M', 'm', 'L', 'l', 'H', 'h', 'V', 'v',
		'C', 'c', 'S', 's', 'Q', 'q', 'T', 't',
		'A', 'a', 'Z', 'z':
		return true
	}
	return false
}

func pathArgCount(cmd byte) int {
	switch cmd | 0x20 { // lowercase
	case 'z':
		return 0
	case 'h', 'v':
		return 1
	case 'm', 'l', 't':
		return 2
	case 's', 'q':
		return 4
	case 'c':
		return 6
	case 'a':
		return 7
	}
	return 0
}

func parsePathNumber(s string) (float64, int) {
	i := 0
	n := len(s)
	if i >= n {
		return 0, 0
	}

	// Skip leading whitespace/commas
	for i < n && (s[i] == ' ' || s[i] == ',' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	start := i
	if i >= n {
		return 0, 0
	}
	if !unicode.IsDigit(rune(s[i])) && s[i] != '-' && s[i] != '+' && s[i] != '.' {
		return 0, 0
	}

	if s[i] == '-' || s[i] == '+' {
		i++
	}
	dotSeen := false
	for i < n && (unicode.IsDigit(rune(s[i])) || (s[i] == '.' && !dotSeen)) {
		if s[i] == '.' {
			dotSeen = true
		}
		i++
	}
	// Exponent
	if i < n && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < n && (s[i] == '-' || s[i] == '+') {
			i++
		}
		for i < n && unicode.IsDigit(rune(s[i])) {
			i++
		}
	}
	if i == start {
		return 0, 0
	}
	numStr := s[start:i]
	var v float64
	// Manual parse for reliability
	v = 0
	neg := false
	j := 0
	if j < len(numStr) && numStr[j] == '-' {
		neg = true
		j++
	} else if j < len(numStr) && numStr[j] == '+' {
		j++
	}
	for j < len(numStr) && numStr[j] >= '0' && numStr[j] <= '9' {
		v = v*10 + float64(numStr[j]-'0')
		j++
	}
	if j < len(numStr) && numStr[j] == '.' {
		j++
		frac := 0.1
		for j < len(numStr) && numStr[j] >= '0' && numStr[j] <= '9' {
			v += float64(numStr[j]-'0') * frac
			frac /= 10
			j++
		}
	}
	if j < len(numStr) && (numStr[j] == 'e' || numStr[j] == 'E') {
		j++
		expNeg := false
		if j < len(numStr) && numStr[j] == '-' {
			expNeg = true
			j++
		} else if j < len(numStr) && numStr[j] == '+' {
			j++
		}
		exp := 0.0
		for j < len(numStr) && numStr[j] >= '0' && numStr[j] <= '9' {
			exp = exp*10 + float64(numStr[j]-'0')
			j++
		}
		if expNeg {
			v /= math.Pow(10, exp)
		} else {
			v *= math.Pow(10, exp)
		}
	}
	if neg {
		v = -v
	}
	return v, i
}

// renderPath converts parsed path commands into PDF content stream operators.
func renderPath(cmds []pathCommand, buf *[]byte) {
	var cx, cy float64       // current point
	var sx, sy float64       // subpath start
	var lcx, lcy float64     // last control point (for S/T)
	var lastCmd byte

	for _, pc := range cmds {
		switch pc.cmd {
		case 'M':
			cx, cy = pc.args[0], pc.args[1]
			sx, sy = cx, cy
			appendOp(buf, cx, cy, "m")
		case 'm':
			cx += pc.args[0]
			cy += pc.args[1]
			sx, sy = cx, cy
			appendOp(buf, cx, cy, "m")
		case 'L':
			cx, cy = pc.args[0], pc.args[1]
			appendOp(buf, cx, cy, "l")
		case 'l':
			cx += pc.args[0]
			cy += pc.args[1]
			appendOp(buf, cx, cy, "l")
		case 'H':
			cx = pc.args[0]
			appendOp(buf, cx, cy, "l")
		case 'h':
			cx += pc.args[0]
			appendOp(buf, cx, cy, "l")
		case 'V':
			cy = pc.args[0]
			appendOp(buf, cx, cy, "l")
		case 'v':
			cy += pc.args[0]
			appendOp(buf, cx, cy, "l")
		case 'C':
			x1, y1 := pc.args[0], pc.args[1]
			x2, y2 := pc.args[2], pc.args[3]
			cx, cy = pc.args[4], pc.args[5]
			lcx, lcy = x2, y2
			appendCurve(buf, x1, y1, x2, y2, cx, cy)
		case 'c':
			x1, y1 := cx+pc.args[0], cy+pc.args[1]
			x2, y2 := cx+pc.args[2], cy+pc.args[3]
			cx, cy = cx+pc.args[4], cy+pc.args[5]
			lcx, lcy = x2, y2
			appendCurve(buf, x1, y1, x2, y2, cx, cy)
		case 'S':
			// Reflect previous control point
			x1, y1 := reflectControl(cx, cy, lcx, lcy, lastCmd)
			x2, y2 := pc.args[0], pc.args[1]
			cx, cy = pc.args[2], pc.args[3]
			lcx, lcy = x2, y2
			appendCurve(buf, x1, y1, x2, y2, cx, cy)
		case 's':
			x1, y1 := reflectControl(cx, cy, lcx, lcy, lastCmd)
			x2, y2 := cx+pc.args[0], cy+pc.args[1]
			cx, cy = cx+pc.args[2], cy+pc.args[3]
			lcx, lcy = x2, y2
			appendCurve(buf, x1, y1, x2, y2, cx, cy)
		case 'Q':
			qx, qy := pc.args[0], pc.args[1]
			ex, ey := pc.args[2], pc.args[3]
			// Convert quadratic to cubic
			c1x := cx + 2.0/3*(qx-cx)
			c1y := cy + 2.0/3*(qy-cy)
			c2x := ex + 2.0/3*(qx-ex)
			c2y := ey + 2.0/3*(qy-ey)
			cx, cy = ex, ey
			lcx, lcy = qx, qy
			appendCurve(buf, c1x, c1y, c2x, c2y, cx, cy)
		case 'q':
			qx, qy := cx+pc.args[0], cy+pc.args[1]
			ex, ey := cx+pc.args[2], cy+pc.args[3]
			c1x := cx + 2.0/3*(qx-cx)
			c1y := cy + 2.0/3*(qy-cy)
			c2x := ex + 2.0/3*(qx-ex)
			c2y := ey + 2.0/3*(qy-ey)
			cx, cy = ex, ey
			lcx, lcy = qx, qy
			appendCurve(buf, c1x, c1y, c2x, c2y, cx, cy)
		case 'T':
			qx, qy := reflectControlQ(cx, cy, lcx, lcy, lastCmd)
			ex, ey := pc.args[0], pc.args[1]
			c1x := cx + 2.0/3*(qx-cx)
			c1y := cy + 2.0/3*(qy-cy)
			c2x := ex + 2.0/3*(qx-ex)
			c2y := ey + 2.0/3*(qy-ey)
			cx, cy = ex, ey
			lcx, lcy = qx, qy
			appendCurve(buf, c1x, c1y, c2x, c2y, cx, cy)
		case 't':
			qx, qy := reflectControlQ(cx, cy, lcx, lcy, lastCmd)
			ex, ey := cx+pc.args[0], cy+pc.args[1]
			c1x := cx + 2.0/3*(qx-cx)
			c1y := cy + 2.0/3*(qy-cy)
			c2x := ex + 2.0/3*(qx-ex)
			c2y := ey + 2.0/3*(qy-ey)
			cx, cy = ex, ey
			lcx, lcy = qx, qy
			appendCurve(buf, c1x, c1y, c2x, c2y, cx, cy)
		case 'A':
			arcs := arcToBeziers(cx, cy, pc.args[0], pc.args[1], pc.args[2], pc.args[3] != 0, pc.args[4] != 0, pc.args[5], pc.args[6])
			for _, a := range arcs {
				appendCurve(buf, a[0], a[1], a[2], a[3], a[4], a[5])
			}
			cx, cy = pc.args[5], pc.args[6]
		case 'a':
			ex, ey := cx+pc.args[5], cy+pc.args[6]
			arcs := arcToBeziers(cx, cy, pc.args[0], pc.args[1], pc.args[2], pc.args[3] != 0, pc.args[4] != 0, ex, ey)
			for _, a := range arcs {
				appendCurve(buf, a[0], a[1], a[2], a[3], a[4], a[5])
			}
			cx, cy = ex, ey
		case 'Z', 'z':
			*buf = append(*buf, "h\n"...)
			cx, cy = sx, sy
		}
		lastCmd = pc.cmd
	}
}

func reflectControl(cx, cy, lcx, lcy float64, lastCmd byte) (float64, float64) {
	switch lastCmd {
	case 'C', 'c', 'S', 's':
		return 2*cx - lcx, 2*cy - lcy
	}
	return cx, cy
}

func reflectControlQ(cx, cy, lcx, lcy float64, lastCmd byte) (float64, float64) {
	switch lastCmd {
	case 'Q', 'q', 'T', 't':
		return 2*cx - lcx, 2*cy - lcy
	}
	return cx, cy
}

func appendOp(buf *[]byte, x, y float64, op string) {
	*buf = append(*buf, fmt.Sprintf("%.4g %.4g %s\n", x, y, op)...)
}

func appendCurve(buf *[]byte, x1, y1, x2, y2, x3, y3 float64) {
	*buf = append(*buf, fmt.Sprintf("%.4g %.4g %.4g %.4g %.4g %.4g c\n", x1, y1, x2, y2, x3, y3)...)
}

// arcToBeziers converts an SVG arc to a series of cubic bezier curves.
// Returns slices of [x1,y1,x2,y2,x3,y3] control points.
func arcToBeziers(x1, y1, rx, ry, phi float64, largeArc, sweep bool, x2, y2 float64) [][6]float64 {
	if rx == 0 || ry == 0 {
		return nil
	}
	rx = math.Abs(rx)
	ry = math.Abs(ry)

	sinPhi := math.Sin(phi * math.Pi / 180)
	cosPhi := math.Cos(phi * math.Pi / 180)

	// Step 1: compute (x1', y1')
	dx := (x1 - x2) / 2
	dy := (y1 - y2) / 2
	x1p := cosPhi*dx + sinPhi*dy
	y1p := -sinPhi*dx + cosPhi*dy

	// Step 2: compute (cx', cy')
	x1pSq := x1p * x1p
	y1pSq := y1p * y1p
	rxSq := rx * rx
	rySq := ry * ry

	// Ensure radii are large enough
	lambda := x1pSq/rxSq + y1pSq/rySq
	if lambda > 1 {
		scale := math.Sqrt(lambda)
		rx *= scale
		ry *= scale
		rxSq = rx * rx
		rySq = ry * ry
	}

	num := rxSq*rySq - rxSq*y1pSq - rySq*x1pSq
	den := rxSq*y1pSq + rySq*x1pSq
	if den == 0 {
		return nil
	}
	sq := num / den
	if sq < 0 {
		sq = 0
	}
	sq = math.Sqrt(sq)
	if largeArc == sweep {
		sq = -sq
	}
	cxp := sq * rx * y1p / ry
	cyp := -sq * ry * x1p / rx

	// Step 3: compute (cx, cy) from (cx', cy')
	cx := cosPhi*cxp - sinPhi*cyp + (x1+x2)/2
	cy := sinPhi*cxp + cosPhi*cyp + (y1+y2)/2

	// Step 4: compute theta1 and dtheta
	theta1 := vecAngle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	dtheta := vecAngle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if !sweep && dtheta > 0 {
		dtheta -= 2 * math.Pi
	}
	if sweep && dtheta < 0 {
		dtheta += 2 * math.Pi
	}

	// Split into segments of at most 90 degrees
	segments := int(math.Ceil(math.Abs(dtheta) / (math.Pi / 2)))
	if segments == 0 {
		segments = 1
	}
	segAngle := dtheta / float64(segments)

	var result [][6]float64
	for i := 0; i < segments; i++ {
		t1 := theta1 + float64(i)*segAngle
		t2 := t1 + segAngle
		curves := arcSegmentToBezier(cx, cy, rx, ry, sinPhi, cosPhi, t1, t2)
		result = append(result, curves)
	}
	return result
}

func vecAngle(ux, uy, vx, vy float64) float64 {
	dot := ux*vx + uy*vy
	lenU := math.Sqrt(ux*ux + uy*uy)
	lenV := math.Sqrt(vx*vx + vy*vy)
	cos := dot / (lenU * lenV)
	if cos < -1 {
		cos = -1
	}
	if cos > 1 {
		cos = 1
	}
	angle := math.Acos(cos)
	if ux*vy-uy*vx < 0 {
		angle = -angle
	}
	return angle
}

func arcSegmentToBezier(cx, cy, rx, ry, sinPhi, cosPhi, t1, t2 float64) [6]float64 {
	alpha := math.Sin(t2-t1) * (math.Sqrt(4+3*math.Pow(math.Tan((t2-t1)/2), 2)) - 1) / 3

	// Start point (on the ellipse)
	cos1, sin1 := math.Cos(t1), math.Sin(t1)
	cos2, sin2 := math.Cos(t2), math.Sin(t2)

	// Derivative at t1 and t2
	dx1 := -rx*sin1
	dy1 := ry*cos1
	dx2 := -rx*sin2
	dy2 := ry*cos2

	// End point on ellipse
	ex := rx*cos2
	ey := ry*sin2

	// Control point 1
	cp1x := rx*cos1 + alpha*dx1
	cp1y := ry*sin1 + alpha*dy1

	// Control point 2
	cp2x := ex - alpha*dx2
	cp2y := ey - alpha*dy2

	// Transform back from ellipse space
	return [6]float64{
		cosPhi*cp1x - sinPhi*cp1y + cx,
		sinPhi*cp1x + cosPhi*cp1y + cy,
		cosPhi*cp2x - sinPhi*cp2y + cx,
		sinPhi*cp2x + cosPhi*cp2y + cy,
		cosPhi*ex - sinPhi*ey + cx,
		sinPhi*ex + cosPhi*ey + cy,
	}
}
