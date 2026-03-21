package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func createTestPNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

func TestLoadPNG_Basic(t *testing.T) {
	data := createTestPNG(t, 4, 4, color.RGBA{255, 0, 0, 255})
	img, err := LoadPNG(data)
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	if img.Width != 4 || img.Height != 4 {
		t.Errorf("dimensions = %dx%d, want 4x4", img.Width, img.Height)
	}
	if img.ColorSpace != "DeviceRGB" {
		t.Errorf("colorspace = %q, want DeviceRGB", img.ColorSpace)
	}
}

func TestLoadPNG_Gray(t *testing.T) {
	grayImg := image.NewGray(image.Rect(0, 0, 2, 2))
	grayImg.SetGray(0, 0, color.Gray{128})
	var buf bytes.Buffer
	png.Encode(&buf, grayImg)

	img, err := LoadPNG(buf.Bytes())
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	if img.ColorSpace != "DeviceGray" {
		t.Errorf("colorspace = %q, want DeviceGray", img.ColorSpace)
	}
}

func TestLoadPNG_WithAlpha(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(0, 0, 2, 2))
	rgba.Set(0, 0, color.RGBA{255, 0, 0, 128})
	rgba.Set(1, 0, color.RGBA{0, 255, 0, 64})
	rgba.Set(0, 1, color.RGBA{0, 0, 255, 0})
	rgba.Set(1, 1, color.RGBA{255, 255, 255, 255})
	var buf bytes.Buffer
	png.Encode(&buf, rgba)

	img, err := LoadPNG(buf.Bytes())
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	if img.AlphaData == nil {
		t.Error("expected alpha data")
	}
}

func TestLoadPNG_Invalid(t *testing.T) {
	_, err := LoadPNG([]byte("not a png"))
	if err == nil {
		t.Error("expected error for invalid PNG data")
	}
}

func TestLoadPNG_1x1(t *testing.T) {
	data := createTestPNG(t, 1, 1, color.RGBA{0, 0, 0, 255})
	img, err := LoadPNG(data)
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	if img.Width != 1 || img.Height != 1 {
		t.Errorf("dimensions = %dx%d, want 1x1", img.Width, img.Height)
	}
}

func TestLoadPNG_DataSize(t *testing.T) {
	data := createTestPNG(t, 3, 2, color.RGBA{100, 200, 50, 255})
	img, err := LoadPNG(data)
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	// RGB: 3 * 2 * 3 = 18 bytes
	if len(img.Data) != 18 {
		t.Errorf("data len = %d, want 18", len(img.Data))
	}
}

func TestLoadPNG_Filter(t *testing.T) {
	data := createTestPNG(t, 2, 2, color.White)
	img, err := LoadPNG(data)
	if err != nil {
		t.Fatalf("LoadPNG() error = %v", err)
	}
	if img.Filter != "FlateDecode" {
		t.Errorf("filter = %q, want FlateDecode", img.Filter)
	}
}
