package reader

import (
	"fmt"
	"strconv"
	"strings"
)

// ExtractText extracts the text content from a 0-indexed page.
func (r *Reader) ExtractText(pageNum int) (string, error) {
	page, err := r.Page(pageNum)
	if err != nil {
		return "", err
	}

	ext := &textExtractor{
		resolver: r.resolver,
		fonts:    make(map[string]*fontInfo),
	}

	// Load fonts from page resources.
	if fontDict, ok := page.Resources["/Font"]; ok {
		fontMap, err := r.resolver.ResolveReference(fontDict)
		if err != nil {
			return "", fmt.Errorf("reader: failed to resolve /Font: %w", err)
		}
		if fm, ok := fontMap.(map[string]interface{}); ok {
			for name, ref := range fm {
				fontObj, err := r.resolver.ResolveReference(ref)
				if err != nil {
					return "", fmt.Errorf("reader: failed to resolve font %s: %w", name, err)
				}
				if fd, ok := fontObj.(map[string]interface{}); ok {
					fi, err := parseFontInfo(r.resolver, fd)
					if err == nil {
						ext.fonts[name] = fi
					}
				}
			}
		}
	}

	ext.parse(page.Contents)

	return ext.buildText(), nil
}

type textSpan struct {
	text     string
	x, y     float64
	fontSize float64
}

type textExtractor struct {
	resolver *Resolver
	fonts    map[string]*fontInfo
	spans    []textSpan

	// Current graphics state.
	curFont     *fontInfo
	curFontSize float64
	tm          [6]float64 // text matrix
	lm          [6]float64 // line matrix
	tdx, tdy    float64    // last Td offsets
	inText      bool
}

func (e *textExtractor) parse(content []byte) {
	tok := NewTokenizer(content)
	var operands []interface{}

	for {
		t, err := tok.Next()
		if err != nil || t.Type == TokenEOF {
			break
		}

		switch t.Type {
		case TokenInteger:
			operands = append(operands, t.Int)
		case TokenReal:
			operands = append(operands, t.Real)
		case TokenString:
			operands = append(operands, t.Value)
		case TokenHexString:
			operands = append(operands, decodeHex(t.Value))
		case TokenName:
			operands = append(operands, "/"+t.Value)
		case TokenArrayBegin:
			arr := e.parseArray(tok)
			operands = append(operands, arr)
		case TokenKeyword:
			e.execOperator(t.Value, operands)
			operands = operands[:0]
		default:
			operands = append(operands, t.Value)
		}
	}
}

func (e *textExtractor) parseArray(tok *Tokenizer) []interface{} {
	var arr []interface{}
	for {
		t, err := tok.Next()
		if err != nil || t.Type == TokenArrayEnd || t.Type == TokenEOF {
			break
		}
		switch t.Type {
		case TokenInteger:
			arr = append(arr, t.Int)
		case TokenReal:
			arr = append(arr, t.Real)
		case TokenString:
			arr = append(arr, t.Value)
		case TokenHexString:
			arr = append(arr, decodeHex(t.Value))
		case TokenName:
			arr = append(arr, "/"+t.Value)
		case TokenArrayBegin:
			arr = append(arr, e.parseArray(tok))
		default:
			arr = append(arr, t.Value)
		}
	}
	return arr
}

