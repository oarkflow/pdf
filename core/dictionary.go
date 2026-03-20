package core

import (
	"io"
)

// DictEntry is a single key-value pair in an ordered PDF dictionary.
type DictEntry struct {
	Key   string
	Value PdfObject
}

// PdfDictionary is an ordered dictionary that preserves insertion order for
// deterministic PDF output.
type PdfDictionary struct {
	entries []DictEntry
}

// NewDictionary creates an empty PdfDictionary.
func NewDictionary() *PdfDictionary {
	return &PdfDictionary{}
}

func (d *PdfDictionary) Type() ObjectType { return ObjDictionary }

// Set adds or updates a key-value pair. If the key already exists its value is
// replaced in-place, preserving order.
func (d *PdfDictionary) Set(key string, value PdfObject) {
	for i := range d.entries {
		if d.entries[i].Key == key {
			d.entries[i].Value = value
			return
		}
	}
	d.entries = append(d.entries, DictEntry{Key: key, Value: value})
}

// Get returns the value for key, or nil if not present.
func (d *PdfDictionary) Get(key string) PdfObject {
	for _, e := range d.entries {
		if e.Key == key {
			return e.Value
		}
	}
	return nil
}

// Has reports whether key is present.
func (d *PdfDictionary) Has(key string) bool {
	for _, e := range d.entries {
		if e.Key == key {
			return true
		}
	}
	return false
}

// Remove deletes a key from the dictionary.
func (d *PdfDictionary) Remove(key string) {
	for i, e := range d.entries {
		if e.Key == key {
			d.entries = append(d.entries[:i], d.entries[i+1:]...)
			return
		}
	}
}

// Keys returns the keys in insertion order.
func (d *PdfDictionary) Keys() []string {
	keys := make([]string, len(d.entries))
	for i, e := range d.entries {
		keys[i] = e.Key
	}
	return keys
}

// Len returns the number of entries.
func (d *PdfDictionary) Len() int {
	return len(d.entries)
}

// WriteTo writes the dictionary in PDF syntax: << /Key1 value1 /Key2 value2 >>
func (d *PdfDictionary) WriteTo(w io.Writer) (int64, error) {
	var total int64
	n, err := io.WriteString(w, "<<")
	total += int64(n)
	if err != nil {
		return total, err
	}
	for _, e := range d.entries {
		n, err = io.WriteString(w, " ")
		total += int64(n)
		if err != nil {
			return total, err
		}
		name := PdfName(e.Key)
		written, err := name.WriteTo(w)
		total += written
		if err != nil {
			return total, err
		}
		n, err = io.WriteString(w, " ")
		total += int64(n)
		if err != nil {
			return total, err
		}
		written, err = e.Value.WriteTo(w)
		total += written
		if err != nil {
			return total, err
		}
	}
	n, err = io.WriteString(w, " >>")
	total += int64(n)
	return total, err
}
