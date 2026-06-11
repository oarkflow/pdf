package image

import (
	"bytes"
	"fmt"
	"hash/fnv"
	goimage "image"
	"image/color"
	_ "image/gif"
	"sync"

	"github.com/oarkflow/pdf/core"
	"golang.org/x/image/webp"
)

// Image holds decoded image data ready for embedding in a PDF.
type Image struct {
	Width       int
	Height      int
	ColorSpace  string // DeviceRGB, DeviceGray, DeviceCMYK
	BitsPerComp int
	Data        []byte // raw pixel data
	AlphaData   []byte // optional alpha channel
	Filter      string // FlateDecode, DCTDecode
	RawStream   []byte // for JPEG passthrough
	DecodeParms *core.PdfDictionary
	flateOnce   sync.Once
	flateData   []byte
	flateErr    error
	alphaOnce   sync.Once
	alphaFlate  []byte
	alphaErr    error
}

// FlattenAlphaOnWhite returns an image with any alpha channel composited over
// white. It returns img unchanged when there is no alpha data to flatten.
func (img *Image) FlattenAlphaOnWhite() *Image {
	if img == nil || len(img.AlphaData) == 0 || img.ColorSpace != "DeviceRGB" || len(img.Data) != img.Width*img.Height*3 || len(img.AlphaData) != img.Width*img.Height {
		return img
	}
	data := make([]byte, len(img.Data))
	for i, a := range img.AlphaData {
		src := i * 3
		alpha := int(a)
		inv := 255 - alpha
		data[src] = byte((int(img.Data[src])*alpha + 255*inv + 127) / 255)
		data[src+1] = byte((int(img.Data[src+1])*alpha + 255*inv + 127) / 255)
		data[src+2] = byte((int(img.Data[src+2])*alpha + 255*inv + 127) / 255)
	}
	return &Image{
		Width:       img.Width,
		Height:      img.Height,
		ColorSpace:  img.ColorSpace,
		BitsPerComp: img.BitsPerComp,
		Data:        data,
		Filter:      "FlateDecode",
	}
}

// FromGoImage converts a Go image.Image to our Image struct.
func FromGoImage(img goimage.Image) *Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Check if grayscale.
	switch img.(type) {
	case *goimage.Gray:
		g := img.(*goimage.Gray)
		data := make([]byte, w*h)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				idx := (y-bounds.Min.Y)*w + (x - bounds.Min.X)
				data[idx] = g.GrayAt(x, y).Y
			}
		}
		return &Image{
			Width:       w,
			Height:      h,
			ColorSpace:  "DeviceGray",
			BitsPerComp: 8,
			Data:        data,
			Filter:      "FlateDecode",
		}
	}

	// Convert to RGB + optional alpha.
	rgb := make([]byte, w*h*3)
	var alpha []byte
	hasAlpha := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			idx := (y-bounds.Min.Y)*w + (x - bounds.Min.X)
			// Convert from pre-multiplied to straight if needed.
			if a == 0 {
				rgb[idx*3] = 0
				rgb[idx*3+1] = 0
				rgb[idx*3+2] = 0
			} else {
				rgb[idx*3] = uint8(r >> 8)
				rgb[idx*3+1] = uint8(g >> 8)
				rgb[idx*3+2] = uint8(b >> 8)
			}
			a8 := uint8(a >> 8)
			if a8 != 255 {
				hasAlpha = true
			}
			if alpha == nil {
				alpha = make([]byte, w*h)
				// Fill already-processed pixels with 255.
				for i := 0; i < idx; i++ {
					alpha[i] = 255
				}
			}
			if alpha != nil {
				alpha[idx] = a8
			}
		}
	}

	result := &Image{
		Width:       w,
		Height:      h,
		ColorSpace:  "DeviceRGB",
		BitsPerComp: 8,
		Data:        rgb,
		Filter:      "FlateDecode",
	}
	if hasAlpha && alpha != nil {
		result.AlphaData = alpha
	}
	return result
}

