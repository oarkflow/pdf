package layout

import "fmt"

// BarcodeElement is a stub for barcode layout.
type BarcodeElement struct {
	Type   string  // "qr", "code128", "ean13", "pdf417"
	Data   string
	Width  float64
	Height float64
	Quiet  float64 // quiet zone
}

// NewBarcode creates a new barcode element.
func NewBarcode(barcodeType string, data string) *BarcodeElement {
	return &BarcodeElement{
		Type:   barcodeType,
		Data:   data,
		Width:  100,
		Height: 40,
		Quiet:  4,
	}
}

// PlanLayout implements Element.
func (b *BarcodeElement) PlanLayout(area LayoutArea) LayoutPlan {
	totalW := b.Width + b.Quiet*2
	totalH := b.Height + b.Quiet*2

	if totalH > area.Height {
		return LayoutPlan{Status: LayoutNothing}
	}

	localQuiet := b.Quiet
	localW := b.Width
	localH := b.Height

	block := PlacedBlock{
		X: 0, Y: 0,
		Width: totalW, Height: totalH,
		Tag:     "Figure",
		AltText: b.Data,
		Draw: func(ctx *DrawContext, x, pdfY float64) {
			// Stub: actual barcode rendering would go here
			// Draw a placeholder rectangle
			ctx.WriteString("q\n")
			ctx.WriteString("0.8 0.8 0.8 rg\n")
			ctx.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f re f\n",
				x+localQuiet, pdfY-totalH+localQuiet, localW, localH))
			ctx.WriteString("Q\n")
		},
	}

	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: totalH,
		Blocks:   []PlacedBlock{block},
	}
}
