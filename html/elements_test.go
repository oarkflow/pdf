package html

import (
	"strings"
	"testing"

	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/svg"
)

type testElement struct {
	plan layout.LayoutPlan
}

func (e testElement) PlanLayout(area layout.LayoutArea) layout.LayoutPlan {
	return e.plan
}

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

func TestWrapRunsPreservesExplicitMultiSpaceGaps(t *testing.T) {
	runs := []layout.TextRun{
		{Text: "Website", FontName: "Helvetica", FontSize: 10},
		{Text: "    ", FontName: "Helvetica", FontSize: 10},
		{Text: "Contact", FontName: "Helvetica", FontSize: 10},
	}

	lines := wrapRuns(runs, 500, 10)
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(lines))
	}
	var got strings.Builder
	for _, run := range lines[0].runs {
		got.WriteString(run.Text)
	}
	if got.String() != "Website    Contact" {
		t.Fatalf("wrapped text = %q", got.String())
	}
}

func TestParagraphDrawAddsLinkAnnotation(t *testing.T) {
	el := &ParagraphElement{
		Runs: []layout.TextRun{{
			Text:     "Contact us",
			FontName: "Helvetica",
			FontSize: 12,
			Color:    [3]float64{0, 0, 1},
			Link:     "mailto:billing@example.com",
		}},
		Style: &ComputedStyle{
			FontSize:   12,
			LineHeight: 1.2,
			Color:      [3]float64{0, 0, 0},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 200, Height: 200})
	if len(plan.Blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(plan.Blocks))
	}

	ctx := layout.NewDrawContext(300, 300)
	plan.Blocks[0].Draw(ctx, 20, 260)

	if len(ctx.Links) != 1 {
		t.Fatalf("links = %d, want 1", len(ctx.Links))
	}
	if got := ctx.Links[0].URI; got != "mailto:billing@example.com" {
		t.Fatalf("link uri = %q", got)
	}
}

func TestDivElementPaginatesOverflow(t *testing.T) {
	el := &DivElement{
		Children: []layout.Element{
			testElement{
				plan: layout.LayoutPlan{
					Status:   layout.LayoutPartial,
					Consumed: 36,
					Blocks:   []layout.PlacedBlock{{Width: 100, Height: 36}},
					Overflow: testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 20}},
				},
			},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 180, Height: 48})
	if plan.Status != layout.LayoutPartial {
		t.Fatalf("status = %v, want partial", plan.Status)
	}
	if plan.Overflow == nil {
		t.Fatal("expected overflow element")
	}
	if plan.Consumed > 48 {
		t.Fatalf("consumed = %.2f, exceeds page height", plan.Consumed)
	}
}

func TestDivElementDefersOvertallFullChildToNextPage(t *testing.T) {
	el := &DivElement{
		Children: []layout.Element{
			testElement{
				plan: layout.LayoutPlan{
					Status:   layout.LayoutFull,
					Consumed: 30,
					Blocks:   []layout.PlacedBlock{{Width: 80, Height: 30}},
				},
			},
			testElement{
				plan: layout.LayoutPlan{
					Status:   layout.LayoutFull,
					Consumed: 40,
					Blocks:   []layout.PlacedBlock{{Width: 80, Height: 40}},
				},
			},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 180, Height: 50})
	if plan.Status != layout.LayoutPartial {
		t.Fatalf("status = %v, want partial", plan.Status)
	}
	if plan.Overflow == nil {
		t.Fatal("expected overflow element")
	}
	if len(plan.Blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(plan.Blocks))
	}
}

