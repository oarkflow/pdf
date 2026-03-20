package image

import (
	"bytes"
	"fmt"
	goimage "image"

	_ "golang.org/x/image/tiff"
)

// LoadTIFF loads a TIFF image.
func LoadTIFF(data []byte) (*Image, error) {
	img, _, err := goimage.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("tiff: %w", err)
	}
	return FromGoImage(img), nil
}
