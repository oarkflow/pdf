package reader

import (
	"strconv"
	"strings"
)

// fontInfo holds parsed font data for text extraction.
type fontInfo struct {
	name        string
	encoding    string // /WinAnsiEncoding, /MacRomanEncoding, etc.
	toUnicode   map[uint16]rune
	differences map[byte]string // /Differences array from encoding dict
	widths      map[int]float64
	baseFont    string
	isType0     bool
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
			descDict, err := getDict(resolver, descFonts[0])
			if err != nil {
				return nil, err
			}
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
	} else if encDict, ok := enc.(map[string]interface{}); ok {
		// Encoding dictionary with /BaseEncoding and /Differences.
		if base, ok := encDict["/BaseEncoding"].(string); ok {
			fi.encoding = base
		}
		// Parse /Differences array.
		if diffRef, ok := encDict["/Differences"]; ok {
			diffObj, _ := resolver.ResolveReference(diffRef)
			if diffArr, ok := diffObj.([]interface{}); ok {
				fi.differences = parseDifferences(diffArr)
			}
		}
	}

	// ToUnicode CMap.
	if tuRef, ok := fontDict["/ToUnicode"]; ok {
		data, err := getStreamData(resolver, tuRef)
		if err != nil {
			return nil, err
		}
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

// parseDifferences parses a PDF /Differences array.
// The format is: [code /name1 /name2 ... code /name3 ...]
// where code sets the starting character code and subsequent names map sequentially.
func parseDifferences(arr []interface{}) map[byte]string {
	result := make(map[byte]string)
	code := 0
	for _, item := range arr {
		switch v := item.(type) {
		case int64:
			code = int(v)
		case float64:
			code = int(v)
		case int:
			code = v
		case string:
			if strings.HasPrefix(v, "/") {
				if code >= 0 && code <= 255 {
					result[byte(code)] = v[1:] // strip leading /
				}
				code++
			}
		}
	}
	return result
}

// GlyphNameToUnicode maps common Adobe glyph names to Unicode runes.
var GlyphNameToUnicode = map[string]rune{
	"space": ' ', "exclam": '!', "quotedbl": '"', "numbersign": '#',
	"dollar": '$', "percent": '%', "ampersand": '&', "quotesingle": '\'',
	"parenleft": '(', "parenright": ')', "asterisk": '*', "plus": '+',
	"comma": ',', "hyphen": '-', "period": '.', "slash": '/',
	"zero": '0', "one": '1', "two": '2', "three": '3',
	"four": '4', "five": '5', "six": '6', "seven": '7',
	"eight": '8', "nine": '9', "colon": ':', "semicolon": ';',
	"less": '<', "equal": '=', "greater": '>', "question": '?',
	"at": '@',
	"A": 'A', "B": 'B', "C": 'C', "D": 'D', "E": 'E', "F": 'F',
	"G": 'G', "H": 'H', "I": 'I', "J": 'J', "K": 'K', "L": 'L',
	"M": 'M', "N": 'N', "O": 'O', "P": 'P', "Q": 'Q', "R": 'R',
	"S": 'S', "T": 'T', "U": 'U', "V": 'V', "W": 'W', "X": 'X',
	"Y": 'Y', "Z": 'Z',
	"bracketleft": '[', "backslash": '\\', "bracketright": ']',
	"asciicircum": '^', "underscore": '_', "grave": '`',
	"a": 'a', "b": 'b', "c": 'c', "d": 'd', "e": 'e', "f": 'f',
	"g": 'g', "h": 'h', "i": 'i', "j": 'j', "k": 'k', "l": 'l',
	"m": 'm', "n": 'n', "o": 'o', "p": 'p', "q": 'q', "r": 'r',
	"s": 's', "t": 't', "u": 'u', "v": 'v', "w": 'w', "x": 'x',
	"y": 'y', "z": 'z',
	"braceleft": '{', "bar": '|', "braceright": '}', "asciitilde": '~',
	// Extended Latin
	"Agrave": 'À', "Aacute": 'Á', "Acircumflex": 'Â', "Atilde": 'Ã',
	"Adieresis": 'Ä', "Aring": 'Å', "AE": 'Æ', "Ccedilla": 'Ç',
	"Egrave": 'È', "Eacute": 'É', "Ecircumflex": 'Ê', "Edieresis": 'Ë',
	"Igrave": 'Ì', "Iacute": 'Í', "Icircumflex": 'Î', "Idieresis": 'Ï',
	"Eth": 'Ð', "Ntilde": 'Ñ', "Ograve": 'Ò', "Oacute": 'Ó',
	"Ocircumflex": 'Ô', "Otilde": 'Õ', "Odieresis": 'Ö', "multiply": '×',
	"Oslash": 'Ø', "Ugrave": 'Ù', "Uacute": 'Ú', "Ucircumflex": 'Û',
	"Udieresis": 'Ü', "Yacute": 'Ý', "Thorn": 'Þ', "germandbls": 'ß',
	"agrave": 'à', "aacute": 'á', "acircumflex": 'â', "atilde": 'ã',
	"adieresis": 'ä', "aring": 'å', "ae": 'æ', "ccedilla": 'ç',
	"egrave": 'è', "eacute": 'é', "ecircumflex": 'ê', "edieresis": 'ë',
	"igrave": 'ì', "iacute": 'í', "icircumflex": 'î', "idieresis": 'ï',
	"eth": 'ð', "ntilde": 'ñ', "ograve": 'ò', "oacute": 'ó',
	"ocircumflex": 'ô', "otilde": 'õ', "odieresis": 'ö', "divide": '÷',
	"oslash": 'ø', "ugrave": 'ù', "uacute": 'ú', "ucircumflex": 'û',
	"udieresis": 'ü', "yacute": 'ý', "thorn": 'þ', "ydieresis": 'ÿ',
	// Common symbols
	"bullet": '•', "endash": '–', "emdash": '—',
	"quotedblleft": '\u201C', "quotedblright": '\u201D',
	"quoteleft": '\u2018', "quoteright": '\u2019',
	"ellipsis": '…', "trademark": '™', "copyright": '©', "registered": '®',
	"degree": '°', "plusminus": '±', "mu": 'µ', "paragraph": '¶',
	"section": '§', "Euro": '€', "sterling": '£', "yen": '¥',
	"cent": '¢', "currency": '¤',
	"fi": '\uFB01', "fl": '\uFB02',
	"dagger": '†', "daggerdbl": '‡',
	"guillemotleft": '«', "guillemotright": '»',
	"guilsinglleft": '‹', "guilsinglright": '›',
	"fraction": '⁄', "minus": '−',
	"periodcentered": '·', "quotesinglbase": '‚', "quotedblbase": '„',
	"perthousand": '‰',
	"Scaron": 'Š', "scaron": 'š', "Zcaron": 'Ž', "zcaron": 'ž',
	"OE": 'Œ', "oe": 'œ', "Ydieresis": 'Ÿ',
	"lozenge": '◊', "dotlessi": 'ı',
	"circumflex": 'ˆ', "tilde": '˜', "macron": '¯',
	"breve": '˘', "dotaccent": '˙', "ring": '˚',
	"cedilla": '¸', "hungarumlaut": '˝', "ogonek": '˛', "caron": 'ˇ',
	"nbspace": '\u00A0', "exclamdown": '¡', "questiondown": '¿',
	"logicalnot": '¬', "radical": '√', "florin": 'ƒ',
	"approxequal": '≈', "Delta": 'Δ', "notequal": '≠',
	"lessequal": '≤', "greaterequal": '≥', "infinity": '∞',
	"integral": '∫', "summation": '∑', "product": '∏',
	"pi": 'π', "Omega": 'Ω', "partialdiff": '∂',
}