func TestDivElementOverflowHiddenClipsChildren(t *testing.T) {
	el := &DivElement{
		Style: &ComputedStyle{Overflow: "hidden"},
		BoxModel: layout.BoxModel{
			BorderRadius: 6,
		},
		Children: []layout.Element{
			testElement{
				plan: layout.LayoutPlan{
					Status:   layout.LayoutFull,
					Consumed: 20,
					Blocks: []layout.PlacedBlock{{
						Width:  80,
						Height: 20,
						Draw: func(ctx *layout.DrawContext, x, topY float64) {
							ctx.WriteString("% child\n")
						},
					}},
				},
			},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 100, Height: 100})
	ctx := layout.NewDrawContext(200, 200)
	for _, block := range plan.Blocks {
		if block.Draw != nil {
			block.Draw(ctx, 10, 150)
		}
	}

	stream := string(ctx.ContentStream)
	if !strings.Contains(stream, "W n") {
		t.Fatalf("expected clip path in stream, got:\n%s", stream)
	}
	if !strings.Contains(stream, "Q\n") {
		t.Fatalf("expected clip restore in stream, got:\n%s", stream)
	}
}

func TestDivElementAspectRatioReservesHeight(t *testing.T) {
	el := &DivElement{
		Style: &ComputedStyle{
			AspectRatio: 0.9,
		},
		Children: []layout.Element{
			testElement{
				plan: layout.LayoutPlan{
					Status:   layout.LayoutFull,
					Consumed: 20,
				},
			},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 180, Height: 500})
	want := 200.0
	if plan.Consumed < want-0.01 || plan.Consumed > want+0.01 {
		t.Fatalf("consumed = %.2f, want %.2f", plan.Consumed, want)
	}
}

func TestConvertTailwindAspectRatioClassReservesHeight(t *testing.T) {
	htmlInput := `<!DOCTYPE html><html><body><div class="w-44 aspect-9/10 bg-zinc-100"></div></body></html>`
	result, err := Convert(htmlInput, Options{UseTailwind: true, DefaultFontSize: 10})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	el, ok := result.Elements[0].(*DivElement)
	if !ok {
		t.Fatalf("element type = %T, want *DivElement", result.Elements[0])
	}
	if el.Style == nil || el.Style.AspectRatio == 0 {
		t.Fatal("expected Tailwind aspect-ratio class to populate style")
	}
	plan := el.PlanLayout(layout.LayoutArea{Width: 400, Height: 600})
	if plan.Consumed >= 600 {
		t.Fatalf("consumed = %.2f, expected aspect-ratio constrained height", plan.Consumed)
	}
}

func TestToWinAnsiEmitsSingleByteCopyright(t *testing.T) {
	got := []byte(toWinAnsi("©"))
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0] != 0xA9 {
		t.Fatalf("byte = 0x%X, want 0xA9", got[0])
	}
}

func TestConvertTreatsStyledInlineBlockAsInlineBox(t *testing.T) {
	htmlInput := `<!DOCTYPE html><html><body><div class="text-center"><button class="bg-blue-600 text-white px-6 py-2 rounded-lg">Subscribe Now</button></div></body></html>`
	result, err := Convert(htmlInput, Options{UseTailwind: true, DefaultFontSize: 10})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(result.Elements))
	}

	root, ok := result.Elements[0].(*DivElement)
	if !ok {
		t.Fatalf("root type = %T, want *DivElement", result.Elements[0])
	}
	if len(root.Children) != 1 {
		t.Fatalf("root children = %d, want 1", len(root.Children))
	}

	btn, ok := root.Children[0].(*InlineBoxElement)
	if !ok {
		t.Fatalf("child type = %T, want *InlineBoxElement", root.Children[0])
	}
	if btn.BoxModel.Background == nil {
		t.Fatal("expected button background")
	}
	if btn.BoxModel.PaddingLeft <= 0 || btn.BoxModel.PaddingRight <= 0 {
		t.Fatalf("expected horizontal padding, got left=%.2f right=%.2f", btn.BoxModel.PaddingLeft, btn.BoxModel.PaddingRight)
	}
	if btn.BoxModel.BorderRadius <= 0 {
		t.Fatalf("expected border radius, got %.2f", btn.BoxModel.BorderRadius)
	}
	if btn.OuterAlign != "center" {
		t.Fatalf("outer align = %q, want center", btn.OuterAlign)
	}
}