func (e *textExtractor) execOperator(op string, operands []interface{}) {
	switch op {
	case "BT":
		e.inText = true
		e.tm = [6]float64{1, 0, 0, 1, 0, 0}
		e.lm = e.tm
	case "ET":
		e.inText = false

	case "Tf":
		if len(operands) >= 2 {
			fontName, _ := operands[0].(string)
			size := operandFloat(operands[1])
			e.curFontSize = size
			if fi, ok := e.fonts[fontName]; ok {
				e.curFont = fi
			}
		}

	case "Td":
		if len(operands) >= 2 {
			tx := operandFloat(operands[0])
			ty := operandFloat(operands[1])
			e.lm[4] += tx
			e.lm[5] += ty
			e.tm = e.lm
			e.tdx = tx
			e.tdy = ty
		}

	case "TD":
		if len(operands) >= 2 {
			tx := operandFloat(operands[0])
			ty := operandFloat(operands[1])
			e.lm[4] += tx
			e.lm[5] += ty
			e.tm = e.lm
			e.tdx = tx
			e.tdy = ty
		}

	case "Tm":
		if len(operands) >= 6 {
			for i := 0; i < 6; i++ {
				e.tm[i] = operandFloat(operands[i])
			}
			e.lm = e.tm
		}

	case "T*":
		e.lm[5] += e.tdy
		e.tm = e.lm

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
						// Large negative displacement = space.
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
		// Move to next line and show string.
		e.lm[5] += e.tdy
		e.tm = e.lm
		if len(operands) >= 1 {
			s, _ := operands[0].(string)
			e.addSpan(e.decodeString(s))
		}

	case "\"":
		// Set word/char spacing, move, show.
		e.lm[5] += e.tdy
		e.tm = e.lm
		if len(operands) >= 3 {
			s, _ := operands[2].(string)
			e.addSpan(e.decodeString(s))
		}
	}
}

func (e *textExtractor) decodeString(s string) string {
	if e.curFont == nil {
		return s
	}
	if e.curFont.isType0 && e.curFont.toUnicode != nil {
		return e.decodeType0(s)
	}
	if e.curFont.toUnicode != nil {
		var buf strings.Builder
		for i := 0; i < len(s); i++ {
			code := uint16(s[i])
			if r, ok := e.curFont.toUnicode[code]; ok {
				buf.WriteRune(r)
			} else {
				buf.WriteByte(s[i])
			}
		}
		return buf.String()
	}
	// Use encoding.
	return decodeWithEncoding(s, e.curFont.encoding)
}

func (e *textExtractor) decodeType0(s string) string {
	var buf strings.Builder
	for i := 0; i+1 < len(s); i += 2 {
		code := uint16(s[i])<<8 | uint16(s[i+1])
		if r, ok := e.curFont.toUnicode[code]; ok {
			buf.WriteRune(r)
		} else {
			buf.WriteRune(rune(code))
		}
	}
	return buf.String()
}

func (e *textExtractor) addSpan(text string) {
	if text == "" {
		return
	}
	e.spans = append(e.spans, textSpan{
		text:     text,
		x:        e.tm[4],
		y:        e.tm[5],
		fontSize: e.curFontSize,
	})
}

func (e *textExtractor) buildText() string {
	if len(e.spans) == 0 {
		return ""
	}

	var buf strings.Builder
	var lastY float64
	first := true

	for _, span := range e.spans {
		if first {
			first = false
			lastY = span.y
		} else {
			dy := lastY - span.y
			if dy > 1 || dy < -1 {
				buf.WriteByte('\n')
				lastY = span.y
			}
		}
		buf.WriteString(span.text)
	}

	return buf.String()
}

func operandFloat(v interface{}) float64 {
	switch n := v.(type) {
	case int64:
		return float64(n)
	case float64:
		return n
	case int:
		return float64(n)
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	}
	return 0
}

func decodeWithEncoding(s string, encoding string) string {
	switch encoding {
	case "/WinAnsiEncoding":
		return decodeWinAnsi(s)
	case "/MacRomanEncoding":
		return decodeMacRoman(s)
	}
	return s
}

