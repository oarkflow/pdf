package core

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ObjectType enumerates the PDF object types per ISO 32000 §7.3.
type ObjectType int

const (
	ObjBoolean    ObjectType = iota
	ObjInteger
	ObjNumber
	ObjString
	ObjHexString
	ObjName
	ObjArray
	ObjDictionary
	ObjStream
	ObjNull
	ObjReference
)

// PdfObject is the interface satisfied by all PDF object types.
type PdfObject interface {
	WriteTo(w io.Writer) (int64, error)
	Type() ObjectType
}

// ---------------------------------------------------------------------------
// PdfBoolean
// ---------------------------------------------------------------------------

type PdfBoolean bool

func (b PdfBoolean) Type() ObjectType { return ObjBoolean }

func (b PdfBoolean) WriteTo(w io.Writer) (int64, error) {
	var s string
	if b {
		s = "true"
	} else {
		s = "false"
	}
	n, err := io.WriteString(w, s)
	return int64(n), err
}

// ---------------------------------------------------------------------------
// PdfInteger
// ---------------------------------------------------------------------------

type PdfInteger int64

func (i PdfInteger) Type() ObjectType { return ObjInteger }

func (i PdfInteger) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, strconv.FormatInt(int64(i), 10))
	return int64(n), err
}

// ---------------------------------------------------------------------------
// PdfNumber
// ---------------------------------------------------------------------------

type PdfNumber float64

func (f PdfNumber) Type() ObjectType { return ObjNumber }

func (f PdfNumber) WriteTo(w io.Writer) (int64, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	// Ensure there is a decimal point so it reads as a real number.
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	// Trim unnecessary trailing zeros but keep at least one digit after the dot.
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		s = strings.TrimRight(s, "0")
		if s[len(s)-1] == '.' {
			s += "0"
		}
	}
	n, err := io.WriteString(w, s)
	return int64(n), err
}

// ---------------------------------------------------------------------------
// PdfString – literal string (…)
// ---------------------------------------------------------------------------

type PdfString string

func (s PdfString) Type() ObjectType { return ObjString }

func (s PdfString) WriteTo(w io.Writer) (int64, error) {
	var buf strings.Builder
	buf.WriteByte('(')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '\\':
			buf.WriteString("\\\\")
		case '(':
			buf.WriteString("\\(")
		case ')':
			buf.WriteString("\\)")
		case '\r':
			buf.WriteString("\\r")
		case '\n':
			buf.WriteString("\\n")
		case '\t':
			buf.WriteString("\\t")
		case '\b':
			buf.WriteString("\\b")
		case '\f':
			buf.WriteString("\\f")
		default:
			if ch < 0x20 || ch > 0x7e {
				fmt.Fprintf(&buf, "\\%03o", ch)
			} else {
				buf.WriteByte(ch)
			}
		}
	}
	buf.WriteByte(')')
	n, err := io.WriteString(w, buf.String())
	return int64(n), err
}

// ---------------------------------------------------------------------------
// PdfHexString
// ---------------------------------------------------------------------------

type PdfHexString []byte

func (h PdfHexString) Type() ObjectType { return ObjHexString }

func (h PdfHexString) WriteTo(w io.Writer) (int64, error) {
	var buf strings.Builder
	buf.WriteByte('<')
	for _, b := range h {
		fmt.Fprintf(&buf, "%02X", b)
	}
	buf.WriteByte('>')
	n, err := io.WriteString(w, buf.String())
	return int64(n), err
}

// ---------------------------------------------------------------------------
// PdfName
// ---------------------------------------------------------------------------

type PdfName string

func (nm PdfName) Type() ObjectType { return ObjName }

// nameNeedsEscape returns true if the byte must be #-encoded in a PDF name.
func nameNeedsEscape(c byte) bool {
	if c < 33 || c > 126 {
		return true
	}
	switch c {
	case '#', '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}
	return false
}

func (nm PdfName) WriteTo(w io.Writer) (int64, error) {
	var buf strings.Builder
	buf.WriteByte('/')
	for i := 0; i < len(nm); i++ {
		c := nm[i]
		if nameNeedsEscape(c) {
			fmt.Fprintf(&buf, "#%02X", c)
		} else {
			buf.WriteByte(c)
		}
	}
	n, err := io.WriteString(w, buf.String())
	return int64(n), err
}

// ---------------------------------------------------------------------------
// PdfArray
// ---------------------------------------------------------------------------

type PdfArray []PdfObject

func (a PdfArray) Type() ObjectType { return ObjArray }

func (a PdfArray) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n, err := io.WriteString(w, "[")
	total += int64(n)
	if err != nil {
		return total, err
	}
	for i, obj := range a {
		if i > 0 {
			n, err = io.WriteString(w, " ")
			total += int64(n)
			if err != nil {
				return total, err
			}
		}
		written, err := obj.WriteTo(w)
		total += written
		if err != nil {
			return total, err
		}
	}
	n, err = io.WriteString(w, "]")
	total += int64(n)
	return total, err
}

// ---------------------------------------------------------------------------
// PdfNull
// ---------------------------------------------------------------------------

type PdfNull struct{}

func (PdfNull) Type() ObjectType { return ObjNull }

func (PdfNull) WriteTo(w io.Writer) (int64, error) {
	n, err := io.WriteString(w, "null")
	return int64(n), err
}