func TestConvertTailwindShadowClassRendersAsShadowImage(t *testing.T) {
	htmlInput := `<!DOCTYPE html><html><body><div class="bg-white rounded-lg shadow-lg p-4">Card</div></body></html>`
	result, err := Convert(htmlInput, Options{UseTailwind: true, DefaultFontSize: 10})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(result.Elements))
	}

	card, ok := result.Elements[0].(*DivElement)
	if !ok {
		t.Fatalf("element type = %T, want *DivElement", result.Elements[0])
	}
	if card.BoxModel.BoxShadow == "" {
		t.Fatal("expected Tailwind shadow class to populate BoxShadow")
	}

	plan := card.PlanLayout(layout.LayoutArea{Width: 240, Height: 240})
	if len(plan.Blocks) == 0 {
		t.Fatal("expected layout blocks")
	}

	ctx := layout.NewDrawContext(300, 300)
	for _, block := range plan.Blocks {
		if block.Draw != nil {
			block.Draw(ctx, 20, 260)
		}
	}

	if len(ctx.Images) == 0 {
		t.Fatal("expected shadow image to be registered for Tailwind class")
	}
	foundAlpha := false
	for _, entry := range ctx.Images {
		if entry.Image != nil && len(entry.Image.AlphaData) > 0 {
			foundAlpha = true
			break
		}
	}
	if !foundAlpha {
		t.Fatal("expected Tailwind shadow image to include alpha data")
	}
}

func TestEstimateIntrinsicWidthHandlesNestedFlexContainer(t *testing.T) {
	menu := &FlexContainerElement{
		Style: &ComputedStyle{
			FlexDirection: "row",
			Gap:           6,
		},
		Children: []FlexChildElement{
			{Element: &ParagraphElement{Runs: []layout.TextRun{{Text: "About", FontName: "Helvetica", FontSize: 12}}, Style: &ComputedStyle{FontSize: 12}}},
			{Element: &ParagraphElement{Runs: []layout.TextRun{{Text: "Projects", FontName: "Helvetica", FontSize: 12}}, Style: &ComputedStyle{FontSize: 12}}},
			{Element: &ParagraphElement{Runs: []layout.TextRun{{Text: "Speaking", FontName: "Helvetica", FontSize: 12}}, Style: &ComputedStyle{FontSize: 12}}},
		},
	}

	width := estimateIntrinsicWidth(&DivElement{
		Children: []layout.Element{menu},
	})
	if width <= 60 {
		t.Fatalf("intrinsic width = %.2f, expected nested flex width to include its text", width)
	}
}

func TestDivElementWrapsBlocksWithTransform(t *testing.T) {
	el := &DivElement{
		Style: &ComputedStyle{
			Transform:       "rotate(2deg)",
			TransformOrigin: "center",
		},
		Children: []layout.Element{
			testElement{
				plan: layout.LayoutPlan{
					Status:   layout.LayoutFull,
					Consumed: 20,
					Blocks: []layout.PlacedBlock{{
						Width: 80, Height: 20,
						Draw: func(ctx *layout.DrawContext, x, topY float64) {
							ctx.WriteString("% child\n")
						},
					}},
				},
			},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 100, Height: 100})
	ctx := layout.NewDrawContext(200, 200)
	for _, block := range plan.Blocks {
		if block.Draw != nil {
			block.Draw(ctx, 10, 150)
		}
	}

	stream := string(ctx.ContentStream)
	if !strings.Contains(stream, " cm ") {
		t.Fatalf("expected transform matrix in stream, got:\n%s", stream)
	}
	if !strings.Contains(stream, "% child") {
		t.Fatalf("expected child content inside transformed block, got:\n%s", stream)
	}
}

