package html

import (
	"strings"

	pdffont "github.com/oarkflow/pdf/font"
)

func resolveFontFace(fontFamily string, faces map[string]pdffont.Face) pdffont.Face {
	for _, family := range splitFontFamilies(fontFamily) {
		// Check explicitly provided faces first.
		if len(faces) > 0 {
			if face := lookupFontFace(family, faces); face != nil {
				return face
			}
		}
		// Skip standard PDF font names — they don't need TrueType loading.
		if pdffont.IsStandardFont(family) {
			continue
		}
		// Try to resolve from system fonts as a fallback.
		if face := resolveSystemFont(family); face != nil {
			return face
		}
	}
	return nil
}

// resolveFontFaceWithFallback resolves a font face for the given font family.
// If the primary resolution fails (e.g., standard PDF font that can't handle
// non-Latin text), and the text contains non-ASCII characters, it tries any
// of the explicitly provided font faces as a fallback.
func resolveFontFaceWithFallback(fontFamily string, faces map[string]pdffont.Face, text string) pdffont.Face {
	face := resolveFontFace(fontFamily, faces)
	if face != nil {
		return face
	}
	// If no embedded face found and text contains non-ASCII, try fallback
	if len(faces) > 0 && containsNonASCII(text) {
		// Return the first available embedded face as fallback
		for _, f := range faces {
			if f != nil {
				return f
			}
		}
	}
	return nil
}

// containsNonASCII returns true if the string contains any non-ASCII characters.
func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

func splitFontFamilies(fontFamily string) []string {
	if fontFamily == "" {
		return nil
	}
	parts := strings.Split(fontFamily, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(stripQuotes(strings.TrimSpace(part)))
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func lookupFontFace(name string, faces map[string]pdffont.Face) pdffont.Face {
	if face, ok := faces[name]; ok {
		return face
	}
	normalized := normalizeFontFamilyName(name)
	for key, face := range faces {
		if normalizeFontFamilyName(key) == normalized {
			return face
		}
	}
	return nil
}

func normalizeFontFamilyName(name string) string {
	name = strings.TrimSpace(strings.ToLower(stripQuotes(name)))
	return strings.Join(strings.Fields(name), " ")
}
