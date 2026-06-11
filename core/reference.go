package core

import (
	"io"
	"strconv"
)

// PdfIndirectReference represents a PDF indirect reference (N G R).
type PdfIndirectReference struct {
	ObjectNumber     int
	GenerationNumber int
}

func (r PdfIndirectReference) Type() ObjectType { return ObjReference }

func (r PdfIndirectReference) WriteTo(w io.Writer) (int64, error) {
	var buf [32]byte
	b := strconv.AppendInt(buf[:0], int64(r.ObjectNumber), 10)
	b = append(b, ' ')
	b = strconv.AppendInt(b, int64(r.GenerationNumber), 10)
	b = append(b, ' ', 'R')
	n, err := w.Write(b)
	return int64(n), err
}

// PdfIndirectObject wraps a PdfObject with an indirect reference.
type PdfIndirectObject struct {
	Reference PdfIndirectReference
	Object    PdfObject
}

func (o *PdfIndirectObject) Type() ObjectType {
	if o.Object != nil {
		return o.Object.Type()
	}
	return ObjNull
}

func (o *PdfIndirectObject) WriteTo(w io.Writer) (int64, error) {
	var total int64
	var buf [40]byte
	b := strconv.AppendInt(buf[:0], int64(o.Reference.ObjectNumber), 10)
	b = append(b, ' ')
	b = strconv.AppendInt(b, int64(o.Reference.GenerationNumber), 10)
	b = append(b, ' ', 'o', 'b', 'j', '\n')
	n, err := w.Write(b)
	total += int64(n)
	if err != nil {
		return total, err
	}
	if o.Object != nil {
		written, err := o.Object.WriteTo(w)
		total += written
		if err != nil {
			return total, err
		}
	}
	n, err = io.WriteString(w, "\nendobj\n")
	total += int64(n)
	return total, err
}