func TestListElementSkipsMarkersWhenListStyleNone(t *testing.T) {
	el := &ListElement{
		Style: &ComputedStyle{
			ListStyleType: "none",
			FontSize:      12,
			LineHeight:    1.4,
		},
		Items: []ListItem{
			{
				Marker: "1. ",
				Children: []layout.Element{
					testElement{
						plan: layout.LayoutPlan{
							Status:   layout.LayoutFull,
							Consumed: 18,
							Blocks: []layout.PlacedBlock{{
								Width: 80, Height: 18,
								Draw: func(ctx *layout.DrawContext, x, topY float64) {
									ctx.WriteString("% child item\n")
								},
							}},
						},
					},
				},
			},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 200, Height: 200})
	ctx := layout.NewDrawContext(300, 300)
	for _, block := range plan.Blocks {
		if block.Draw != nil {
			block.Draw(ctx, 20, 250)
		}
	}

	stream := string(ctx.ContentStream)
	if strings.Contains(stream, "(1. ) Tj") {
		t.Fatalf("expected list-style none to suppress markers, got:\n%s", stream)
	}
	if !strings.Contains(stream, "% child item") {
		t.Fatalf("expected child content to render, got:\n%s", stream)
	}
}

func TestGridContainerElementUsesGapAndTrackWidths(t *testing.T) {
	grid := &GridContainerElement{
		Style: &ComputedStyle{Gap: 20},
		Columns: []GridTrack{
			{Fr: 1},
			{Fr: 1},
		},
		Children: []layout.Element{
			testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 40, Blocks: []layout.PlacedBlock{{Width: 50, Height: 40}}}},
			testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 40, Blocks: []layout.PlacedBlock{{Width: 50, Height: 40}}}},
		},
	}

	plan := grid.PlanLayout(layout.LayoutArea{Width: 220, Height: 200})
	if len(plan.Blocks) != 2 {
		t.Fatalf("blocks = %d, want 2", len(plan.Blocks))
	}
	if plan.Blocks[1].X <= plan.Blocks[0].X+90 {
		t.Fatalf("expected second column to include gap, got x1=%.2f x2=%.2f", plan.Blocks[0].X, plan.Blocks[1].X)
	}
}

func TestConvertFlexContainerPreservesDirectTextChildren(t *testing.T) {
	htmlInput := `<!DOCTYPE html><html><body><time class="flex items-center">July 14, 2022</time></body></html>`
	result, err := Convert(htmlInput, Options{UseTailwind: true, DefaultFontSize: 10})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(result.Elements))
	}
	flex, ok := result.Elements[0].(*FlexContainerElement)
	if !ok {
		t.Fatalf("element type = %T, want *FlexContainerElement", result.Elements[0])
	}
	if len(flex.Children) == 0 {
		t.Fatal("expected flex children")
	}
	para, ok := flex.Children[0].Element.(*ParagraphElement)
	if !ok {
		t.Fatalf("first child type = %T, want *ParagraphElement", flex.Children[0].Element)
	}
	if len(para.Runs) == 0 || !strings.Contains(para.Runs[0].Text, "July 14, 2022") {
		t.Fatalf("expected preserved text run, got %#v", para.Runs)
	}
}

func TestConvertHeadingPreservesNestedLinkedSpanText(t *testing.T) {
	htmlInput := `<!DOCTYPE html><html><body><h2 class="text-base font-semibold tracking-tight text-zinc-800"><div class="absolute -inset-x-4 -inset-y-6 z-0 scale-95 bg-zinc-50 opacity-0"></div><a href="/articles/rewriting-the-cosmos-kernel-in-rust"><span class="absolute -inset-x-4 -inset-y-6 z-20"></span><span class="relative z-10">Rewriting the cosmOS kernel in Rust</span></a></h2></body></html>`
	result, err := Convert(htmlInput, Options{UseTailwind: true, DefaultFontSize: 10})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(result.Elements))
	}
	h, ok := result.Elements[0].(*HeadingElement)
	if !ok {
		t.Fatalf("element type = %T, want *HeadingElement", result.Elements[0])
	}
	if len(h.Runs) == 0 {
		t.Fatal("expected heading runs")
	}
	var got strings.Builder
	for _, run := range h.Runs {
		got.WriteString(run.Text)
	}
	if !strings.Contains(got.String(), "Rewriting the cosmOS kernel in Rust") {
		t.Fatalf("heading text = %q", got.String())
	}
}