// BuildXObject creates a PDF image XObject (and optional SMask) from this Image.
// Returns the main image object and optionally a soft mask object.
func (img *Image) BuildXObject(objNum int, smaskObjNum int) (main core.PdfIndirectObject, smask *core.PdfIndirectObject, err error) {
	var stream *core.PdfStream
	if img.RawStream != nil && img.Filter != "" {
		stream = core.NewStream(img.RawStream)
		stream.Dictionary.Set("Filter", core.PdfName(img.Filter))
		stream.Dictionary.Set("Length", core.PdfInteger(len(img.RawStream)))
	} else if img.Filter == "FlateDecode" {
		data, compressErr := img.CompressedData()
		if compressErr != nil {
			return core.PdfIndirectObject{}, nil, compressErr
		}
		stream = core.NewStream(data)
		stream.Dictionary.Set("Filter", core.PdfName("FlateDecode"))
		stream.Dictionary.Set("Length", core.PdfInteger(len(data)))
	} else {
		stream = core.NewStream(img.Data)
		if err := stream.Compress(); err != nil {
			return core.PdfIndirectObject{}, nil, fmt.Errorf("compressing image data: %w", err)
		}
	}

	stream.Dictionary.Set("Type", core.PdfName("XObject"))
	stream.Dictionary.Set("Subtype", core.PdfName("Image"))
	stream.Dictionary.Set("Width", core.PdfInteger(img.Width))
	stream.Dictionary.Set("Height", core.PdfInteger(img.Height))
	stream.Dictionary.Set("ColorSpace", core.PdfName(img.ColorSpace))
	stream.Dictionary.Set("BitsPerComponent", core.PdfInteger(img.BitsPerComp))
	if img.DecodeParms != nil {
		stream.Dictionary.Set("DecodeParms", img.DecodeParms)
	}

	// Build SMask if alpha data is present.
	if len(img.AlphaData) > 0 {
		alphaData, compressErr := img.CompressedAlphaData()
		if compressErr != nil {
			return core.PdfIndirectObject{}, nil, compressErr
		}
		smaskStream := core.NewStream(alphaData)
		smaskStream.Dictionary.Set("Filter", core.PdfName("FlateDecode"))
		smaskStream.Dictionary.Set("Length", core.PdfInteger(len(alphaData)))
		smaskStream.Dictionary.Set("Type", core.PdfName("XObject"))
		smaskStream.Dictionary.Set("Subtype", core.PdfName("Image"))
		smaskStream.Dictionary.Set("Width", core.PdfInteger(img.Width))
		smaskStream.Dictionary.Set("Height", core.PdfInteger(img.Height))
		smaskStream.Dictionary.Set("ColorSpace", core.PdfName("DeviceGray"))
		smaskStream.Dictionary.Set("BitsPerComponent", core.PdfInteger(8))

		smaskObj := &core.PdfIndirectObject{
			Reference: core.PdfIndirectReference{ObjectNumber: smaskObjNum, GenerationNumber: 0},
			Object:    smaskStream,
		}
		smask = smaskObj

		stream.Dictionary.Set("SMask", core.PdfIndirectReference{ObjectNumber: smaskObjNum, GenerationNumber: 0})
	}

	main = core.PdfIndirectObject{
		Reference: core.PdfIndirectReference{ObjectNumber: objNum, GenerationNumber: 0},
		Object:    stream,
	}
	return main, smask, nil
}

// CompressedData returns a cached Flate-compressed copy of the image data.
func (img *Image) CompressedData() ([]byte, error) {
	img.flateOnce.Do(func() {
		stream := core.NewStream(img.Data)
		if err := stream.Compress(); err != nil {
			img.flateErr = fmt.Errorf("compressing image data: %w", err)
			return
		}
		img.flateData = stream.Data
	})
	return img.flateData, img.flateErr
}

// CompressedAlphaData returns a cached Flate-compressed copy of the alpha mask.
func (img *Image) CompressedAlphaData() ([]byte, error) {
	img.alphaOnce.Do(func() {
		stream := core.NewStream(img.AlphaData)
		if err := stream.Compress(); err != nil {
			img.alphaErr = fmt.Errorf("compressing alpha data: %w", err)
			return
		}
		img.alphaFlate = stream.Data
	})
	return img.alphaFlate, img.alphaErr
}

// Load auto-detects image format from magic bytes and loads accordingly.
func Load(data []byte) (*Image, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("image data too short")
	}
	// JPEG: starts with FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return LoadJPEG(data)
	}
	// PNG: starts with 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return LoadPNG(data)
	}
	// TIFF: II (little-endian) or MM (big-endian)
	if (data[0] == 'I' && data[1] == 'I') || (data[0] == 'M' && data[1] == 'M') {
		return LoadTIFF(data)
	}
	// WebP: starts with RIFF....WEBP
	if data[0] == 'R' && data[1] == 'I' && data[2] == 'F' && data[3] == 'F' && len(data) > 11 && data[8] == 'W' && data[9] == 'E' && data[10] == 'B' && data[11] == 'P' {
		return LoadWebP(data)
	}
	// GIF: decode the first frame through the standard image registry.
	if data[0] == 'G' && data[1] == 'I' && data[2] == 'F' {
		img, _, err := goimage.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("gif: %w", err)
		}
		return FromGoImage(img), nil
	}
	return nil, fmt.Errorf("unsupported image format")
}

// LoadWebP decodes a WebP image and returns it as an Image.
func LoadWebP(data []byte) (*Image, error) {
	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decoding WebP: %w", err)
	}
	return FromGoImage(img), nil
}

// Ensure color is imported (used by FromGoImage indirectly).
var _ = color.RGBAModel

var decodedImageCache sync.Map

// LoadCached loads image data and reuses decoded immutable image objects by
// content hash. It is useful for high-volume template/report generation where
// the same logo or chart appears in many documents.
func LoadCached(data []byte) (*Image, error) {
	key := imageCacheKey(data)
	if cached, ok := decodedImageCache.Load(key); ok {
		return cached.(*Image), nil
	}
	img, err := Load(data)
	if err != nil {
		return nil, err
	}
	actual, _ := decodedImageCache.LoadOrStore(key, img)
	return actual.(*Image), nil
}

// ResetDecodeCache clears the decoded image cache.
func ResetDecodeCache() {
	decodedImageCache = sync.Map{}
}

func imageCacheKey(data []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(data)
	return h.Sum64()
}
