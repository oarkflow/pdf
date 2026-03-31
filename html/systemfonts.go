package html

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	pdffont "github.com/oarkflow/pdf/font"
)

// systemFontCache caches loaded system fonts by their resolved file path.
var (
	systemFontCache   = make(map[string]pdffont.Face)
	systemFontCacheMu sync.Mutex
)

// resolveSystemFont attempts to find and load a system font matching the given
// family name. It uses fontconfig (fc-match) on Linux to locate .ttf/.otf files.
// Returns nil if the font cannot be found or loaded.
func resolveSystemFont(family string) pdffont.Face {
	if family == "" {
		return nil
	}

	systemFontCacheMu.Lock()
	defer systemFontCacheMu.Unlock()

	// Check cache by family name.
	cacheKey := strings.ToLower(strings.TrimSpace(family))
	if face, ok := systemFontCache[cacheKey]; ok {
		return face
	}

	path := findSystemFontFile(family)
	if path == "" {
		// Cache nil result to avoid repeated lookups.
		systemFontCache[cacheKey] = nil
		return nil
	}

	// Check if we already loaded this exact path.
	if face, ok := systemFontCache[path]; ok {
		systemFontCache[cacheKey] = face
		return face
	}

	face, err := pdffont.LoadTrueTypeFile(path)
	if err != nil {
		systemFontCache[cacheKey] = nil
		return nil
	}

	systemFontCache[cacheKey] = face
	systemFontCache[path] = face
	return face
}

// findSystemFontFile uses fontconfig (fc-match) to locate a font file for the
// given family name. Returns an empty string if not found.
func findSystemFontFile(family string) string {
	// Try fc-match first (available on most Linux systems).
	if path := fcMatch(family); path != "" {
		return path
	}

	// Fallback: search common font directories.
	return searchCommonFontDirs(family)
}

// fcMatch runs `fc-match` to resolve a font family to a file path.
func fcMatch(family string) string {
	cmd := exec.Command("fc-match", "--format=%{file}", family)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return ""
	}
	// Verify the file exists and is a supported format.
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".ttf" && ext != ".otf" && ext != ".woff" {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// searchCommonFontDirs searches common font directories for a font file matching
// the given family name.
func searchCommonFontDirs(family string) string {
	dirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		filepath.Join(os.Getenv("HOME"), ".local/share/fonts"),
		filepath.Join(os.Getenv("HOME"), ".fonts"),
	}

	// Normalize for file matching.
	normalized := strings.ToLower(strings.ReplaceAll(family, " ", ""))

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		result := walkFontDir(dir, normalized)
		if result != "" {
			return result
		}
	}
	return ""
}

func walkFontDir(dir, normalized string) string {
	var result string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ttf" && ext != ".otf" {
			return nil
		}
		base := strings.ToLower(strings.TrimSuffix(info.Name(), ext))
		base = strings.ReplaceAll(base, " ", "")
		base = strings.ReplaceAll(base, "-", "")
		base = strings.ReplaceAll(base, "_", "")
		if strings.Contains(base, normalized) || strings.Contains(normalized, base) {
			result = path
			return filepath.SkipAll
		}
		return nil
	})
	return result
}
