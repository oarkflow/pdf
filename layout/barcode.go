package layout

import (
	"fmt"
	"strings"

	pdfbarcode "github.com/oarkflow/pdf/barcode"
)

// BarcodeElement lays out and draws QR, Code 128, EAN-13, and PDF417 barcodes.
type BarcodeElement struct {
	Type   string // "qr", "code128", "ean13", "pdf417"
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
	localType := strings.ToLower(strings.TrimSpace(b.Type))
	localData := b.Data

	block := PlacedBlock{
		X: 0, Y: 0,
		Width: totalW, Height: totalH,
		Tag:     "Figure",
		AltText: b.Data,
		Draw: func(ctx *DrawContext, x, pdfY float64) {
			drawBarcode(ctx, localType, localData, x+localQuiet, pdfY-localQuiet, localW, localH)
		},
	}

	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: totalH,
		Blocks:   []PlacedBlock{block},
	}
}

func drawBarcode(ctx *DrawContext, typ, data string, x, topY, width, height float64) {
	ctx.WriteString("q\n1 1 1 rg\n")
	ctx.WriteString(fmt.Sprintf("%.2f %.2f %.2f %.2f re f\n", x, topY-height, width, height))
	ctx.WriteString("0 0 0 rg\n")
	switch typ {
	case "qr":
		qr, err := pdfbarcode.EncodeQR(data, pdfbarcode.ECMedium)
		if err == nil {
			drawBarcodeMatrix(ctx, qr.Modules, x, topY, width, height)
		}
	case "ean13":
		ean, err := pdfbarcode.EncodeEAN13(data)
		if err == nil {
			drawBarcodeBars(ctx, ean.Bars, x, topY, width, height)
		}
	case "pdf417":
		pdf417, err := pdfbarcode.EncodePDF417(data, 2)
		if err == nil {
			drawBarcodeMatrix(ctx, pdf417.Matrix, x, topY, width, height)
		}
	default:
		code, err := pdfbarcode.EncodeCode128(data)
		if err == nil {
			drawBarcodeBars(ctx, code.Bars, x, topY, width, height)
		}
	}
	ctx.WriteString("Q\n")
}

func drawBarcodeBars(ctx *DrawContext, bars []bool, x, topY, width, height float64) {
	if len(bars) == 0 {
		return
	}
	barW := width / float64(len(bars))
	for i, bar := range bars {
		if !bar {
			continue
		}
		ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n", x+float64(i)*barW, topY-height, barW, height))
	}
}

func drawBarcodeMatrix(ctx *DrawContext, matrix [][]bool, x, topY, width, height float64) {
	if len(matrix) == 0 || len(matrix[0]) == 0 {
		return
	}
	cellW := width / float64(len(matrix[0]))
	cellH := height / float64(len(matrix))
	for row, cells := range matrix {
		for col, dark := range cells {
			if !dark {
				continue
			}
			ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f %.3f re f\n", x+float64(col)*cellW, topY-float64(row+1)*cellH, cellW, cellH))
		}
	}
}
