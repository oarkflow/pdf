package reader

import (
	"fmt"
	"strings"
)

// FontInfo holds parsed font data for text extraction.
// This is the exported version of fontInfo for use by external packages.
type FontInfo struct {
	Name        string
	Encoding    string
	ToUnicode   map[uint16]rune
	Differences map[byte]string // glyph name remapping from /Encoding /Differences
	Widths      map[int]float64
	BaseFont    string
	IsType0     bool
}

// ParseFontInfo extracts font info from a font dictionary.
// This is the exported version for use by external packages.
func ParseFontInfo(resolver *Resolver, fontDict map[string]interface{}) (*FontInfo, error) {
	fi, err := parseFontInfo(resolver, fontDict)
	if err != nil {
		return nil, err
	}
	return &FontInfo{
		Name:        fi.name,
		Encoding:    fi.encoding,
		ToUnicode:   fi.toUnicode,
		Differences: fi.differences,
		Widths:      fi.widths,
		BaseFont:    fi.baseFont,
		IsType0:     fi.isType0,
	}, nil
}

// PageDict returns the raw page dictionary for the 0-indexed page.
// This provides access to annotations and other page-level data.
func (r *Reader) PageDict(n int) (map[string]interface{}, error) {
	if n < 0 || n >= len(r.pages) {
		return nil, fmt.Errorf("page %d out of range [0, %d)", n, len(r.pages))
	}
	return r.pages[n], nil
}

// DecodeString decodes a PDF string using the given font info.
func DecodeString(fi *FontInfo, s string) string {
	if fi == nil {
		return s
	}
	if fi.IsType0 && fi.ToUnicode != nil {
		return decodeType0Exported(fi, s)
	}
	if fi.ToUnicode != nil {
		return decodeWithToUnicode(fi, s)
	}
	// Try /Differences first, then fall back to named encoding.
	if fi.Differences != nil {
		return decodeWithDifferences(fi, s)
	}
	return decodeWithEncoding(s, fi.Encoding)
}

func decodeWithDifferences(fi *FontInfo, s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if glyphName, ok := fi.Differences[b]; ok {
			if r, ok := GlyphNameToUnicode[glyphName]; ok {
				buf.WriteRune(r)
				continue
			}
			// If glyph name is a single character name like "A", use it directly.
			if len(glyphName) == 1 {
				buf.WriteByte(glyphName[0])
				continue
			}
		}
		// Fall back to base encoding.
		if fi.Encoding != "" {
			r := decodeByteWithEncoding(b, fi.Encoding)
			buf.WriteRune(r)
		} else {
			buf.WriteByte(b)
		}
	}
	return buf.String()
}

func decodeByteWithEncoding(b byte, encoding string) rune {
	switch encoding {
	case "/WinAnsiEncoding":
		return winAnsiToUnicode[b]
	case "/MacRomanEncoding":
		if b < 128 {
			return rune(b)
		}
		r := macRomanToUnicode[b-128]
		if r == 0 {
			return rune(b)
		}
		return r
	}
	return rune(b)
}

func decodeType0Exported(fi *FontInfo, s string) string {
	var buf []byte
	for i := 0; i+1 < len(s); i += 2 {
		code := uint16(s[i])<<8 | uint16(s[i+1])
		if r, ok := fi.ToUnicode[code]; ok {
			buf = append(buf, []byte(string(r))...)
		} else {
			buf = append(buf, []byte(string(rune(code)))...)
		}
	}
	return string(buf)
}

func decodeWithToUnicode(fi *FontInfo, s string) string {
	var buf []byte
	for i := 0; i < len(s); i++ {
		code := uint16(s[i])
		if r, ok := fi.ToUnicode[code]; ok {
			buf = append(buf, []byte(string(r))...)
		} else {
			buf = append(buf, s[i])
		}
	}
	return string(buf)
}

// GetDict resolves a value to a dictionary. Exported for converter use.
func GetDict(resolver *Resolver, v interface{}) (map[string]interface{}, error) {
	return getDict(resolver, v)
}

// GetStreamData resolves a value to decompressed stream data. Exported for converter use.
func GetStreamData(resolver *Resolver, v interface{}) ([]byte, error) {
	return getStreamData(resolver, v)
}

// OperandFloat converts a PDF operand to float64. Exported for converter use.
func OperandFloat(v interface{}) float64 {
	return operandFloat(v)
}

// GetInt extracts an int64 from a dictionary. Exported for converter use.
func GetInt(dict map[string]interface{}, key string) (int64, bool) {
	return getInt(dict, key)
}

// ToFloat converts a PDF numeric value to float64. Exported for converter use.
func ToFloat(v interface{}) float64 {
	return toFloat(v)
}
