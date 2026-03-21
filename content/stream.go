package content

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type TextItem struct {
	Text   string  // if non-empty, render as string
	Offset float64 // if Text is empty, render as number (kerning)
}

type Stream struct {
	buf bytes.Buffer
}

func New() *Stream                { return &Stream{} }
func (s *Stream) Bytes() []byte   { return s.buf.Bytes() }
func (s *Stream) String() string  { return s.buf.String() }

// formatFloat formats a float trimming trailing zeros (max 4 decimal places).
func formatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', 4, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `(`, `\(`)
	s = strings.ReplaceAll(s, `)`, `\)`)
	return s
}

// Graphics state

func (s *Stream) SaveState() {
	fmt.Fprint(&s.buf, "q\n")
}

func (s *Stream) RestoreState() {
	fmt.Fprint(&s.buf, "Q\n")
}

func (s *Stream) SetCTM(a, b, c, d, e, f float64) {
	fmt.Fprintf(&s.buf, "%s %s %s %s %s %s cm\n",
		formatFloat(a), formatFloat(b), formatFloat(c),
		formatFloat(d), formatFloat(e), formatFloat(f))
}

// Text

func (s *Stream) BeginText() {
	fmt.Fprint(&s.buf, "BT\n")
}

func (s *Stream) EndText() {
	fmt.Fprint(&s.buf, "ET\n")
}

func (s *Stream) SetFont(name string, size float64) {
	fmt.Fprintf(&s.buf, "/%s %s Tf\n", name, formatFloat(size))
}

func (s *Stream) MoveText(tx, ty float64) {
	fmt.Fprintf(&s.buf, "%s %s Td\n", formatFloat(tx), formatFloat(ty))
}

func (s *Stream) ShowText(text string) {
	fmt.Fprintf(&s.buf, "(%s) Tj\n", escapeString(text))
}

func (s *Stream) ShowTextArray(items []TextItem) {
	var b strings.Builder
	b.WriteByte('[')
	for _, item := range items {
		if item.Text != "" {
			fmt.Fprintf(&b, "(%s)", escapeString(item.Text))
		} else {
			b.WriteString(formatFloat(item.Offset))
		}
	}
	b.WriteByte(']')
	fmt.Fprintf(&s.buf, "%s TJ\n", b.String())
}

func (s *Stream) SetTextLeading(leading float64) {
	fmt.Fprintf(&s.buf, "%s TL\n", formatFloat(leading))
}

func (s *Stream) NextLine() {
	fmt.Fprint(&s.buf, "T*\n")
}

func (s *Stream) SetCharSpacing(spacing float64) {
	fmt.Fprintf(&s.buf, "%s Tc\n", formatFloat(spacing))
}

func (s *Stream) SetWordSpacing(spacing float64) {
	fmt.Fprintf(&s.buf, "%s Tw\n", formatFloat(spacing))
}

func (s *Stream) SetTextRenderMode(mode int) {
	fmt.Fprintf(&s.buf, "%d Tr\n", mode)
}

func (s *Stream) SetTextRise(rise float64) {
	fmt.Fprintf(&s.buf, "%s Ts\n", formatFloat(rise))
}

// Color

func (s *Stream) SetFillColorRGB(r, g, b float64) {
	fmt.Fprintf(&s.buf, "%s %s %s rg\n", formatFloat(r), formatFloat(g), formatFloat(b))
}

func (s *Stream) SetStrokeColorRGB(r, g, b float64) {
	fmt.Fprintf(&s.buf, "%s %s %s RG\n", formatFloat(r), formatFloat(g), formatFloat(b))
}

func (s *Stream) SetFillColorCMYK(c, m, y, k float64) {
	fmt.Fprintf(&s.buf, "%s %s %s %s k\n",
		formatFloat(c), formatFloat(m), formatFloat(y), formatFloat(k))
}

func (s *Stream) SetStrokeColorCMYK(c, m, y, k float64) {
	fmt.Fprintf(&s.buf, "%s %s %s %s K\n",
		formatFloat(c), formatFloat(m), formatFloat(y), formatFloat(k))
}

func (s *Stream) SetFillGray(g float64) {
	fmt.Fprintf(&s.buf, "%s g\n", formatFloat(g))
}

func (s *Stream) SetStrokeGray(g float64) {
	fmt.Fprintf(&s.buf, "%s G\n", formatFloat(g))
}

// Path construction

func (s *Stream) MoveTo(x, y float64) {
	fmt.Fprintf(&s.buf, "%s %s m\n", formatFloat(x), formatFloat(y))
}

func (s *Stream) LineTo(x, y float64) {
	fmt.Fprintf(&s.buf, "%s %s l\n", formatFloat(x), formatFloat(y))
}

func (s *Stream) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	fmt.Fprintf(&s.buf, "%s %s %s %s %s %s c\n",
		formatFloat(x1), formatFloat(y1), formatFloat(x2),
		formatFloat(y2), formatFloat(x3), formatFloat(y3))
}

func (s *Stream) Rectangle(x, y, w, h float64) {
	fmt.Fprintf(&s.buf, "%s %s %s %s re\n",
		formatFloat(x), formatFloat(y), formatFloat(w), formatFloat(h))
}

func (s *Stream) ClosePath() {
	fmt.Fprint(&s.buf, "h\n")
}

// Path painting

func (s *Stream) Stroke() {
	fmt.Fprint(&s.buf, "S\n")
}

func (s *Stream) Fill() {
	fmt.Fprint(&s.buf, "f\n")
}

func (s *Stream) FillAndStroke() {
	fmt.Fprint(&s.buf, "B\n")
}

func (s *Stream) FillEvenOdd() {
	fmt.Fprint(&s.buf, "f*\n")
}

func (s *Stream) CloseAndStroke() {
	fmt.Fprint(&s.buf, "s\n")
}

func (s *Stream) Clip() {
	fmt.Fprint(&s.buf, "W n\n")
}

// Line style

func (s *Stream) SetLineWidth(w float64) {
	fmt.Fprintf(&s.buf, "%s w\n", formatFloat(w))
}

func (s *Stream) SetLineCap(cap int) {
	fmt.Fprintf(&s.buf, "%d J\n", cap)
}

func (s *Stream) SetLineJoin(join int) {
	fmt.Fprintf(&s.buf, "%d j\n", join)
}

func (s *Stream) SetDashPattern(array []float64, phase float64) {
	var parts []string
	for _, v := range array {
		parts = append(parts, formatFloat(v))
	}
	fmt.Fprintf(&s.buf, "[%s] %s d\n", strings.Join(parts, " "), formatFloat(phase))
}

// XObject

func (s *Stream) DrawXObject(name string) {
	fmt.Fprintf(&s.buf, "/%s Do\n", name)
}

// Marked content

// BeginMarkedContent writes a BDC operator with a tag and marked content ID.
func (s *Stream) BeginMarkedContent(tag string, mcid int) {
	fmt.Fprintf(&s.buf, "/%s <</MCID %d>> BDC\n", tag, mcid)
}

// BeginMarkedContentBMC writes a BMC operator with just a tag (no properties).
func (s *Stream) BeginMarkedContentBMC(tag string) {
	fmt.Fprintf(&s.buf, "/%s BMC\n", tag)
}

// EndMarkedContent writes an EMC operator.
func (s *Stream) EndMarkedContent() {
	fmt.Fprint(&s.buf, "EMC\n")
}

// Transparency

func (s *Stream) SetExtGState(name string) {
	fmt.Fprintf(&s.buf, "/%s gs\n", name)
}
