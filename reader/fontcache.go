package reader

import (
	"strconv"
	"strings"
)

// fontInfo holds parsed font data for text extraction.
type fontInfo struct {
	name      string
	encoding  string // /WinAnsiEncoding, /MacRomanEncoding, etc.
	toUnicode map[uint16]rune
	widths    map[int]float64
	baseFont  string
	isType0   bool
}

// parseFontInfo extracts font info from a font dictionary.
func parseFontInfo(resolver *Resolver, fontDict map[string]interface{}) (*fontInfo, error) {
	fi := &fontInfo{
		widths: make(map[int]float64),
	}

	fi.baseFont = getString(fontDict, "/BaseFont")
	fi.name = fi.baseFont

	subtype := getString(fontDict, "/Subtype")
	if subtype == "/Type0" {
		fi.isType0 = true
		// Try to get descendant font info.
		if descFonts, ok := fontDict["/DescendantFonts"].([]interface{}); ok && len(descFonts) > 0 {
			descDict := getDict(resolver, descFonts[0])
			if descDict != nil {
				if bf := getString(descDict, "/BaseFont"); bf != "" {
					fi.baseFont = bf
				}
			}
		}
	}

	// Encoding.
	enc := fontDict["/Encoding"]
	enc, _ = resolver.ResolveReference(enc)
	if s, ok := enc.(string); ok {
		fi.encoding = s
	}

	// ToUnicode CMap.
	if tuRef, ok := fontDict["/ToUnicode"]; ok {
		data := getStreamData(resolver, tuRef)
		if data != nil {
			fi.toUnicode = parseToUnicodeCMap(data)
		}
	}

	// Widths.
	if firstChar, ok := getInt(fontDict, "/FirstChar"); ok {
		widths, _ := fontDict["/Widths"]
		widths, _ = resolver.ResolveReference(widths)
		if arr, ok := widths.([]interface{}); ok {
			for i, w := range arr {
				fi.widths[int(firstChar)+i] = operandFloat(w)
			}
		}
	}

	return fi, nil
}

// parseToUnicodeCMap parses a ToUnicode CMap stream and returns a mapping
// from character codes to Unicode runes.
func parseToUnicodeCMap(data []byte) map[uint16]rune {
	result := make(map[uint16]rune)
	s := string(data)

	// Parse beginbfchar ... endbfchar sections.
	for {
		idx := strings.Index(s, "beginbfchar")
		if idx < 0 {
			break
		}
		s = s[idx+len("beginbfchar"):]
		endIdx := strings.Index(s, "endbfchar")
		if endIdx < 0 {
			break
		}
		section := s[:endIdx]
		s = s[endIdx+len("endbfchar"):]

		parseBfCharSection(section, result)
	}

	// Reset and parse beginbfrange ... endbfrange sections.
	s = string(data)
	for {
		idx := strings.Index(s, "beginbfrange")
		if idx < 0 {
			break
		}
		s = s[idx+len("beginbfrange"):]
		endIdx := strings.Index(s, "endbfrange")
		if endIdx < 0 {
			break
		}
		section := s[:endIdx]
		s = s[endIdx+len("endbfrange"):]

		parseBfRangeSection(section, result)
	}

	return result
}

func parseBfCharSection(section string, result map[uint16]rune) {
	tok := NewTokenizer([]byte(section))
	for {
		srcTok, err := tok.Next()
		if err != nil || srcTok.Type == TokenEOF {
			break
		}
		dstTok, err := tok.Next()
		if err != nil || dstTok.Type == TokenEOF {
			break
		}

		srcCode := parseHexCode(srcTok)
		dstRune := parseHexRune(dstTok)
		if srcCode >= 0 && dstRune >= 0 {
			result[uint16(srcCode)] = rune(dstRune)
		}
	}
}

func parseBfRangeSection(section string, result map[uint16]rune) {
	tok := NewTokenizer([]byte(section))
	for {
		startTok, err := tok.Next()
		if err != nil || startTok.Type == TokenEOF {
			break
		}
		endTok, err := tok.Next()
		if err != nil || endTok.Type == TokenEOF {
			break
		}
		dstTok, err := tok.Next()
		if err != nil || dstTok.Type == TokenEOF {
			break
		}

		startCode := parseHexCode(startTok)
		endCode := parseHexCode(endTok)
		if startCode < 0 || endCode < 0 {
			continue
		}

		if dstTok.Type == TokenArrayBegin {
			// Array of destination values.
			for code := startCode; code <= endCode; code++ {
				valTok, err := tok.Next()
				if err != nil || valTok.Type == TokenArrayEnd || valTok.Type == TokenEOF {
					break
				}
				r := parseHexRune(valTok)
				if r >= 0 {
					result[uint16(code)] = rune(r)
				}
			}
			// Consume remaining array end if needed.
			for {
				pk, _ := tok.Peek()
				if pk.Type == TokenArrayEnd {
					tok.Next()
					break
				}
				if pk.Type == TokenEOF {
					break
				}
				tok.Next()
			}
		} else {
			// Single destination: incrementing.
			dstRune := parseHexRune(dstTok)
			if dstRune >= 0 {
				for code := startCode; code <= endCode; code++ {
					result[uint16(code)] = rune(dstRune + (code - startCode))
				}
			}
		}
	}
}

func parseHexCode(t Token) int {
	if t.Type == TokenHexString {
		v, err := strconv.ParseUint(t.Value, 16, 16)
		if err != nil {
			return -1
		}
		return int(v)
	}
	return -1
}

func parseHexRune(t Token) int {
	if t.Type == TokenHexString {
		v, err := strconv.ParseUint(t.Value, 16, 32)
		if err != nil {
			return -1
		}
		return int(v)
	}
	return -1
}