func TestConvertArticleCardRendersHeadingAndMetaText(t *testing.T) {
	htmlInput := `<!DOCTYPE html><html><body><article class="group relative flex flex-col items-start"><h2 class="text-base font-semibold tracking-tight text-zinc-800"><div class="absolute -inset-x-4 -inset-y-6 z-0 scale-95 bg-zinc-50 opacity-0"></div><a href="/articles/rewriting-the-cosmos-kernel-in-rust"><span class="absolute -inset-x-4 -inset-y-6 z-20"></span><span class="relative z-10">Rewriting the cosmOS kernel in Rust</span></a></h2><time class="relative z-10 order-first mb-3 flex items-center text-sm text-zinc-400 pl-3.5">July 14, 2022</time><p class="relative z-10 mt-2 text-sm text-zinc-600">When we released the first version of cosmOS last year, it was written in Go.</p><div class="relative z-10 mt-4 flex items-center text-sm font-medium text-teal-500">Read article</div></article></body></html>`
	result, err := Convert(htmlInput, Options{UseTailwind: true, DefaultFontSize: 10})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	card, ok := result.Elements[0].(*FlexContainerElement)
	if !ok {
		t.Fatalf("element type = %T, want *FlexContainerElement", result.Elements[0])
	}
	plan := card.PlanLayout(layout.LayoutArea{Width: 260, Height: 260})
	ctx := layout.NewDrawContext(320, 320)
	for _, block := range plan.Blocks {
		if block.Draw != nil {
			block.Draw(ctx, 20, 280)
		}
	}
	stream := string(ctx.ContentStream)
	if !strings.Contains(stream, "Rewriting the cosmOS kernel in Rust") {
		t.Fatalf("expected heading text in stream, got:\n%s", stream)
	}
	if !strings.Contains(stream, "July 14, 2022") {
		t.Fatalf("expected date text in stream, got:\n%s", stream)
	}
	if !strings.Contains(stream, "Read article") {
		t.Fatalf("expected CTA text in stream, got:\n%s", stream)
	}
}

func TestFlexContainerElementHonorsChildOrder(t *testing.T) {
	el := &FlexContainerElement{
		Style: &ComputedStyle{
			FlexDirection: "column",
		},
		Children: []FlexChildElement{
			{
				Element: &ParagraphElement{Runs: []layout.TextRun{{Text: "Title", FontName: "Helvetica", FontSize: 12}}, Style: &ComputedStyle{FontSize: 12}},
				Style:   &ComputedStyle{Order: 0},
			},
			{
				Element: &ParagraphElement{Runs: []layout.TextRun{{Text: "Date", FontName: "Helvetica", FontSize: 12}}, Style: &ComputedStyle{FontSize: 12}},
				Style:   &ComputedStyle{Order: -9999},
			},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 200, Height: 120})
	ctx := layout.NewDrawContext(300, 300)
	for _, block := range plan.Blocks {
		if block.Draw != nil {
			block.Draw(ctx, 20, 250)
		}
	}
	stream := string(ctx.ContentStream)
	dateIdx := strings.Index(stream, "(Date) Tj")
	titleIdx := strings.Index(stream, "(Title) Tj")
	if dateIdx == -1 || titleIdx == -1 {
		t.Fatalf("expected both texts in stream, got:\n%s", stream)
	}
	if dateIdx > titleIdx {
		t.Fatalf("expected ordered date before title, got stream:\n%s", stream)
	}
}

