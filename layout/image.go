package layout

import "fmt"

// ImageElement is an image layout element.
type ImageElement struct {
	Source  []byte   // raw image data
	Image   ImageEntry
	Width   float64  // desired width (0 = auto)
	Height  float64  // desired height (0 = auto)
	OrigW   int      // original pixel width
	OrigH   int      // original pixel height
	Fit     ImageFit
	Align   Alignment
	Alt     string
	PDFName string // assigned during rendering
	ObjNum  int
}

// ImageFit controls how the image fits its container.
type ImageFit int

const (
	FitContain ImageFit = iota
	FitCover
	FitFill
	FitNone
)

// NewImage creates a new image element.
func NewImage(data []byte, origWidth, origHeight int) *ImageElement {
	return &ImageElement{
		Source: data,
		OrigW:  origWidth,
		OrigH:  origHeight,
		Fit:    FitContain,
	}
}

// PlanLayout implements Element.
func (img *ImageElement) PlanLayout(area LayoutArea) LayoutPlan {
	displayW, displayH := img.computeSize(area.Width, area.Height)

	if displayH > area.Height {
		return LayoutPlan{Status: LayoutNothing}
	}

	// Alignment offset
	offsetX := 0.0
	switch img.Align {
	case AlignCenter:
		offsetX = (area.Width - displayW) / 2
	case AlignRight:
		offsetX = area.Width - displayW
	}

	localW := displayW
	localH := displayH
	localEntry := img.Image
	localEntry.Width = img.OrigW
	localEntry.Height = img.OrigH
	alt := img.Alt

	block := PlacedBlock{
		X: offsetX, Y: 0,
		Width: localW, Height: localH,
		Tag:     "Figure",
		AltText: alt,
		Draw: func(ctx *DrawContext, x, pdfY float64) {
			// Register image
			imgName := fmt.Sprintf("Im%d", len(ctx.Images)+1)
			localEntry.PDFName = imgName
			ctx.Images[imgName] = localEntry

			// Emit image placement: q w 0 0 h x y cm /ImN Do Q
			ctx.WriteString(fmt.Sprintf("q %.2f 0 0 %.2f %.2f %.2f cm /%s Do Q\n",
				localW, localH, x, pdfY-localH, imgName))
		},
	}

	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: displayH,
		Blocks:   []PlacedBlock{block},
	}
}

func (img *ImageElement) computeSize(maxW, maxH float64) (float64, float64) {
	if img.OrigW == 0 || img.OrigH == 0 {
		w := img.Width
		h := img.Height
		if w == 0 {
			w = 100
		}
		if h == 0 {
			h = 100
		}
		return w, h
	}

	aspect := float64(img.OrigW) / float64(img.OrigH)
	w := img.Width
	h := img.Height

	switch img.Fit {
	case FitContain:
		if w == 0 && h == 0 {
			w = float64(img.OrigW)
			h = float64(img.OrigH)
		} else if w == 0 {
			w = h * aspect
		} else if h == 0 {
			h = w / aspect
		}
		// Scale down to fit
		if w > maxW {
			w = maxW
			h = w / aspect
		}
		if h > maxH {
			h = maxH
			w = h * aspect
		}

	case FitCover:
		if w == 0 {
			w = maxW
		}
		if h == 0 {
			h = maxH
		}

	case FitFill:
		if w == 0 {
			w = maxW
		}
		if h == 0 {
			h = maxH
		}

	case FitNone:
		w = float64(img.OrigW)
		h = float64(img.OrigH)
	}

	return w, h
}
