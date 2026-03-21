package barcode

import "testing"

func FuzzEncodeQR(f *testing.F) {
	f.Add("Hello")
	f.Add("12345")
	f.Add("")
	f.Add("https://example.com/path?q=1&b=2")

	f.Fuzz(func(t *testing.T, data string) {
		_, _ = EncodeQR(data, ECMedium)
	})
}
