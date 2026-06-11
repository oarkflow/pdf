package pdf

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/oarkflow/pdf/html"
)

const minBenchmarkImageBytes = 512 * 1024

func BenchmarkFromHTMLStreamingWithLargeImage(b *testing.B) {
	imageBytes := benchmarkPNGImage(b)
	htmlContent := benchmarkHTMLWithImage(b, imageBytes, "image/png")
	benchmarkFromHTMLStreamingWithContent(b, htmlContent, len(imageBytes))
}

func BenchmarkFromHTMLStreamingWithLargeJPEGImage(b *testing.B) {
	imageBytes := benchmarkJPEGImage(b)
	htmlContent := benchmarkHTMLWithImage(b, imageBytes, "image/jpeg")
	benchmarkFromHTMLStreamingWithContent(b, htmlContent, len(imageBytes))
}

func BenchmarkCompiledHTMLStreamingWithLargeImage(b *testing.B) {
	imageBytes := benchmarkPNGImage(b)
	htmlContent := benchmarkHTMLWithImage(b, imageBytes, "image/png")
	compiled, err := CompileHTML(htmlContent, benchmarkHTMLOptions())
	if err != nil {
		b.Fatal(err)
	}
	benchmarkCompiledHTMLStreaming(b, compiled, len(imageBytes))
}

func BenchmarkCompiledHTMLStreamingWithLargeJPEGImage(b *testing.B) {
	imageBytes := benchmarkJPEGImage(b)
	htmlContent := benchmarkHTMLWithImage(b, imageBytes, "image/jpeg")
	compiled, err := CompileHTML(htmlContent, benchmarkHTMLOptions())
	if err != nil {
		b.Fatal(err)
	}
	benchmarkCompiledHTMLStreaming(b, compiled, len(imageBytes))
}

func BenchmarkFrozenPDFStreamingWithLargeImage(b *testing.B) {
	imageBytes := benchmarkPNGImage(b)
	htmlContent := benchmarkHTMLWithImage(b, imageBytes, "image/png")
	compiled, err := CompileHTML(htmlContent, benchmarkHTMLOptions())
	if err != nil {
		b.Fatal(err)
	}
	frozen, err := compiled.Freeze()
	if err != nil {
		b.Fatal(err)
	}
	benchmarkFrozenPDFStreaming(b, frozen, len(imageBytes))
}

func BenchmarkFrozenPDFStreamingWithLargeJPEGImage(b *testing.B) {
	imageBytes := benchmarkJPEGImage(b)
	htmlContent := benchmarkHTMLWithImage(b, imageBytes, "image/jpeg")
	compiled, err := CompileHTML(htmlContent, benchmarkHTMLOptions())
	if err != nil {
		b.Fatal(err)
	}
	frozen, err := compiled.Freeze()
	if err != nil {
		b.Fatal(err)
	}
	benchmarkFrozenPDFStreaming(b, frozen, len(imageBytes))
}

func BenchmarkFrozenPDFStreamingWithLargeImageParallel(b *testing.B) {
	imageBytes := benchmarkPNGImage(b)
	htmlContent := benchmarkHTMLWithImage(b, imageBytes, "image/png")
	compiled, err := CompileHTML(htmlContent, benchmarkHTMLOptions())
	if err != nil {
		b.Fatal(err)
	}
	frozen, err := compiled.Freeze()
	if err != nil {
		b.Fatal(err)
	}
	benchmarkFrozenPDFStreamingParallel(b, frozen, len(imageBytes))
}

func BenchmarkFrozenPDFStreamingWithLargeJPEGImageParallel(b *testing.B) {
	imageBytes := benchmarkJPEGImage(b)
	htmlContent := benchmarkHTMLWithImage(b, imageBytes, "image/jpeg")
	compiled, err := CompileHTML(htmlContent, benchmarkHTMLOptions())
	if err != nil {
		b.Fatal(err)
	}
	frozen, err := compiled.Freeze()
	if err != nil {
		b.Fatal(err)
	}
	benchmarkFrozenPDFStreamingParallel(b, frozen, len(imageBytes))
}