// decodeWinAnsi decodes bytes using Windows-1252 encoding.
func decodeWinAnsi(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		r := winAnsiToUnicode[b]
		if r == 0 {
			r = rune(b)
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

// decodeMacRoman decodes bytes using Mac Roman encoding.
func decodeMacRoman(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < 128 {
			buf.WriteByte(b)
		} else {
			r := macRomanToUnicode[b-128]
			if r == 0 {
				r = rune(b)
			}
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// winAnsiToUnicode maps Windows-1252 bytes 0x80-0x9F to Unicode code points.
var winAnsiToUnicode [256]rune

func init() {
	for i := 0; i < 256; i++ {
		winAnsiToUnicode[i] = rune(i)
	}
	// Windows-1252 special characters in 0x80-0x9F range.
	m := map[byte]rune{
		0x80: 0x20AC, 0x82: 0x201A, 0x83: 0x0192, 0x84: 0x201E,
		0x85: 0x2026, 0x86: 0x2020, 0x87: 0x2021, 0x88: 0x02C6,
		0x89: 0x2030, 0x8A: 0x0160, 0x8B: 0x2039, 0x8C: 0x0152,
		0x8E: 0x017D, 0x91: 0x2018, 0x92: 0x2019, 0x93: 0x201C,
		0x94: 0x201D, 0x95: 0x2022, 0x96: 0x2013, 0x97: 0x2014,
		0x98: 0x02DC, 0x99: 0x2122, 0x9A: 0x0161, 0x9B: 0x203A,
		0x9C: 0x0153, 0x9E: 0x017E, 0x9F: 0x0178,
	}
	for k, v := range m {
		winAnsiToUnicode[k] = v
	}
}

var macRomanToUnicode = [128]rune{
	0x00C4, 0x00C5, 0x00C7, 0x00C9, 0x00D1, 0x00D6, 0x00DC, 0x00E1,
	0x00E0, 0x00E2, 0x00E4, 0x00E3, 0x00E5, 0x00E7, 0x00E9, 0x00E8,
	0x00EA, 0x00EB, 0x00ED, 0x00EC, 0x00EE, 0x00EF, 0x00F1, 0x00F3,
	0x00F2, 0x00F4, 0x00F6, 0x00F5, 0x00FA, 0x00F9, 0x00FB, 0x00FC,
	0x2020, 0x00B0, 0x00A2, 0x00A3, 0x00A7, 0x2022, 0x00B6, 0x00DF,
	0x00AE, 0x00A9, 0x2122, 0x00B4, 0x00A8, 0x2260, 0x00C6, 0x00D8,
	0x221E, 0x00B1, 0x2264, 0x2265, 0x00A5, 0x00B5, 0x2202, 0x2211,
	0x220F, 0x03C0, 0x222B, 0x00AA, 0x00BA, 0x03A9, 0x00E6, 0x00F8,
	0x00BF, 0x00A1, 0x00AC, 0x221A, 0x0192, 0x2248, 0x2206, 0x00AB,
	0x00BB, 0x2026, 0x00A0, 0x00C0, 0x00C3, 0x00D5, 0x0152, 0x0153,
	0x2013, 0x2014, 0x201C, 0x201D, 0x2018, 0x2019, 0x00F7, 0x25CA,
	0x00FF, 0x0178, 0x2044, 0x20AC, 0x2039, 0x203A, 0xFB01, 0xFB02,
	0x2021, 0x00B7, 0x201A, 0x201E, 0x2030, 0x00C2, 0x00CA, 0x00C1,
	0x00CB, 0x00C8, 0x00CD, 0x00CE, 0x00CF, 0x00CC, 0x00D3, 0x00D4,
	0xF8FF, 0x00D2, 0x00DA, 0x00DB, 0x00D9, 0x0131, 0x02C6, 0x02DC,
	0x00AF, 0x02D8, 0x02D9, 0x02DA, 0x00B8, 0x02DD, 0x02DB, 0x02C7,
}

func getString(dict map[string]interface{}, key string) string {
	v, _ := dict[key].(string)
	return v
}

func getFloat(dict map[string]interface{}, key string) float64 {
	v, ok := dict[key]
	if !ok {
		return 0
	}
	return operandFloat(v)
}

// Helpers used for font resolution.
func getDict(resolver *Resolver, v interface{}) (map[string]interface{}, error) {
	resolved, err := resolver.ResolveReference(v)
	if err != nil {
		return nil, fmt.Errorf("reader: getDict resolve failed: %w", err)
	}
	d, _ := resolved.(map[string]interface{})
	return d, nil
}

func getStreamData(resolver *Resolver, v interface{}) ([]byte, error) {
	resolved, err := resolver.ResolveReference(v)
	if err != nil {
		return nil, fmt.Errorf("reader: getStreamData resolve failed: %w", err)
	}
	if so, ok := resolved.(*StreamObject); ok {
		data, err := resolver.DecompressStream(so.Dict, so.Data)
		if err != nil {
			return nil, fmt.Errorf("reader: getStreamData decompress failed: %w", err)
		}
		return data, nil
	}
	return nil, nil
}

// Exported helper for use in merge.
func formatFloat(f float64) string {
	return fmt.Sprintf("%g", f)
}
