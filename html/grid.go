package html

import (
	"fmt"
	"strings"

	"github.com/oarkflow/pdf/layout"
)

func convertFlexContainer(node *Node, style *ComputedStyle, c *converter) layout.Element {
	inlineChildren := collectAnonymousInlineChildren(node, c)
	var children []FlexChildElement
	for _, child := range inlineChildren {
		children = append(children, FlexChildElement{
			Element: child.Element,
			Style:   child.Style,
		})
	}
	return &FlexContainerElement{
		Children: children,
		Style:    style,
		BoxModel: c.computeBoxModel(style),
	}
}

type anonymousChild struct {
	Element layout.Element
	Style   *ComputedStyle
}

func collectAnonymousInlineChildren(node *Node, c *converter) []anonymousChild {
	var children []anonymousChild
	var currentRuns []layout.TextRun

	flushRuns := func() {
		if len(currentRuns) == 0 {
			return
		}
		children = append(children, anonymousChild{
			Element: &ParagraphElement{
				Runs:  currentRuns,
				Style: node.Style,
			},
			Style: node.Style,
		})
		currentRuns = nil
	}

	for _, child := range node.Children {
		if child.IsText() {
			text := child.Text
			if strings.TrimSpace(text) == "" {
				if len(currentRuns) > 0 && strings.ContainsAny(text, " \t\n\r") {
					currentRuns = append(currentRuns, layout.TextRun{
						Text:     " ",
						FontName: node.Style.FontFamily,
						FontSize: node.Style.FontSize,
						Color:    node.Style.Color,
					})
				}
				continue
			}
			currentRuns = append(currentRuns, c.textRunsFromText(child)...)
			continue
		}

		childStyle := child.Style
		if childStyle == nil {
			childStyle = NewDefaultStyle()
		}

		if child.Tag == "img" {
			flushRuns()
			if el := c.convertNode(child); el != nil {
				children = append(children, anonymousChild{Element: el, Style: child.Style})
			}
			continue
		}

		if childStyle.Display == "inline-block" && shouldRenderInlineBlockAsElement(child, childStyle) {
			flushRuns()
			if el := c.convertInlineBox(child, node.Style); el != nil {
				children = append(children, anonymousChild{Element: el, Style: child.Style})
			}
			continue
		}

		if isInlineTag(child.Tag) || isInlineDisplay(childStyle.Display) {
			if childStyle.MarginLeft.Value > 0 || childStyle.MarginLeft.Unit != "" {
				ml := childStyle.MarginLeft.ToPoints(childStyle.FontSize, c.rootFontSize)
				if ml > 0 {
					spaceWidth := childStyle.FontSize * 0.25
					if spaceWidth > 0 {
						numSpaces := int(ml/spaceWidth + 0.5)
						if numSpaces < 1 {
							numSpaces = 1
						}
						currentRuns = append(currentRuns, layout.TextRun{
							Text:     strings.Repeat(" ", numSpaces),
							FontName: childStyle.FontFamily,
							FontSize: childStyle.FontSize,
							Color:    childStyle.Color,
						})
					}
				}
			}
			currentRuns = append(currentRuns, c.collectTextRuns(child)...)
			continue
		}

		flushRuns()
		if el := c.convertNode(child); el != nil {
			children = append(children, anonymousChild{
				Element: el,
				Style:   child.Style,
			})
		}
	}

	flushRuns()
	return children
}

func convertGridContainer(node *Node, style *ComputedStyle, c *converter) layout.Element {
	var children []layout.Element
	for _, child := range collectAnonymousInlineChildren(node, c) {
		children = append(children, child.Element)
	}
	columns := parseGridTemplate(style.GridTemplateColumns)
	rows := parseGridTemplate(style.GridTemplateRows)
	if len(columns) == 0 {
		columns = []GridTrack{{Auto: true}}
	}
	return &GridContainerElement{
		Children: children,
		Style:    style,
		BoxModel: c.computeBoxModel(style),
		Columns:  columns,
		Rows:     rows,
	}
}

func parseGridTemplate(template string) []GridTrack {
	if template == "" {
		return nil
	}

	// Handle repeat(N, track) syntax
	if strings.HasPrefix(template, "repeat(") {
		inner := template[7:]
		if idx := strings.LastIndex(inner, ")"); idx >= 0 {
			inner = inner[:idx]
		}
		// Split into count and track definition
		commaIdx := strings.Index(inner, ",")
		if commaIdx >= 0 {
			countStr := strings.TrimSpace(inner[:commaIdx])
			trackDef := strings.TrimSpace(inner[commaIdx+1:])
			count := 1
			fmt.Sscanf(countStr, "%d", &count)
			// Parse the track definition (handle minmax)
			track := parseGridTrackValue(trackDef)
			var tracks []GridTrack
			for i := 0; i < count; i++ {
				tracks = append(tracks, track)
			}
			return tracks
		}
	}

	parts := splitCSSValues(template)
	var tracks []GridTrack
	for _, p := range parts {
		tracks = append(tracks, parseGridTrackValue(p))
	}
	return tracks
}

func parseGridTrackValue(p string) GridTrack {
	p = strings.TrimSpace(p)
	if p == "auto" {
		return GridTrack{Auto: true}
	}
	// Handle minmax(min, max) — use the max value
	if strings.HasPrefix(p, "minmax(") {
		inner := p[7:]
		if idx := strings.LastIndex(inner, ")"); idx >= 0 {
			inner = inner[:idx]
		}
		parts := strings.SplitN(inner, ",", 2)
		if len(parts) == 2 {
			maxVal := strings.TrimSpace(parts[1])
			return parseGridTrackValue(maxVal)
		}
	}
	if len(p) > 2 && p[len(p)-2:] == "fr" {
		val := 1.0
		fmt.Sscanf(p[:len(p)-2], "%f", &val)
		return GridTrack{Fr: val}
	}
	l := parseLength(p)
	return GridTrack{Size: l}
}
