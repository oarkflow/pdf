package layout

import (
	"fmt"
	"strings"
)

// Paragraph is a text paragraph element with word wrapping, alignment, and line spacing.
type Paragraph struct {
	Runs        []TextRun
	Align       Alignment
	LineHeight  float64 // multiplier (default 1.2)
	SpaceBefore float64
	SpaceAfter  float64
	Indent      float64
	MaxLines    int // 0 = unlimited
}

// NewParagraph creates a paragraph with a single default TextRun.
func NewParagraph(text string) *Paragraph {
	return &Paragraph{
		Runs: []TextRun{{
			Text:     text,
			FontName: "Helvetica",
			FontSize: 12,
			Color:    [3]float64{0, 0, 0},
		}},
		LineHeight: 1.2,
	}
}

// NewStyledParagraph creates a paragraph with multiple styled runs.
func NewStyledParagraph(runs ...TextRun) *Paragraph {
	for i := range runs {
		if runs[i].FontName == "" {
			runs[i].FontName = "Helvetica"
		}
		if runs[i].FontSize == 0 {
			runs[i].FontSize = 12
		}
	}
	return &Paragraph{
		Runs:       runs,
		LineHeight: 1.2,
	}
}

// measureText approximates text width using fontSize * 0.5 per character.
func measureText(text string, fontSize float64) float64 {
	return float64(len(text)) * fontSize * 0.5
}

// splitRunsIntoWords splits text runs into measured words.
func splitRunsIntoWords(runs []TextRun) []Word {
	// Flatten all runs into a single text with style tracking
	var words []Word

	// Concatenate all text, tracking which run each char belongs to
	var chars []charInfo
	for _, r := range runs {
		for _, ch := range r.Text {
			chars = append(chars, charInfo{ch: ch, run: r})
		}
	}

	// Split into words at spaces
	var currentWord []charInfo
	for _, ci := range chars {
		if ci.ch == ' ' || ci.ch == '\t' {
			if len(currentWord) > 0 {
				w := buildWord(currentWord)
				// Space width based on first run's font size
				w.Space = currentWord[0].run.FontSize * 0.5
				words = append(words, w)
				currentWord = nil
			}
		} else {
			currentWord = append(currentWord, ci)
		}
	}
	if len(currentWord) > 0 {
		words = append(words, buildWord(currentWord))
	}

	return words
}

type charInfo struct {
	ch  rune
	run TextRun
}

func buildWord(chars []charInfo) Word {
	var runs []TextRun
	var width float64

	if len(chars) == 0 {
		return Word{}
	}

	// Group consecutive chars with same style into runs
	currentRun := chars[0].run
	currentRun.Text = string(chars[0].ch)
	width += chars[0].run.FontSize * 0.5

	for i := 1; i < len(chars); i++ {
		if sameStyle(chars[i].run, currentRun) {
			currentRun.Text += string(chars[i].ch)
		} else {
			runs = append(runs, currentRun)
			currentRun = chars[i].run
			currentRun.Text = string(chars[i].ch)
		}
		width += chars[i].run.FontSize * 0.5
	}
	runs = append(runs, currentRun)

	return Word{Runs: runs, Width: width}
}

func sameStyle(a, b TextRun) bool {
	return a.FontName == b.FontName && a.FontSize == b.FontSize &&
		a.Bold == b.Bold && a.Italic == b.Italic &&
		a.Color == b.Color && a.Underline == b.Underline &&
		a.Strike == b.Strike && a.Link == b.Link
}

// wrapWords wraps words into lines fitting the given width.
func wrapWords(words []Word, width float64, indent float64, maxLines int) (lines []Line, overflowWords []Word) {
	if len(words) == 0 {
		return nil, nil
	}

	var currentLine Line
	lineWidth := indent
	isFirstLine := true

	for i, w := range words {
		needed := w.Width
		if len(currentLine.Words) > 0 {
			needed += w.Space
		}
		if isFirstLine && len(currentLine.Words) == 0 {
			lineWidth = indent
		}

		if len(currentLine.Words) > 0 && lineWidth+needed > width {
			// Finish current line
			currentLine.Width = lineWidth
			updateLineMetrics(&currentLine)
			lines = append(lines, currentLine)

			if maxLines > 0 && len(lines) >= maxLines {
				return lines, words[i:]
			}

			currentLine = Line{}
			lineWidth = 0
			isFirstLine = false
		}

		if len(currentLine.Words) > 0 {
			lineWidth += w.Space
		}
		currentLine.Words = append(currentLine.Words, w)
		lineWidth += w.Width
	}

	if len(currentLine.Words) > 0 {
		currentLine.Width = lineWidth
		updateLineMetrics(&currentLine)
		lines = append(lines, currentLine)
	}

	return lines, nil
}

func updateLineMetrics(line *Line) {
	maxSize := 0.0
	for _, w := range line.Words {
		for _, r := range w.Runs {
			if r.FontSize > maxSize {
				maxSize = r.FontSize
			}
		}
	}
	line.Ascent = maxSize * 0.8
	line.Height = maxSize
}