func TestFlexContainerElementPaginatesColumnOverflow(t *testing.T) {
	el := &FlexContainerElement{
		Style: &ComputedStyle{
			FlexDirection: "column",
			Gap:           8,
		},
		Children: []FlexChildElement{
			{Element: testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 40, Blocks: []layout.PlacedBlock{{Width: 80, Height: 40}}}}},
			{Element: testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 40, Blocks: []layout.PlacedBlock{{Width: 80, Height: 40}}}}},
			{Element: testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 40, Blocks: []layout.PlacedBlock{{Width: 80, Height: 40}}}}},
		},
	}

	plan := el.PlanLayout(layout.LayoutArea{Width: 180, Height: 100})
	if plan.Status != layout.LayoutPartial {
		t.Fatalf("status = %v, want partial", plan.Status)
	}
	if plan.Overflow == nil {
		t.Fatal("expected overflow element")
	}
	if plan.Consumed > 100 {
		t.Fatalf("consumed = %.2f exceeds page height", plan.Consumed)
	}
}

func TestGridContainerElementPaginatesOverflow(t *testing.T) {
	grid := &GridContainerElement{
		Style: &ComputedStyle{Gap: 10},
		Columns: []GridTrack{
			{Fr: 1},
			{Fr: 1},
		},
		Children: []layout.Element{
			testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 60, Blocks: []layout.PlacedBlock{{Width: 40, Height: 60}}}},
			testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 60, Blocks: []layout.PlacedBlock{{Width: 40, Height: 60}}}},
			testElement{plan: layout.LayoutPlan{Status: layout.LayoutFull, Consumed: 60, Blocks: []layout.PlacedBlock{{Width: 40, Height: 60}}}},
		},
	}

	plan := grid.PlanLayout(layout.LayoutArea{Width: 220, Height: 120})
	if plan.Status != layout.LayoutPartial {
		t.Fatalf("status = %v, want partial", plan.Status)
	}
	if plan.Overflow == nil {
		t.Fatal("expected overflow element")
	}
	if plan.Consumed > 120 {
		t.Fatalf("consumed = %.2f exceeds page height", plan.Consumed)
	}
}

func TestConvertTailwindDynamicShadowClassRendersAsShadowImage(t *testing.T) {
	htmlInput := `<!DOCTYPE html><html><body><div class="bg-white rounded-lg shadow-[0_10px_15px_-3px_rgb(0_0_0_/_0.12)] p-4">Card</div></body></html>`
	result, err := Convert(htmlInput, Options{UseTailwind: true, DefaultFontSize: 10})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(result.Elements) != 1 {
		t.Fatalf("elements = %d, want 1", len(result.Elements))
	}

	card, ok := result.Elements[0].(*DivElement)
	if !ok {
		t.Fatalf("element type = %T, want *DivElement", result.Elements[0])
	}
	if got := card.BoxModel.BoxShadow; got != "0 10px 15px -3px rgb(0 0 0 / 0.12)" {
		t.Fatalf("box shadow = %q", got)
	}

	plan := card.PlanLayout(layout.LayoutArea{Width: 240, Height: 240})
	ctx := layout.NewDrawContext(300, 300)
	for _, block := range plan.Blocks {
		if block.Draw != nil {
			block.Draw(ctx, 20, 260)
		}
	}
	if len(ctx.Images) == 0 {
		t.Fatal("expected dynamic Tailwind shadow image to be registered")
	}
}

func TestConvertTailwindShadowColorUtilityAffectsRenderedShadow(t *testing.T) {
	htmlInput := `<!DOCTYPE html><html><body><div class="bg-white rounded-lg shadow-lg shadow-red-500/25 p-4">Card</div></body></html>`
	result, err := Convert(htmlInput, Options{UseTailwind: true, DefaultFontSize: 10})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	card, ok := result.Elements[0].(*DivElement)
	if !ok {
		t.Fatalf("element type = %T, want *DivElement", result.Elements[0])
	}
	if !strings.Contains(card.BoxModel.BoxShadow, "rgba(239, 68, 68, 0.25)") {
		t.Fatalf("expected red Tailwind shadow color, got %q", card.BoxModel.BoxShadow)
	}
}

