package layout

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 0.001 }

func TestBoxModelTotalHorizontal(t *testing.T) {
	b := BoxModel{
		MarginLeft: 5, MarginRight: 5,
		PaddingLeft: 10, PaddingRight: 10,
		BorderLeftWidth: 1, BorderRightWidth: 1,
	}
	if !approx(b.TotalHorizontal(), 32) {
		t.Errorf("got %f, want 32", b.TotalHorizontal())
	}
}

func TestBoxModelTotalVertical(t *testing.T) {
	b := BoxModel{
		MarginTop: 5, MarginBottom: 5,
		PaddingTop: 10, PaddingBottom: 10,
		BorderTopWidth: 2, BorderBottomWidth: 2,
	}
	if !approx(b.TotalVertical(), 34) {
		t.Errorf("got %f, want 34", b.TotalVertical())
	}
}

func TestBoxModelContentLeft(t *testing.T) {
	b := BoxModel{MarginLeft: 3, BorderLeftWidth: 1, PaddingLeft: 5}
	if !approx(b.ContentLeft(), 9) {
		t.Errorf("got %f, want 9", b.ContentLeft())
	}
}

func TestBoxModelContentTop(t *testing.T) {
	b := BoxModel{MarginTop: 4, BorderTopWidth: 2, PaddingTop: 6}
	if !approx(b.ContentTop(), 12) {
		t.Errorf("got %f, want 12", b.ContentTop())
	}
}

func TestBoxModelInnerHorizontal(t *testing.T) {
	b := BoxModel{
		MarginLeft: 100, MarginRight: 100, // should not be included
		PaddingLeft: 10, PaddingRight: 10,
		BorderLeftWidth: 1, BorderRightWidth: 1,
	}
	if !approx(b.InnerHorizontal(), 22) {
		t.Errorf("got %f, want 22", b.InnerHorizontal())
	}
}

func TestBoxModelZero(t *testing.T) {
	b := BoxModel{}
	if b.TotalHorizontal() != 0 || b.TotalVertical() != 0 {
		t.Error("zero box model should have zero totals")
	}
	if b.ContentLeft() != 0 || b.ContentTop() != 0 {
		t.Error("zero box model should have zero content offsets")
	}
}

func TestNewDrawContext(t *testing.T) {
	ctx := NewDrawContext(612, 792)
	if ctx.PageWidth != 612 || ctx.PageHeight != 792 {
		t.Error("page dimensions wrong")
	}
	if ctx.Fonts == nil || ctx.Images == nil {
		t.Error("maps should be initialized")
	}
}

func TestDrawContextWrite(t *testing.T) {
	ctx := NewDrawContext(100, 100)
	ctx.Write([]byte("q\n"))
	ctx.WriteString("Q\n")
	if string(ctx.ContentStream) != "q\nQ\n" {
		t.Errorf("got %q", string(ctx.ContentStream))
	}
}

func TestLayoutStatusConstants(t *testing.T) {
	if LayoutFull != 0 || LayoutPartial != 1 || LayoutNothing != 2 {
		t.Error("LayoutStatus constants changed")
	}
}

func TestAlignmentConstants(t *testing.T) {
	if AlignLeft != 0 || AlignCenter != 1 || AlignRight != 2 || AlignJustify != 3 {
		t.Error("Alignment constants changed")
	}
}
