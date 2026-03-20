package html

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Fetcher handles fetching external resources.
type Fetcher struct {
	BaseURL string
	Client  *http.Client
	Cache   map[string][]byte
}

// NewFetcher creates a new resource fetcher.
func NewFetcher(baseURL string) *Fetcher {
	return &Fetcher{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Cache: make(map[string][]byte),
	}
}

// Fetch fetches a resource by URL, using cache.
func (f *Fetcher) Fetch(rawURL string) ([]byte, error) {
	// Handle data URIs
	if strings.HasPrefix(rawURL, "data:") {
		return parseDataURI(rawURL)
	}

	resolved := f.ResolveURL(rawURL)

	// Check cache
	if data, ok := f.Cache[resolved]; ok {
		return data, nil
	}

	resp, err := f.Client.Get(resolved)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	f.Cache[resolved] = data
	return data, nil
}

// FetchCSS fetches a CSS file and resolves @import directives.
func (f *Fetcher) FetchCSS(rawURL string) (string, error) {
	data, err := f.Fetch(rawURL)
	if err != nil {
		return "", err
	}
	css := string(data)

	// Resolve @import
	resolved := f.ResolveURL(rawURL)
	css = f.resolveImports(css, resolved)

	return css, nil
}

// ResolveURL resolves a relative URL against the base URL.
func (f *Fetcher) ResolveURL(ref string) string {
	if ref == "" {
		return f.BaseURL
	}

	// Already absolute
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref
	}

	if f.BaseURL == "" {
		return ref
	}

	base, err := url.Parse(f.BaseURL)
	if err != nil {
		return ref
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}

	return base.ResolveReference(refURL).String()
}

func (f *Fetcher) resolveImports(css string, baseURL string) string {
	var result strings.Builder
	lines := strings.Split(css, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "@import") {
			// Extract URL from @import
			importURL := extractImportURL(trimmed)
			if importURL != "" {
				// Resolve against the CSS file's URL
				oldBase := f.BaseURL
				f.BaseURL = baseURL
				importedCSS, err := f.FetchCSS(importURL)
				f.BaseURL = oldBase
				if err == nil {
					result.WriteString(importedCSS)
					result.WriteString("\n")
				}
			}
		} else {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}
	return result.String()
}

func extractImportURL(line string) string {
	// @import url("..."); or @import "...";
	line = strings.TrimPrefix(line, "@import")
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(line, ";")
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "url(") {
		line = strings.TrimPrefix(line, "url(")
		line = strings.TrimSuffix(line, ")")
		line = strings.TrimSpace(line)
	}

	line = strings.Trim(line, "\"'")
	return line
}

func parseDataURI(uri string) ([]byte, error) {
	// data:[<mediatype>][;base64],<data>
	rest := strings.TrimPrefix(uri, "data:")
	commaIdx := strings.IndexByte(rest, ',')
	if commaIdx < 0 {
		return []byte(rest), nil
	}

	meta := rest[:commaIdx]
	data := rest[commaIdx+1:]

	if strings.Contains(meta, ";base64") {
		return base64.StdEncoding.DecodeString(data)
	}

	return []byte(data), nil
}
