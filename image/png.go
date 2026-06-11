package image

import (
	"bytes"
	"encoding/binary"
	"fmt"
	goimage "image"
	"image/png"

	"github.com/oarkflow/pdf/core"
)

// LoadPNG loads a PNG image, fully decoding it.
func LoadPNG(data []byte) (*Image, error) {
	if img, ok, err := loadPNGPassthrough(data); err != nil {
		return nil, err
	} else if ok {
		return img, nil
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("png: %w", err)
	}
	return FromGoImage(img), nil
}

func loadPNGPassthrough(data []byte) (*Image, bool, error) {
	const pngHeaderLen = 8
	if len(data) < pngHeaderLen || !bytes.Equal(data[:pngHeaderLen], []byte("\x89PNG\r\n\x1a\n")) {
		return nil, false, nil
	}

	var width, height uint32
	var bitDepth, colorType, interlace byte
	var idat bytes.Buffer

	for offset := pngHeaderLen; offset < len(data); {
		if offset+12 > len(data) {
			return nil, false, fmt.Errorf("png: truncated chunk")
		}
		length := binary.BigEndian.Uint32(data[offset : offset+4])
		chunkType := string(data[offset+4 : offset+8])
		chunkStart := offset + 8
		chunkEnd := chunkStart + int(length)
		if chunkEnd+4 > len(data) {
			return nil, false, fmt.Errorf("png: truncated %s chunk", chunkType)
		}
		chunkData := data[chunkStart:chunkEnd]

		switch chunkType {
		case "IHDR":
			if length != 13 {
				return nil, false, fmt.Errorf("png: invalid IHDR length")
			}
			width = binary.BigEndian.Uint32(chunkData[0:4])
			height = binary.BigEndian.Uint32(chunkData[4:8])
			bitDepth = chunkData[8]
			colorType = chunkData[9]
			interlace = chunkData[12]
		case "IDAT":
			idat.Write(chunkData)
		case "IEND":
			offset = len(data)
			continue
		}

		offset = chunkEnd + 4
	}

	if width == 0 || height == 0 || idat.Len() == 0 {
		return nil, false, nil
	}
	if bitDepth != 8 || interlace != 0 {
		return nil, false, nil
	}

	colorSpace := ""
	colors := 0
	switch colorType {
	case 0:
		colorSpace = "DeviceGray"
		colors = 1
	case 2:
		colorSpace = "DeviceRGB"
		colors = 3
	default:
		return nil, false, nil
	}

	decodeParms := core.NewDictionary()
	decodeParms.Set("Predictor", core.PdfInteger(15))
	decodeParms.Set("Colors", core.PdfInteger(colors))
	decodeParms.Set("BitsPerComponent", core.PdfInteger(8))
	decodeParms.Set("Columns", core.PdfInteger(width))

	return &Image{
		Width:       int(width),
		Height:      int(height),
		ColorSpace:  colorSpace,
		BitsPerComp: 8,
		Filter:      "FlateDecode",
		RawStream:   idat.Bytes(),
		DecodeParms: decodeParms,
	}, true, nil
}

// Ensure goimage and bytes are used.
var _ goimage.Image
var _ = bytes.NewReader
