package converter

import (
	"math"
	"strings"

	"github.com/oarkflow/pdf/reader"
)

// graphicsState tracks the current PDF graphics state for styled extraction.
type graphicsState struct {
	ctm         [6]float64 // current transformation matrix
	fillColor   [3]float64 // RGB fill color (0-1)
	strokeColor [3]float64 // RGB stroke color (0-1)
}

// styledExtractor parses PDF content streams and produces StyledSpans.
type styledExtractor struct {
	resolver *reader.Resolver
	fonts    map[string]*reader.FontInfo
	spans    []StyledSpan

	// Graphics state stack.
	gs      graphicsState
	gsStack []graphicsState

	// Text state.
	curFont     *reader.FontInfo
	curFontName string
	curFontSize float64
	tm          [6]float64 // text matrix
	lm          [6]float64 // line matrix
	tdx, tdy    float64    // last Td offsets
	inText      bool

	// Image tracking.
	images     []imageRef
	ctmAtDo    [6]float64
}

type imageRef struct {
	name string
	ctm  [6]float64
}

func newStyledExtractor(resolver *reader.Resolver) *styledExtractor {
	return &styledExtractor{
		resolver: resolver,
		fonts:    make(map[string]*reader.FontInfo),
		gs: graphicsState{
			ctm: [6]float64{1, 0, 0, 1, 0, 0},
		},
	}
}

// loadFonts loads all fonts from page resources.
func (e *styledExtractor) loadFonts(resources map[string]interface{}) error {
	fontDict, ok := resources["/Font"]
	if !ok {
		return nil
	}
	fontMap, err := e.resolver.ResolveReference(fontDict)
	if err != nil {
		return err
	}
	fm, ok := fontMap.(map[string]interface{})
	if !ok {
		return nil
	}
	for name, ref := range fm {
		fontObj, err := e.resolver.ResolveReference(ref)
		if err != nil {
			continue
		}
		fd, ok := fontObj.(map[string]interface{})
		if !ok {
			continue
		}
		fi, err := reader.ParseFontInfo(e.resolver, fd)
		if err == nil {
			e.fonts[name] = fi
		}
	}
	return nil
}

// parse processes a content stream and extracts styled spans and image references.
func (e *styledExtractor) parse(content []byte) {
	tok := reader.NewTokenizer(content)
	var operands []interface{}

	for {
		t, err := tok.Next()
		if err != nil || t.Type == reader.TokenEOF {
			break
		}

		switch t.Type {
		case reader.TokenInteger:
			operands = append(operands, t.Int)
		case reader.TokenReal:
			operands = append(operands, t.Real)
		case reader.TokenString:
			operands = append(operands, t.Value)
		case reader.TokenHexString:
			operands = append(operands, hexDecode(t.Value))
		case reader.TokenName:
			operands = append(operands, "/"+t.Value)
		case reader.TokenArrayBegin:
			arr := parseArray(tok)
			operands = append(operands, arr)
		case reader.TokenKeyword:
			e.execOperator(t.Value, operands)
			operands = operands[:0]
		default:
			operands = append(operands, t.Value)
		}
	}
}

func parseArray(tok *reader.Tokenizer) []interface{} {
	var arr []interface{}
	for {
		t, err := tok.Next()
		if err != nil || t.Type == reader.TokenArrayEnd || t.Type == reader.TokenEOF {
			break
		}
		switch t.Type {
		case reader.TokenInteger:
			arr = append(arr, t.Int)
		case reader.TokenReal:
			arr = append(arr, t.Real)
		case reader.TokenString:
			arr = append(arr, t.Value)
		case reader.TokenHexString:
			arr = append(arr, hexDecode(t.Value))
		case reader.TokenName:
			arr = append(arr, "/"+t.Value)
		case reader.TokenArrayBegin:
			arr = append(arr, parseArray(tok))
		default:
			arr = append(arr, t.Value)
		}
	}
	return arr
}

