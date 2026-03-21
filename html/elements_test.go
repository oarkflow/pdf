package html

import (
	"strings"
	"testing"

	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/svg"
)

func TestParseSVGDimensions_ViewBoxCommaSeparated(t *testing.T) {
	root := &svg.SVGNode{
		Attrs: map[string]string{
			"viewBox": "0,0,120,80",
		},
	}

	w, h := parseSVGDimensions(root)
	if w != 90 || h != 60 {
		t.Fatalf("dimensions = %.2fx%.2f, want 90x60", w, h)
	}
}

func TestPlanSVGLayout_AppliesObjectFitOffsetInPlacementMatrix(t *testing.T) {
	el := &ImageElement{
		Style: &ComputedStyle{
			Width:     CSSLength{Value: 200, Unit: "pt"},
			Height:    CSSLength{Value: 200, Unit: "pt"},
			ObjectFit: "contain",
		},
	}

	plan := el.planSVGLayout([]byte(`<svg width="100" height="50"><rect width="100" height="50" fill="red"/></svg>`), 200, 200, layout.BoxModel{})
	if len(plan.Blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(plan.Blocks))
	}

	ctx := layout.NewDrawContext(300, 300)
	plan.Blocks[0].Draw(ctx, 10, 210)
	stream := string(ctx.ContentStream)

	if !strings.Contains(stream, "2.6667 0 0 -2.6667 10.0000 160.0000 cm") {
		t.Fatalf("expected placement matrix with contain offset, got:\n%s", stream)
	}
}
