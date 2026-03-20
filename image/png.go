package image

import (
	"bytes"
	"fmt"
	goimage "image"
	"image/png"
)

// LoadPNG loads a PNG image, fully decoding it.
func LoadPNG(data []byte) (*Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("png: %w", err)
	}
	return FromGoImage(img), nil
}

// Ensure goimage and bytes are used.
var _ goimage.Image
var _ = bytes.NewReader
