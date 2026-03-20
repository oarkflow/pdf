package core

import (
	"fmt"
	"io"
)

// PdfIndirectReference represents a PDF indirect reference (N G R).
type PdfIndirectReference struct {
	ObjectNumber     int
	GenerationNumber int
}

func (r PdfIndirectReference) Type() ObjectType { return ObjReference }

func (r PdfIndirectReference) WriteTo(w io.Writer) (int64, error) {
	s := fmt.Sprintf("%d %d R", r.ObjectNumber, r.GenerationNumber)
	n, err := io.WriteString(w, s)
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
	s := fmt.Sprintf("%d %d obj\n", o.Reference.ObjectNumber, o.Reference.GenerationNumber)
	n, err := io.WriteString(w, s)
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
