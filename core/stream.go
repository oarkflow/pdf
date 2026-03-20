package core

import (
	"bytes"
	"compress/zlib"
	"io"
)

// PdfStream represents a PDF stream object (dictionary + raw byte data).
type PdfStream struct {
	Dictionary *PdfDictionary
	Data       []byte
	compressed bool
}

// NewStream creates a new PdfStream with the given data and an empty dictionary.
func NewStream(data []byte) *PdfStream {
	return &PdfStream{
		Dictionary: NewDictionary(),
		Data:       data,
	}
}

func (s *PdfStream) Type() ObjectType { return ObjStream }

// Compress applies zlib (FlateDecode) compression to the stream data. It is a
// no-op if the stream is already compressed.
func (s *PdfStream) Compress() error {
	if s.compressed {
		return nil
	}
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(s.Data); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	s.Data = buf.Bytes()
	s.compressed = true
	s.Dictionary.Set("Filter", PdfName("FlateDecode"))
	s.Dictionary.Set("Length", PdfInteger(len(s.Data)))
	return nil
}

// WriteTo writes the stream in PDF syntax.
func (s *PdfStream) WriteTo(w io.Writer) (int64, error) {
	// Ensure Length is set.
	s.Dictionary.Set("Length", PdfInteger(len(s.Data)))

	var total int64
	written, err := s.Dictionary.WriteTo(w)
	total += written
	if err != nil {
		return total, err
	}
	n, err := io.WriteString(w, "\nstream\n")
	total += int64(n)
	if err != nil {
		return total, err
	}
	n2, err := w.Write(s.Data)
	total += int64(n2)
	if err != nil {
		return total, err
	}
	n, err = io.WriteString(w, "\nendstream")
	total += int64(n)
	return total, err
}
