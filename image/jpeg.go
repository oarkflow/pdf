package image

import (
	"bytes"
	"fmt"
	goimage "image"
	"image/color"
	_ "image/jpeg"
)

// LoadJPEG loads a JPEG image for DCTDecode passthrough.
// It reads dimensions and colorspace from the JPEG data without fully decoding.
func LoadJPEG(data []byte) (*Image, error) {
	cfg, _, err := goimage.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("jpeg: %w", err)
	}

	cs := "DeviceRGB"
	if cfg.ColorModel == color.GrayModel || cfg.ColorModel == color.Gray16Model {
		cs = "DeviceGray"
	} else if cfg.ColorModel == color.CMYKModel {
		cs = "DeviceCMYK"
	}

	return &Image{
		Width:       cfg.Width,
		Height:      cfg.Height,
		ColorSpace:  cs,
		BitsPerComp: 8,
		Filter:      "DCTDecode",
		RawStream:   data,
	}, nil
}
