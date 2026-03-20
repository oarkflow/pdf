package layout

import "fmt"

// SignatureField is a signature placeholder element.
type SignatureField struct {
	Width, Height float64
	Label         string
}

// NewSignatureField creates a new signature field.
func NewSignatureField(width, height float64) *SignatureField {
	return &SignatureField{
		Width:  width,
		Height: height,
		Label:  "Signature",
	}
}

// PlanLayout implements Element.
func (sf *SignatureField) PlanLayout(area LayoutArea) LayoutPlan {
	if sf.Height > area.Height {
		return LayoutPlan{Status: LayoutNothing}
	}

	localW := sf.Width
	localH := sf.Height
	label := sf.Label

	block := PlacedBlock{
		X: 0, Y: 0,
		Width: localW, Height: localH,
		Tag: "Form",
		Draw: func(ctx *DrawContext, x, pdfY float64) {
			// Draw signature line
			ctx.WriteString("q\n")
			ctx.WriteString("0 0 0 RG\n0.5 w\n")
			lineY := pdfY - localH + 5
			ctx.WriteString(fmt.Sprintf("%.2f %.2f m %.2f %.2f l S\n",
				x, lineY, x+localW, lineY))

			// Draw label
			if label != "" {
				fontKey := ensureFont(ctx, "Helvetica", false, false)
				ctx.WriteString(fmt.Sprintf("0.5 0.5 0.5 rg\nBT\n/%s 8 Tf\n%.2f %.2f Td\n(%s) Tj\nET\n",
					fontKey, x, lineY-10, pdfEscapeString(label)))
			}
			ctx.WriteString("Q\n")
		},
	}

	return LayoutPlan{
		Status:   LayoutFull,
		Consumed: sf.Height,
		Blocks:   []PlacedBlock{block},
	}
}
