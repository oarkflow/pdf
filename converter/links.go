package converter

import (
	"github.com/oarkflow/pdf/reader"
)

// extractLinks extracts hyperlink annotations from a PDF page dictionary.
func extractLinks(resolver *reader.Resolver, pageDict map[string]interface{}, pageNum int) []ExtractedLink {
	annotsRef, ok := pageDict["/Annots"]
	if !ok {
		return nil
	}

	annotsObj, err := resolver.ResolveReference(annotsRef)
	if err != nil {
		return nil
	}

	annots, ok := annotsObj.([]interface{})
	if !ok {
		return nil
	}

	var links []ExtractedLink
	for _, annotRef := range annots {
		annotObj, err := resolver.ResolveReference(annotRef)
		if err != nil {
			continue
		}
		annotDict, ok := annotObj.(map[string]interface{})
		if !ok {
			continue
		}

		subtype, _ := annotDict["/Subtype"].(string)
		if subtype != "/Link" {
			continue
		}

		// Extract URL from action dictionary.
		var url string
		if actionRef, ok := annotDict["/A"]; ok {
			actionObj, err := resolver.ResolveReference(actionRef)
			if err == nil {
				if actionDict, ok := actionObj.(map[string]interface{}); ok {
					url, _ = actionDict["/URI"].(string)
				}
			}
		}

		if url == "" {
			continue
		}

		// Extract rectangle.
		var rect [4]float64
		if rectRef, ok := annotDict["/Rect"]; ok {
			rectObj, err := resolver.ResolveReference(rectRef)
			if err == nil {
				if arr, ok := rectObj.([]interface{}); ok && len(arr) >= 4 {
					for i := 0; i < 4; i++ {
						rect[i] = reader.ToFloat(arr[i])
					}
				}
			}
		}

		links = append(links, ExtractedLink{
			URL:     url,
			Rect:    rect,
			PageNum: pageNum,
		})
	}

	return links
}

// matchLinksToSpans associates links with text spans based on position overlap.
func matchLinksToSpans(links []ExtractedLink, spans []StyledSpan, pageHeight float64) {
	// Links use PDF coordinates (origin bottom-left). Spans also use PDF Y.
	// A span overlaps a link if its position falls within the link rect.
	for i := range spans {
		for _, link := range links {
			x1, y1, x2, y2 := link.Rect[0], link.Rect[1], link.Rect[2], link.Rect[3]
			sx, sy := spans[i].X, spans[i].Y
			if sx >= x1 && sx <= x2 && sy >= y1 && sy <= y2 {
				// Mark span as linked (we store the URL in a field we'll add).
				// For now, links are tracked at the page level.
				break
			}
		}
	}
}