func benchmarkHTMLWithImage(b *testing.B, imageBytes []byte, mediaType string) string {
	b.Helper()
	if len(imageBytes) < minBenchmarkImageBytes {
		b.Fatalf("benchmark image is %d bytes, want at least %d", len(imageBytes), minBenchmarkImageBytes)
	}

	imageURI := "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(imageBytes)
	return `<!doctype html>
<html>
<head>
  <title>Large image benchmark</title>
  <style>
    body { font-family: Helvetica, sans-serif; margin: 0; }
    h1 { font-size: 22pt; margin: 0 0 16px; }
    p { font-size: 10pt; line-height: 1.4; margin: 0 0 12px; }
    img { display: block; width: 420px; height: 420px; object-fit: cover; }
  </style>
</head>
<body>
  <h1>HTML to PDF Image Benchmark</h1>
  <p>This benchmark measures full HTML conversion, layout, image decoding, and PDF serialization.</p>
  <img src="` + imageURI + `" alt="Generated benchmark image">
</body>
</html>`
}

func benchmarkHTMLOptions() html.Options {
	return html.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Helvetica",
		PageSize:          [2]float64{595.28, 841.89},
		Margins:           [4]float64{40, 40, 40, 40},
	}
}

func benchmarkFromHTMLStreamingWithContent(b *testing.B, htmlContent string, imageBytes int) {
	b.Helper()
	var out bytes.Buffer
	b.ReportAllocs()
	b.SetBytes(int64(imageBytes))
	b.ResetTimer()
	b.ReportMetric(float64(imageBytes)/1024, "KiB_image")

	for i := 0; i < b.N; i++ {
		out.Reset()
		if err := FromHTMLStreaming(htmlContent, &out, benchmarkHTMLOptions()); err != nil {
			b.Fatal(err)
		}
		if out.Len() == 0 {
			b.Fatal("empty PDF output")
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pdfs/s")
}

func benchmarkCompiledHTMLStreaming(b *testing.B, compiled *CompiledHTML, imageBytes int) {
	b.Helper()
	var out bytes.Buffer
	b.ReportAllocs()
	b.SetBytes(int64(imageBytes))
	b.ResetTimer()
	b.ReportMetric(float64(imageBytes)/1024, "KiB_image")

	for i := 0; i < b.N; i++ {
		out.Reset()
		if err := compiled.WriteStreamingTo(&out); err != nil {
			b.Fatal(err)
		}
		if out.Len() == 0 {
			b.Fatal("empty PDF output")
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pdfs/s")
}

func benchmarkFrozenPDFStreaming(b *testing.B, frozen *FrozenPDF, imageBytes int) {
	b.Helper()
	var out bytes.Buffer
	b.ReportAllocs()
	b.SetBytes(int64(imageBytes))
	b.ResetTimer()
	b.ReportMetric(float64(imageBytes)/1024, "KiB_image")

	for i := 0; i < b.N; i++ {
		out.Reset()
		if err := frozen.WriteStreamingTo(&out); err != nil {
			b.Fatal(err)
		}
		if out.Len() == 0 {
			b.Fatal("empty PDF output")
		}
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pdfs/s")
}

func benchmarkFrozenPDFStreamingParallel(b *testing.B, frozen *FrozenPDF, imageBytes int) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(imageBytes))
	b.ResetTimer()
	b.ReportMetric(float64(imageBytes)/1024, "KiB_image")

	b.RunParallel(func(pb *testing.PB) {
		var out bytes.Buffer
		for pb.Next() {
			out.Reset()
			if err := frozen.WriteStreamingTo(&out); err != nil {
				b.Fatal(err)
			}
			if out.Len() == 0 {
				b.Fatal("empty PDF output")
			}
		}
	})

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "pdfs/s")
}

func benchmarkPNGImage(tb testing.TB) []byte {
	tb.Helper()

	img := benchmarkNRGBAImage()
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.NoCompression}
	if err := encoder.Encode(&buf, img); err != nil {
		tb.Fatalf("encoding benchmark image: %v", err)
	}
	return buf.Bytes()
}

func benchmarkJPEGImage(tb testing.TB) []byte {
	tb.Helper()

	img := benchmarkNRGBAImage()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		tb.Fatalf("encoding benchmark image: %v", err)
	}
	return buf.Bytes()
}

func benchmarkNRGBAImage() *image.NRGBA {
	const size = 768
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	state := uint32(0x12345678)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			state = state*1664525 + 1013904223 + uint32(x*31+y*17)
			offset := img.PixOffset(x, y)
			img.Pix[offset] = byte(state >> 24)
			img.Pix[offset+1] = byte(state >> 16)
			img.Pix[offset+2] = byte(state >> 8)
			img.Pix[offset+3] = 255
		}
	}
	return img
}