// PlanLayout implements the Element interface for Paragraph.
func (p *Paragraph) PlanLayout(area LayoutArea) LayoutPlan {
	lineHeight := p.LineHeight
	if lineHeight == 0 {
		lineHeight = 1.2
	}

	totalBefore := p.SpaceBefore
	totalAfter := p.SpaceAfter

	// Check if at least space before + one line fits
	words := splitRunsIntoWords(p.Runs)
	if len(words) == 0 {
		return LayoutPlan{Status: LayoutFull, Consumed: totalBefore + totalAfter}
	}

	lines, overflow := wrapWords(words, area.Width, p.Indent, 0)
	_ = overflow

	// Calculate total height
	totalHeight := totalBefore
	for _, ln := range lines {
		totalHeight += ln.Height * lineHeight
	}
	totalHeight += totalAfter

	if totalHeight <= area.Height || len(lines) <= 1 {
		// Fits fully (or just one line, place it anyway)
		blocks := p.buildLineBlocks(lines, area.Width, lineHeight, totalBefore)
		consumed := totalHeight
		if consumed > area.Height {
			consumed = area.Height
		}
		return LayoutPlan{
			Status:   LayoutFull,
			Consumed: consumed,
			Blocks:   blocks,
		}
	}

	// Partial: figure out how many lines fit
	fitted := 0
	usedHeight := totalBefore
	for _, ln := range lines {
		lh := ln.Height * lineHeight
		if usedHeight+lh > area.Height {
			break
		}
		usedHeight += lh
		fitted++
	}

	if fitted == 0 {
		return LayoutPlan{Status: LayoutNothing}
	}

	blocks := p.buildLineBlocks(lines[:fitted], area.Width, lineHeight, totalBefore)

	// Build overflow paragraph from remaining lines
	var remainingWords []Word
	for _, ln := range lines[fitted:] {
		remainingWords = append(remainingWords, ln.Words...)
	}
	overflowPara := p.clone()
	overflowPara.Runs = wordsToRuns(remainingWords)
	overflowPara.SpaceBefore = 0
	overflowPara.Indent = 0

	return LayoutPlan{
		Status:   LayoutPartial,
		Consumed: usedHeight,
		Blocks:   blocks,
		Overflow: overflowPara,
	}
}

func (p *Paragraph) clone() *Paragraph {
	cp := *p
	cp.Runs = make([]TextRun, len(p.Runs))
	copy(cp.Runs, p.Runs)
	return &cp
}

func wordsToRuns(words []Word) []TextRun {
	var runs []TextRun
	for i, w := range words {
		for _, r := range w.Runs {
			if i > 0 && len(runs) > 0 {
				// Add space before word
				last := &runs[len(runs)-1]
				if sameStyle(r, *last) {
					last.Text += " " + r.Text
					continue
				}
				last.Text += " "
			}
			runs = append(runs, r)
		}
	}
	return runs
}

func (p *Paragraph) buildLineBlocks(lines []Line, areaWidth float64, lineHeight float64, spaceBefore float64) []PlacedBlock {
	blocks := make([]PlacedBlock, 0, len(lines))
	y := spaceBefore

	for lineIdx, ln := range lines {
		localY := y
		localLine := ln
		localAlign := p.Align
		isLast := lineIdx == len(lines)-1

		block := PlacedBlock{
			X:      0,
			Y:      localY,
			Width:  areaWidth,
			Height: localLine.Height * lineHeight,
			Tag:    "P",
			Draw: func(ctx *DrawContext, x, pdfY float64) {
				drawLine(ctx, localLine, x, pdfY, areaWidth, localAlign, isLast)
			},
		}
		blocks = append(blocks, block)
		y += localLine.Height * lineHeight
	}

	return blocks
}

func drawLine(ctx *DrawContext, ln Line, x, pdfY, areaWidth float64, align Alignment, isLast bool) {
	// Calculate starting X based on alignment
	startX := x
	switch align {
	case AlignCenter:
		startX = x + (areaWidth-ln.Width)/2
	case AlignRight:
		startX = x + areaWidth - ln.Width
	case AlignJustify:
		// handled below
	}

	// Calculate word spacing for justify
	wordSpacing := 0.0
	if align == AlignJustify && !isLast && len(ln.Words) > 1 {
		totalWordWidth := 0.0
		for _, w := range ln.Words {
			totalWordWidth += w.Width
		}
		wordSpacing = (areaWidth - totalWordWidth) / float64(len(ln.Words)-1)
	}

	curX := startX
	for i, w := range ln.Words {
		for _, r := range w.Runs {
			// Escape PDF string
			escaped := pdfEscapeString(r.Text)
			fontKey := ensureFont(ctx, r.FontName, r.Bold, r.Italic)

			// Set color
			ctx.WriteString(fmt.Sprintf("%.3f %.3f %.3f rg\n", r.Color[0], r.Color[1], r.Color[2]))

			ctx.WriteString("BT\n")
			ctx.WriteString(fmt.Sprintf("/%s %.1f Tf\n", fontKey, r.FontSize))
			ctx.WriteString(fmt.Sprintf("%.2f %.2f Td\n", curX, pdfY-ln.Ascent))
			ctx.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
			ctx.WriteString("ET\n")

			curX += measureText(r.Text, r.FontSize)
		}
		if i < len(ln.Words)-1 {
			if align == AlignJustify && !isLast {
				curX += wordSpacing
			} else {
				curX += w.Space
			}
		}
	}
}

func pdfEscapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func ensureFont(ctx *DrawContext, name string, bold, italic bool) string {
	suffix := ""
	if bold {
		suffix += "-Bold"
	}
	if italic {
		suffix += "-Oblique"
	}
	fullName := name + suffix

	if entry, ok := ctx.Fonts[fullName]; ok {
		return entry.PDFName
	}

	pdfName := fmt.Sprintf("F%d", len(ctx.Fonts)+1)
	ctx.Fonts[fullName] = FontEntry{
		PDFName: pdfName,
	}
	return pdfName
}
