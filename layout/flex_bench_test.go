package layout

import "testing"

func BenchmarkFlexLayout(b *testing.B) {
	children := make([]Element, 20)
	for i := range children {
		children[i] = NewParagraph("Flex child content for benchmarking layout performance")
	}
	flex := NewFlex(FlexRow, children...)
	flex.Gap = 10
	flex.Justify = JustifySpaceBetween
	flex.AlignItems = AlignItemsCenter
	area := LayoutArea{Width: 800, Height: 600}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		flex.PlanLayout(area)
	}
}