func (e *styledExtractor) execOperator(op string, operands []interface{}) {
	switch op {
	// Graphics state operators.
	case "q":
		e.gsStack = append(e.gsStack, e.gs)
	case "Q":
		if len(e.gsStack) > 0 {
			e.gs = e.gsStack[len(e.gsStack)-1]
			e.gsStack = e.gsStack[:len(e.gsStack)-1]
		}
	case "cm":
		if len(operands) >= 6 {
			var m [6]float64
			for i := 0; i < 6; i++ {
				m[i] = opFloat(operands[i])
			}
			e.gs.ctm = multiplyMatrix(m, e.gs.ctm)
		}

	// Color operators — fill.
	case "rg":
		if len(operands) >= 3 {
			e.gs.fillColor = [3]float64{
				opFloat(operands[0]),
				opFloat(operands[1]),
				opFloat(operands[2]),
			}
		}
	case "g":
		if len(operands) >= 1 {
			v := opFloat(operands[0])
			e.gs.fillColor = [3]float64{v, v, v}
		}
	case "k":
		if len(operands) >= 4 {
			c := opFloat(operands[0])
			m := opFloat(operands[1])
			y := opFloat(operands[2])
			k := opFloat(operands[3])
			e.gs.fillColor = cmykToRGB(c, m, y, k)
		}
	case "cs":
		// Set color space for non-stroking — we track but don't change color.
	case "sc", "scn":
		if len(operands) >= 3 {
			e.gs.fillColor = [3]float64{
				opFloat(operands[0]),
				opFloat(operands[1]),
				opFloat(operands[2]),
			}
		} else if len(operands) >= 1 {
			v := opFloat(operands[0])
			e.gs.fillColor = [3]float64{v, v, v}
		}

	// Color operators — stroke.
	case "RG":
		if len(operands) >= 3 {
			e.gs.strokeColor = [3]float64{
				opFloat(operands[0]),
				opFloat(operands[1]),
				opFloat(operands[2]),
			}
		}
	case "G":
		if len(operands) >= 1 {
			v := opFloat(operands[0])
			e.gs.strokeColor = [3]float64{v, v, v}
		}
	case "K":
		if len(operands) >= 4 {
			c := opFloat(operands[0])
			m := opFloat(operands[1])
			y := opFloat(operands[2])
			k := opFloat(operands[3])
			e.gs.strokeColor = cmykToRGB(c, m, y, k)
		}
	case "CS":
		// Set color space for stroking.
	case "SC", "SCN":
		if len(operands) >= 3 {
			e.gs.strokeColor = [3]float64{
				opFloat(operands[0]),
				opFloat(operands[1]),
				opFloat(operands[2]),
			}
		} else if len(operands) >= 1 {
			v := opFloat(operands[0])
			e.gs.strokeColor = [3]float64{v, v, v}
		}

	// Text object operators.
	case "BT":
		e.inText = true
		e.tm = [6]float64{1, 0, 0, 1, 0, 0}
		e.lm = e.tm
	case "ET":
		e.inText = false

	// Text state operators.
	case "Tf":
		if len(operands) >= 2 {
			fontName, _ := operands[0].(string)
			size := opFloat(operands[1])
			e.curFontSize = size
			e.curFontName = fontName
			if fi, ok := e.fonts[fontName]; ok {
				e.curFont = fi
			}
		}

	// Text positioning operators.
	case "Td":
		if len(operands) >= 2 {
			tx := opFloat(operands[0])
			ty := opFloat(operands[1])
			e.lm[4] += tx
			e.lm[5] += ty
			e.tm = e.lm
			e.tdx = tx
			e.tdy = ty
		}
	case "TD":
		if len(operands) >= 2 {
			tx := opFloat(operands[0])
			ty := opFloat(operands[1])
			e.lm[4] += tx
			e.lm[5] += ty
			e.tm = e.lm
			e.tdx = tx
			e.tdy = ty
		}
	case "Tm":
		if len(operands) >= 6 {
			for i := 0; i < 6; i++ {
				e.tm[i] = opFloat(operands[i])
			}
			e.lm = e.tm
		}
	case "T*":
		e.lm[5] += e.tdy
		e.tm = e.lm

	// Text showing operators.
	case "Tj":
		if len(operands) >= 1 {
			s, _ := operands[0].(string)
			text := e.decodeString(s)
			e.addSpan(text)
		}
	case "TJ":
		if len(operands) >= 1 {
			if arr, ok := operands[0].([]interface{}); ok {
				var buf strings.Builder
				for _, item := range arr {
					switch v := item.(type) {
					case string:
						buf.WriteString(e.decodeString(v))
					case int64:
						if v <= -100 {
							buf.WriteByte(' ')
						}
					case float64:
						if v <= -100 {
							buf.WriteByte(' ')
						}
					}
				}
				e.addSpan(buf.String())
			}
		}
	case "'":
		e.lm[5] += e.tdy
		e.tm = e.lm
		if len(operands) >= 1 {
			s, _ := operands[0].(string)
			e.addSpan(e.decodeString(s))
		}
	case "\"":
		e.lm[5] += e.tdy
		e.tm = e.lm
		if len(operands) >= 3 {
			s, _ := operands[2].(string)
			e.addSpan(e.decodeString(s))
		}

	// XObject invocation (images).
	case "Do":
		if len(operands) >= 1 {
			name, _ := operands[0].(string)
			e.images = append(e.images, imageRef{
				name: name,
				ctm:  e.gs.ctm,
			})
		}
	}
}

