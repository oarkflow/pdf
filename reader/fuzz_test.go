package reader

import "testing"

func FuzzOpen(f *testing.F) {
	f.Add([]byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [] /Count 0 >>\nendobj\nxref\n0 3\n0000000000 65535 f \r\n0000000009 00000 n \r\n0000000058 00000 n \r\ntrailer\n<< /Size 3 /Root 1 0 R >>\nstartxref\n109\n%%EOF\n"))
	f.Add([]byte("%PDF-1.0\n"))
	f.Add([]byte(""))
	f.Add([]byte("not a pdf at all"))

	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := Open(data)
		if err != nil {
			return
		}
		_ = r.NumPages()
		if r.NumPages() > 0 {
			_, _ = r.Page(0)
			_, _ = r.ExtractText(0)
		}
	})
}

func FuzzTokenizer(f *testing.F) {
	f.Add([]byte("123 4.56 (hello) <4E6F> /Name [1 2] << /K /V >>"))
	f.Add([]byte(""))
	f.Add([]byte("((()))"))
	f.Add([]byte("<<<>>>"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tok := NewTokenizer(data)
		for i := 0; i < 1000; i++ {
			_, err := tok.Next()
			if err != nil {
				break
			}
		}
	})
}

func FuzzResolver(f *testing.F) {
	f.Add([]byte("%PDF-1.7\nxref\n0 1\n0000000000 65535 f \r\ntrailer\n<< /Size 1 >>\nstartxref\n10\n%%EOF\n"))
	f.Add([]byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\nxref\n0 2\n0000000000 65535 f \r\n0000000009 00000 n \r\ntrailer\n<< /Size 2 /Root 1 0 R >>\nstartxref\n52\n%%EOF\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		r, err := NewResolver(data)
		if err != nil {
			return
		}
		_, _ = r.Trailer()
		for i := 1; i <= 5; i++ {
			_, _ = r.ResolveObject(i)
		}
	})
}
