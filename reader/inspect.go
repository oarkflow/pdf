package reader

// AnnotationInfo describes a page annotation.
type AnnotationInfo struct {
	Page    int        `json:"page"`
	Subtype string     `json:"subtype,omitempty"`
	Rect    [4]float64 `json:"rect,omitempty"`
	URI     string     `json:"uri,omitempty"`
	Content string     `json:"content,omitempty"`
}

// OutlineInfo describes a bookmark/outline item.
type OutlineInfo struct {
	Title    string        `json:"title"`
	Page     int           `json:"page,omitempty"`
	Children []OutlineInfo `json:"children,omitempty"`
}

// Annotations returns page annotations in document order.
func (r *Reader) Annotations() []AnnotationInfo {
	var out []AnnotationInfo
	for i, page := range r.pages {
		annotsRef, ok := page["/Annots"]
		if !ok {
			continue
		}
		annotsObj, err := r.resolver.ResolveReference(annotsRef)
		if err != nil {
			continue
		}
		annots, ok := annotsObj.([]interface{})
		if !ok {
			continue
		}
		for _, annotRef := range annots {
			annotObj, err := r.resolver.ResolveReference(annotRef)
			if err != nil {
				continue
			}
			annotDict, ok := annotObj.(map[string]interface{})
			if !ok {
				continue
			}
			info := AnnotationInfo{Page: i + 1}
			if subtype, ok := annotDict["/Subtype"].(string); ok {
				info.Subtype = trimName(subtype)
			}
			if contents, ok := annotDict["/Contents"].(string); ok {
				info.Content = contents
			}
			info.Rect = rectFromValue(r.resolver, annotDict["/Rect"])
			if actionObj, err := r.resolver.ResolveReference(annotDict["/A"]); err == nil {
				if action, ok := actionObj.(map[string]interface{}); ok {
					if uri, ok := action["/URI"].(string); ok {
						info.URI = uri
					}
				}
			}
			out = append(out, info)
		}
	}
	return out
}

// Outlines returns the document bookmark tree.
func (r *Reader) Outlines() []OutlineInfo {
	outlinesObj, err := r.resolver.ResolveReference(r.catalog["/Outlines"])
	if err != nil {
		return nil
	}
	root, ok := outlinesObj.(map[string]interface{})
	if !ok {
		return nil
	}
	first, ok := root["/First"]
	if !ok {
		return nil
	}
	return r.readOutlineSiblings(first)
}

func (r *Reader) readOutlineSiblings(first interface{}) []OutlineInfo {
	var out []OutlineInfo
	seen := make(map[IndirectRef]bool)
	cur := first
	for cur != nil {
		ref, _ := cur.(IndirectRef)
		if ref.ObjNum != 0 {
			if seen[ref] {
				break
			}
			seen[ref] = true
		}
		obj, err := r.resolver.ResolveReference(cur)
		if err != nil {
			break
		}
		dict, ok := obj.(map[string]interface{})
		if !ok {
			break
		}
		item := OutlineInfo{}
		if title, ok := dict["/Title"].(string); ok {
			item.Title = title
		}
		item.Page = r.outlinePage(dict)
		if child, ok := dict["/First"]; ok {
			item.Children = r.readOutlineSiblings(child)
		}
		out = append(out, item)
		next, ok := dict["/Next"]
		if !ok {
			break
		}
		cur = next
	}
	return out
}

func (r *Reader) outlinePage(dict map[string]interface{}) int {
	if dest, ok := dict["/Dest"]; ok {
		return r.pageFromDest(dest)
	}
	if actionObj, err := r.resolver.ResolveReference(dict["/A"]); err == nil {
		if action, ok := actionObj.(map[string]interface{}); ok {
			return r.pageFromDest(action["/D"])
		}
	}
	return 0
}

func (r *Reader) pageFromDest(dest interface{}) int {
	resolved, err := r.resolver.ResolveReference(dest)
	if err != nil {
		return 0
	}
	arr, ok := resolved.([]interface{})
	if !ok || len(arr) == 0 {
		return 0
	}
	ref, ok := arr[0].(IndirectRef)
	if !ok {
		return 0
	}
	for i, pageRef := range r.pageRefs {
		if pageRef == ref {
			return i + 1
		}
	}
	return 0
}

func rectFromValue(resolver *Resolver, v interface{}) [4]float64 {
	var rect [4]float64
	resolved, err := resolver.ResolveReference(v)
	if err != nil {
		return rect
	}
	arr, ok := resolved.([]interface{})
	if !ok {
		return rect
	}
	for i := 0; i < len(arr) && i < 4; i++ {
		rect[i] = ToFloat(arr[i])
	}
	return rect
}

func trimName(s string) string {
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}