func (e *styledExtractor) decodeString(s string) string {
	return reader.DecodeString(e.curFont, s)
}

func (e *styledExtractor) addSpan(text string) {
	if text == "" {
		return
	}

	bold, italic := detectFontStyle(e.curFont)

	// Compute the actual position by combining text matrix with CTM.
	// Text rendering matrix = Tm × CTM.
	trm := multiplyMatrix(e.tm, e.gs.ctm)

	e.spans = append(e.spans, StyledSpan{
		Text:     text,
		X:        trm[4],
		Y:        trm[5],
		FontName: e.fontDisplayName(),
		FontSize: math.Abs(e.curFontSize * trm[3]),
		Bold:     bold,
		Italic:   italic,
		Color:    e.gs.fillColor,
	})
}

func (e *styledExtractor) fontDisplayName() string {
	if e.curFont == nil {
		return ""
	}
	name := e.curFont.BaseFont
	// Strip subset prefix (e.g., "ABCDEF+" prefix).
	if idx := strings.Index(name, "+"); idx >= 0 && idx <= 6 {
		name = name[idx+1:]
	}
	return name
}

// detectFontStyle infers bold/italic from the font name.
func detectFontStyle(fi *reader.FontInfo) (bold, italic bool) {
	if fi == nil {
		return false, false
	}
	name := strings.ToLower(fi.BaseFont)
	bold = strings.Contains(name, "bold") || strings.Contains(name, "black") || strings.Contains(name, "heavy")
	italic = strings.Contains(name, "italic") || strings.Contains(name, "oblique")
	return
}

// multiplyMatrix multiplies two 2D affine transformation matrices.
// Each is [a b c d e f] representing:
// | a b 0 |
// | c d 0 |
// | e f 1 |
func multiplyMatrix(a, b [6]float64) [6]float64 {
	return [6]float64{
		a[0]*b[0] + a[1]*b[2],
		a[0]*b[1] + a[1]*b[3],
		a[2]*b[0] + a[3]*b[2],
		a[2]*b[1] + a[3]*b[3],
		a[4]*b[0] + a[5]*b[2] + b[4],
		a[4]*b[1] + a[5]*b[3] + b[5],
	}
}

func cmykToRGB(c, m, y, k float64) [3]float64 {
	return [3]float64{
		(1 - c) * (1 - k),
		(1 - m) * (1 - k),
		(1 - y) * (1 - k),
	}
}

func opFloat(v interface{}) float64 {
	return reader.OperandFloat(v)
}

func hexDecode(hex string) string {
	// Simple hex string decode — pairs of hex digits to bytes.
	hex = strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			return r
		}
		return -1
	}, hex)
	if len(hex)%2 != 0 {
		hex += "0"
	}
	var buf []byte
	for i := 0; i+1 < len(hex); i += 2 {
		hi := unhexByte(hex[i])
		lo := unhexByte(hex[i+1])
		buf = append(buf, hi<<4|lo)
	}
	return string(buf)
}

func unhexByte(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}
