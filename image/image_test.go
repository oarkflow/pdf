package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

var tinyWebP = []byte{
	0x52, 0x49, 0x46, 0x46, 0x3c, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
	0x56, 0x50, 0x38, 0x20, 0x30, 0x00, 0x00, 0x00, 0xd0, 0x01, 0x00, 0x9d,
	0x01, 0x2a, 0x02, 0x00, 0x02, 0x00, 0x02, 0x00, 0x34, 0x25, 0xa0, 0x02,
	0x74, 0xba, 0x01, 0xf8, 0x00, 0x03, 0xb0, 0x00, 0xfe, 0xf0, 0xe8, 0xf7,
	0xff, 0x20, 0xb9, 0x61, 0x75, 0xc8, 0xd7, 0xff, 0x20, 0x3f, 0xe4, 0x07,
	0xfc, 0x80, 0xff, 0xf8, 0xf2, 0x00, 0x00, 0x00,
}

func TestLoad_TooShort(t *testing.T) {
	_, err := Load([]byte{0, 1})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestLoad_UnsupportedFormat(t *testing.T) {
	_, err := Load([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestLoad_JPEGMagic(t *testing.T) {
	// Valid JPEG magic but invalid JPEG data
	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0, 0, 0}
	_, err := Load(data)
	if err == nil {
		t.Error("expected error for invalid JPEG")
	}
}

func TestLoad_PNGMagic(t *testing.T) {
	// Valid PNG magic but invalid PNG data
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0, 0, 0, 0}
	_, err := Load(data)
	if err == nil {
		t.Error("expected error for invalid PNG")
	}
}

func TestLoad_PNG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	src.Set(1, 0, color.RGBA{0, 255, 0, 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}

	result, err := Load(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 2 || result.Height != 1 {
		t.Fatalf("dimensions = %dx%d, want 2x1", result.Width, result.Height)
	}
}

func TestLoad_JPEG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	src.Set(1, 0, color.RGBA{0, 255, 0, 255})
	src.Set(2, 0, color.RGBA{0, 0, 255, 255})

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, nil); err != nil {
		t.Fatal(err)
	}

	result, err := Load(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 3 || result.Height != 2 {
		t.Fatalf("dimensions = %dx%d, want 3x2", result.Width, result.Height)
	}
	if result.Filter != "DCTDecode" {
		t.Fatalf("filter = %q, want DCTDecode", result.Filter)
	}
}

func TestLoad_WebP(t *testing.T) {
	result, err := Load(tinyWebP)
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 2 || result.Height != 2 {
		t.Fatalf("dimensions = %dx%d, want 2x2", result.Width, result.Height)
	}
}

func TestFromGoImage_RGB(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 255})
	img.Set(0, 1, color.RGBA{0, 0, 255, 255})
	img.Set(1, 1, color.RGBA{255, 255, 255, 255})

	result := FromGoImage(img)
	if result.Width != 2 || result.Height != 2 {
		t.Errorf("dimensions = %dx%d, want 2x2", result.Width, result.Height)
	}
	if result.ColorSpace != "DeviceRGB" {
		t.Errorf("colorspace = %q, want DeviceRGB", result.ColorSpace)
	}
	if result.BitsPerComp != 8 {
		t.Errorf("bits = %d, want 8", result.BitsPerComp)
	}
	if len(result.Data) != 2*2*3 {
		t.Errorf("data len = %d, want %d", len(result.Data), 2*2*3)
	}
}

func TestFromGoImage_Gray(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 3, 3))
	img.SetGray(1, 1, color.Gray{Y: 128})

	result := FromGoImage(img)
	if result.ColorSpace != "DeviceGray" {
		t.Errorf("colorspace = %q, want DeviceGray", result.ColorSpace)
	}
	if len(result.Data) != 9 {
		t.Errorf("data len = %d, want 9", len(result.Data))
	}
}

func TestFromGoImage_WithAlpha(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 128})

	result := FromGoImage(img)
	if result.AlphaData == nil {
		t.Error("expected alpha data for semi-transparent image")
	}
}

func TestFromGoImage_FullyOpaque(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 255})

	result := FromGoImage(img)
	if result.AlphaData != nil {
		t.Error("no alpha data expected for fully opaque image")
	}
}

func TestImage_BuildXObject(t *testing.T) {
	img := &Image{
		Width:       10,
		Height:      10,
		ColorSpace:  "DeviceRGB",
		BitsPerComp: 8,
		Data:        make([]byte, 300),
		Filter:      "FlateDecode",
	}
	main, smask, err := img.BuildXObject(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if main.Object == nil {
		t.Error("main object is nil")
	}
	if smask != nil {
		t.Error("smask should be nil when no alpha")
	}
}

func TestImage_BuildXObject_WithAlpha(t *testing.T) {
	img := &Image{
		Width:       2,
		Height:      2,
		ColorSpace:  "DeviceRGB",
		BitsPerComp: 8,
		Data:        make([]byte, 12),
		AlphaData:   make([]byte, 4),
		Filter:      "FlateDecode",
	}
	_, smask, err := img.BuildXObject(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if smask == nil {
		t.Error("smask should not be nil when alpha present")
	}
}