func TestDrawBoxModelFallsBackForAsymmetricRoundedBorders(t *testing.T) {
	ctx := layout.NewDrawContext(200, 200)
	drawBoxModel(ctx, 10, 180, 80, 40, layout.BoxModel{
		BorderTopWidth:    0,
		BorderRightWidth:  1,
		BorderBottomWidth: 1,
		BorderLeftWidth:   1,
		BorderColor:       [3]float64{0.5, 0.5, 0.5},
		BorderRightColor:  [3]float64{0.918, 0.929, 0.945},
		BorderBottomColor: [3]float64{0.918, 0.929, 0.945},
		BorderLeftColor:   [3]float64{0.918, 0.929, 0.945},
		BorderRadius:      6,
	})

	stream := string(ctx.ContentStream)
	if strings.Contains(stream, " c ") {
		t.Fatalf("expected asymmetric borders to avoid rounded rectangle stroke, got:\n%s", stream)
	}
	if strings.Contains(stream, " 0.00 w ") {
		t.Fatalf("expected hidden top border to stay omitted, got:\n%s", stream)
	}
	if !strings.Contains(stream, "0.918 0.929 0.945 RG") {
		t.Fatalf("expected visible borders to use light gray stroke, got:\n%s", stream)
	}
}

func TestDrawBoxModelRendersGradientAndSimpleShadow(t *testing.T) {
	ctx := layout.NewDrawContext(200, 200)
	drawBoxModel(ctx, 10, 180, 100, 50, layout.BoxModel{
		BackgroundImage: "linear-gradient(90deg, #667eea 0%, #764ba2 100%)",
		BoxShadow:       "4px 6px #d1d5db",
	})

	stream := string(ctx.ContentStream)
	if !strings.Contains(stream, "/Im1 Do") {
		t.Fatalf("expected shadow image draw command, got:\n%s", stream)
	}
	if count := strings.Count(stream, " re f\n"); count < 10 {
		t.Fatalf("expected gradient strips in stream, got count=%d\n%s", count, stream)
	}
	if len(ctx.Images) == 0 {
		t.Fatal("expected shadow image to be registered")
	}
}

func TestDrawBoxModelRendersBlurredMultiLayerShadowAsImage(t *testing.T) {
	ctx := layout.NewDrawContext(200, 200)
	drawBoxModel(ctx, 10, 180, 100, 50, layout.BoxModel{
		BoxShadow: "0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)",
	})

	stream := string(ctx.ContentStream)
	if !strings.Contains(stream, "/Im1 Do") {
		t.Fatalf("expected blurred shadow image draw command, got:\n%s", stream)
	}
	entry, ok := ctx.Images["Im1"]
	if !ok || entry.Image == nil {
		t.Fatal("expected blurred shadow image to be registered")
	}
	if len(entry.Image.AlphaData) == 0 {
		t.Fatal("expected blurred shadow image to include alpha data")
	}
}

func TestDrawBoxModelUsesBackgroundPositionSizeAndRepeat(t *testing.T) {
	ctx := layout.NewDrawContext(240, 240)
	drawBoxModel(ctx, 10, 210, 120, 80, layout.BoxModel{
		BackgroundImage:    "linear-gradient(90deg, #667eea 0%, #764ba2 100%)",
		BackgroundPosition: "center top",
		BackgroundSize:     "50% 25%",
		BackgroundRepeat:   "repeat-x",
	})

	stream := string(ctx.ContentStream)
	if !strings.Contains(stream, "40.00 190.00") {
		t.Fatalf("expected positioned gradient tile near centered top origin, got:\n%s", stream)
	}
	if count := strings.Count(stream, " re f\n"); count < 20 {
		t.Fatalf("expected repeated gradient tiles, got count=%d\n%s", count, stream)
	}
}
