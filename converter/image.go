package converter

import (
	"bytes"
	"image"
	"image/color"
	"image/png"

	"github.com/oarkflow/pdf/reader"
)

// extractImages extracts images from a PDF page's XObject resources.
func extractImages(resolver *reader.Resolver, resources map[string]interface{}, imageRefs []imageRef, pageNum int) []ExtractedImage {
	xobjRef, ok := resources["/XObject"]
	if !ok {
		return nil
	}

	xobjResolved, err := resolver.ResolveReference(xobjRef)
	if err != nil {
		return nil
	}
	xobjDict, ok := xobjResolved.(map[string]interface{})
	if !ok {
		return nil
	}

	// Build a map of image ref name → CTM.
	ctmMap := make(map[string][6]float64)
	for _, ref := range imageRefs {
		ctmMap[ref.name] = ref.ctm
	}

	var images []ExtractedImage
	for name, ref := range xobjDict {
		obj, err := resolver.ResolveReference(ref)
		if err != nil {
			continue
		}
		so, ok := obj.(*reader.StreamObject)
		if !ok {
			continue
		}

		subtype, _ := so.Dict["/Subtype"].(string)
		if subtype != "/Image" {
			continue
		}

		w, _ := reader.GetInt(so.Dict, "/Width")
		h, _ := reader.GetInt(so.Dict, "/Height")
		if w == 0 || h == 0 {
			continue
		}

		filter, _ := so.Dict["/Filter"]
		filter, _ = resolver.ResolveReference(filter)
		filterStr, _ := filter.(string)

		var imgData []byte
		var mimeType string

		switch filterStr {
		case "/DCTDecode":
			// JPEG — raw stream data is the JPEG file.
			imgData = so.Data
			mimeType = "image/jpeg"
		default:
			// Try to decode as raw pixels and convert to PNG.
			decoded, err := resolver.DecompressStream(so.Dict, so.Data)
			if err != nil {
				continue
			}
			pngData, err := rawPixelsToPNG(decoded, int(w), int(h), so.Dict)
			if err != nil {
				continue
			}
			imgData = pngData
			mimeType = "image/png"
		}

		if imgData == nil {
			continue
		}

		// Get position from CTM if available.
		var x, y, imgW, imgH float64
		if ctm, ok := ctmMap[name]; ok {
			x = ctm[4]
			y = ctm[5]
			imgW = ctm[0]
			imgH = ctm[3]
		} else {
			imgW = float64(w)
			imgH = float64(h)
		}

		images = append(images, ExtractedImage{
			Data:     imgData,
			MimeType: mimeType,
			X:        x,
			Y:        y,
			Width:    imgW,
			Height:   imgH,
			PageNum:  pageNum,
		})
	}

	return images
}

// rawPixelsToPNG converts raw pixel data to a PNG image.
func rawPixelsToPNG(data []byte, w, h int, dict map[string]interface{}) ([]byte, error) {
	bpc, _ := reader.GetInt(dict, "/BitsPerComponent")
	if bpc == 0 {
		bpc = 8
	}

	cs, _ := dict["/ColorSpace"].(string)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	pos := 0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			switch cs {
			case "/DeviceGray":
				if pos < len(data) {
					v := data[pos]
					pos++
					img.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
				}
			case "/DeviceCMYK":
				if pos+3 < len(data) {
					c := float64(data[pos]) / 255
					m := float64(data[pos+1]) / 255
					yy := float64(data[pos+2]) / 255
					k := float64(data[pos+3]) / 255
					pos += 4
					r := uint8((1 - c) * (1 - k) * 255)
					g := uint8((1 - m) * (1 - k) * 255)
					b := uint8((1 - yy) * (1 - k) * 255)
					img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
				}
			default: // DeviceRGB
				if pos+2 < len(data) {
					r := data[pos]
					g := data[pos+1]
					b := data[pos+2]
					pos += 3
					img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
