package pdf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/oarkflow/pdf/converter"
	"github.com/oarkflow/pdf/document"
	pdfimage "github.com/oarkflow/pdf/image"
	"github.com/oarkflow/pdf/layout"
	"github.com/oarkflow/pdf/reader"
)

// MergeOptions controls mixed PDF/image merging.
type MergeOptions struct {
	Output  string           `json:"output"`
	Page    MergePageOptions `json:"page,omitempty"`
	Inputs  []MergeInput     `json:"inputs"`
	BaseDir string           `json:"-"`
}

// MergePageOptions controls pages created for image inputs.
type MergePageOptions struct {
	Size     string  `json:"size,omitempty"`
	Width    float64 `json:"width,omitempty"`
	Height   float64 `json:"height,omitempty"`
	Margin   float64 `json:"margin,omitempty"`
	ImageFit string  `json:"imageFit,omitempty"`
}

// MergeInput describes one file or directory input for mixed merging.
type MergeInput struct {
	Path      string   `json:"path,omitempty"`
	Directory string   `json:"directory,omitempty"`
	Type      string   `json:"type,omitempty"`
	Include   []string `json:"include,omitempty"`
	Sort      string   `json:"sort,omitempty"`
	Sequence  []string `json:"sequence,omitempty"`
}

// SplitConfig describes one input PDF copied into one or more output PDFs.
type SplitConfig struct {
	Input    string        `json:"input"`
	Password string        `json:"password,omitempty"`
	Outputs  []SplitOutput `json:"outputs"`
	BaseDir  string        `json:"-"`
}

// SplitOutput describes one page selection written to one output PDF.
type SplitOutput struct {
	Output string `json:"output"`
	Pages  string `json:"pages,omitempty"`
}

// ImagePDFOptions controls direct image-to-PDF conversion.
type ImagePDFOptions struct {
	Output string           `json:"output"`
	Page   MergePageOptions `json:"page,omitempty"`
	Inputs []MergeInput     `json:"inputs"`
}

// SearchOptions controls PDF text search.
type SearchOptions struct {
	Query         string `json:"query"`
	CaseSensitive bool   `json:"caseSensitive,omitempty"`
	Password      string `json:"password,omitempty"`
}

// SearchMatch describes one text match in a PDF page.
type SearchMatch struct {
	Page    int    `json:"page"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Text    string `json:"text"`
	Context string `json:"context,omitempty"`
}

// ValidationResult describes basic PDF validation results.
type ValidationResult struct {
	Path      string   `json:"path,omitempty"`
	Valid     bool     `json:"valid"`
	Encrypted bool     `json:"encrypted"`
	Pages     int      `json:"pages"`
	Warnings  []string `json:"warnings,omitempty"`
	Error     string   `json:"error,omitempty"`
}

type mergeSource struct {
	path string
	typ  string
}

// MergeWithOptions merges PDFs and images into a single output PDF.
func MergeWithOptions(opts MergeOptions) error {
	if strings.TrimSpace(opts.Output) == "" {
		return errors.New("pdf: output path is empty")
	}
	if len(opts.Inputs) == 0 {
		return errors.New("pdf: no merge inputs provided")
	}

	sources, err := expandMergeInputs(opts)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return errors.New("pdf: no merge input files matched")
	}

	w := document.NewWriter()
	pageSize := mergePageSize(opts.Page)
	margin := opts.Page.Margin
	for idx, source := range sources {
		data, err := os.ReadFile(source.path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", source.path, err)
		}
		switch source.typ {
		case "pdf":
			if err := reader.AppendPDF(w, data, fmt.Sprintf("src%d", idx)); err != nil {
				return fmt.Errorf("merging %s: %w", source.path, err)
			}
		case "image":
			page, err := imageMergePage(data, pageSize, margin, opts.Page.ImageFit)
			if err != nil {
				return fmt.Errorf("adding image %s: %w", source.path, err)
			}
			if _, err := w.AddPage(page); err != nil {
				return fmt.Errorf("adding image page %s: %w", source.path, err)
			}
		default:
			return fmt.Errorf("unsupported merge input type %q for %s", source.typ, source.path)
		}
	}

	var buf bytes.Buffer
	if _, err := w.WriteTo(&buf); err != nil {
		return fmt.Errorf("writing merged PDF: %w", err)
	}
	return writeOutputFile(resolvePath(opts.BaseDir, opts.Output), buf.Bytes())
}

// MergeWithConfig loads mixed merge options from configPath and runs the merge.
func MergeWithConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var opts MergeOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return fmt.Errorf("parsing merge config: %w", err)
	}
	opts.BaseDir = filepath.Dir(configPath)
	return MergeWithOptions(opts)
}

// ImagesToPDF writes image inputs into one PDF.
func ImagesToPDF(opts ImagePDFOptions) error {
	if strings.TrimSpace(opts.Output) == "" {
		return errors.New("pdf: output path is empty")
	}
	if len(opts.Inputs) == 0 {
		return errors.New("pdf: no image inputs provided")
	}
	inputs := make([]MergeInput, len(opts.Inputs))
	for i, input := range opts.Inputs {
		input.Type = "image"
		if input.Directory != "" && len(input.Include) == 0 {
			input.Include = defaultImageIncludes()
		}
		inputs[i] = input
	}
	return MergeWithOptions(MergeOptions{
		Output: opts.Output,
		Page:   opts.Page,
		Inputs: inputs,
	})
}

// SearchText finds text matches across PDF pages.
func SearchText(inputPath string, opts SearchOptions) ([]SearchMatch, error) {
	if inputPath == "" {
		return nil, errors.New("pdf: input path is empty")
	}
	if strings.TrimSpace(opts.Query) == "" {
		return nil, errors.New("pdf: search query is empty")
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}
	r, err := reader.OpenWithPassword(data, opts.Password)
	if err != nil {
		return nil, err
	}
	needle := opts.Query
	if !opts.CaseSensitive {
		needle = strings.ToLower(needle)
	}
	var matches []SearchMatch
	for page := 0; page < r.NumPages(); page++ {
		text, err := r.ExtractText(page)
		if err != nil {
			return nil, fmt.Errorf("extracting page %d text: %w", page+1, err)
		}
		for lineIdx, line := range strings.Split(text, "\n") {
			haystack := line
			if !opts.CaseSensitive {
				haystack = strings.ToLower(haystack)
			}
			offset := 0
			for {
				idx := strings.Index(haystack[offset:], needle)
				if idx < 0 {
					break
				}
				col := offset + idx
				matches = append(matches, SearchMatch{
					Page:    page + 1,
					Line:    lineIdx + 1,
					Column:  col + 1,
					Text:    line,
					Context: searchContext(line, col, len(opts.Query)),
				})
				offset = col + len(needle)
			}
		}
	}
	return matches, nil
}

// ValidatePDF performs basic structural validation and page-read checks.
func ValidatePDF(inputPath, password string) ValidationResult {
	report := ValidateCompliance(inputPath, ComplianceOptions{
		Profiles: []ComplianceProfile{ProfilePDF},
		Password: password,
	})
	result := ValidationResult{
		Path:      inputPath,
		Valid:     report.Valid,
		Encrypted: report.Encrypted,
		Pages:     report.Pages,
		Error:     report.Error,
	}
	for _, issue := range report.Issues {
		msg := issue.Message
		if issue.Page > 0 {
			msg = fmt.Sprintf("page %d: %s", issue.Page, msg)
		}
		if issue.Severity == IssueError && result.Error == "" {
			result.Error = msg
			result.Valid = false
			continue
		}
		result.Warnings = append(result.Warnings, msg)
	}
	return result
}

// SplitWithConfig loads a split config and writes all configured outputs.
func SplitWithConfig(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cfg SplitConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing split config: %w", err)
	}
	cfg.BaseDir = filepath.Dir(configPath)
	return SplitWithOptions(cfg)
}

// SplitWithOptions writes each configured page selection to its output PDF.
func SplitWithOptions(cfg SplitConfig) error {
	if strings.TrimSpace(cfg.Input) == "" {
		return errors.New("pdf: split input path is empty")
	}
	if len(cfg.Outputs) == 0 {
		return errors.New("pdf: no split outputs provided")
	}
	input := resolvePath(cfg.BaseDir, cfg.Input)
	for _, output := range cfg.Outputs {
		if strings.TrimSpace(output.Output) == "" {
			return errors.New("pdf: split output path is empty")
		}
		pages, err := ParsePageSpec(output.Pages)
		if err != nil {
			return fmt.Errorf("invalid pages for %s: %w", output.Output, err)
		}
		outputPath := resolvePath(cfg.BaseDir, output.Output)
		if err := ensureOutputDir(outputPath); err != nil {
			return fmt.Errorf("creating output directory for %s: %w", output.Output, err)
		}
		if err := Split(input, outputPath, pages, converter.ConvertOptions{Password: cfg.Password}); err != nil {
			return fmt.Errorf("splitting to %s: %w", output.Output, err)
		}
	}
	return nil
}

// ParsePageSpec parses a 1-based page spec such as "1-3,5" into 0-based indexes.
func ParsePageSpec(spec string) ([]int, error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil
	}
	seen := make(map[int]bool)
	var pages []int
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil || start < 1 {
				return nil, fmt.Errorf("invalid start page %q", bounds[0])
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil || end < start {
				return nil, fmt.Errorf("invalid end page %q", bounds[1])
			}
			for p := start; p <= end; p++ {
				idx := p - 1
				if !seen[idx] {
					seen[idx] = true
					pages = append(pages, idx)
				}
			}
			continue
		}
		p, err := strconv.Atoi(part)
		if err != nil || p < 1 {
			return nil, fmt.Errorf("invalid page %q", part)
		}
		idx := p - 1
		if !seen[idx] {
			seen[idx] = true
			pages = append(pages, idx)
		}
	}
	return pages, nil
}

func expandMergeInputs(opts MergeOptions) ([]mergeSource, error) {
	var sources []mergeSource
	for _, input := range opts.Inputs {
		if input.Path != "" && input.Directory != "" {
			return nil, errors.New("pdf: merge input cannot set both path and directory")
		}
		if input.Path != "" {
			path := resolvePath(opts.BaseDir, input.Path)
			typ, err := detectMergeType(path, input.Type)
			if err != nil {
				return nil, err
			}
			sources = append(sources, mergeSource{path: path, typ: typ})
			continue
		}
		if input.Directory == "" {
			return nil, errors.New("pdf: merge input requires path or directory")
		}
		dirSources, err := expandMergeDirectory(opts.BaseDir, input)
		if err != nil {
			return nil, err
		}
		sources = append(sources, dirSources...)
	}
	return sources, nil
}

func expandMergeDirectory(baseDir string, input MergeInput) ([]mergeSource, error) {
	dir := resolvePath(baseDir, input.Directory)
	if len(input.Sequence) > 0 {
		var sources []mergeSource
		for _, name := range input.Sequence {
			path := resolvePath(dir, name)
			typ, err := detectMergeType(path, input.Type)
			if err != nil {
				return nil, err
			}
			sources = append(sources, mergeSource{path: path, typ: typ})
		}
		return sources, nil
	}

	include := input.Include
	if len(include) == 0 {
		include = defaultMergeIncludes()
	}
	var paths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if matchesAny(filepath.Base(path), include) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sortMode := strings.ToLower(strings.TrimSpace(input.Sort))
	if sortMode == "" || sortMode == "name" {
		sort.Strings(paths)
	} else if sortMode != "none" {
		return nil, fmt.Errorf("unsupported directory sort %q", input.Sort)
	}

	var sources []mergeSource
	for _, path := range paths {
		typ, err := detectMergeType(path, input.Type)
		if err != nil {
			return nil, err
		}
		sources = append(sources, mergeSource{path: path, typ: typ})
	}
	return sources, nil
}

func imageMergePage(data []byte, size document.PageSize, margin float64, fit string) (*document.Page, error) {
	img, err := pdfimage.Load(data)
	if err != nil {
		return nil, err
	}
	page := document.NewPage(size)
	boxX := margin
	boxY := margin
	boxW := size.Width - 2*margin
	boxH := size.Height - 2*margin
	if boxW <= 0 || boxH <= 0 {
		return nil, errors.New("image page margin leaves no drawable area")
	}
	drawW, drawH := imageDrawSize(float64(img.Width), float64(img.Height), boxW, boxH, fit)
	x := boxX + (boxW-drawW)/2
	y := boxY + (boxH-drawH)/2
	page.Images["Im1"] = layout.ImageEntry{Image: img, Width: img.Width, Height: img.Height}
	page.Contents = []byte(fmt.Sprintf("q %.2f 0 0 %.2f %.2f %.2f cm /Im1 Do Q\n", drawW, drawH, x, y))
	return page, nil
}

func imageDrawSize(imgW, imgH, boxW, boxH float64, fit string) (float64, float64) {
	switch strings.ToLower(strings.TrimSpace(fit)) {
	case "", "contain":
		scale := math.Min(boxW/imgW, boxH/imgH)
		return imgW * scale, imgH * scale
	case "cover":
		scale := math.Max(boxW/imgW, boxH/imgH)
		return imgW * scale, imgH * scale
	case "fill":
		return boxW, boxH
	case "none":
		return imgW, imgH
	default:
		scale := math.Min(boxW/imgW, boxH/imgH)
		return imgW * scale, imgH * scale
	}
}

func mergePageSize(opts MergePageOptions) document.PageSize {
	if opts.Width > 0 && opts.Height > 0 {
		return document.PageSize{Width: opts.Width, Height: opts.Height}
	}
	switch strings.ToLower(strings.TrimSpace(opts.Size)) {
	case "a3":
		return document.A3
	case "a5":
		return document.A5
	case "letter":
		return document.Letter
	case "legal":
		return document.Legal
	default:
		return document.A4
	}
}

func detectMergeType(path, typ string) (string, error) {
	typ = strings.ToLower(strings.TrimSpace(typ))
	if typ == "" || typ == "auto" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".pdf":
			typ = "pdf"
		case ".png", ".jpg", ".jpeg", ".tif", ".tiff", ".webp", ".gif":
			typ = "image"
		default:
			return "", fmt.Errorf("unsupported merge input extension for %s", path)
		}
	}
	if typ != "pdf" && typ != "image" {
		return "", fmt.Errorf("unsupported merge input type %q for %s", typ, path)
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("input file %s: %w", path, err)
	}
	return typ, nil
}

func matchesAny(name string, patterns []string) bool {
	name = strings.ToLower(name)
	for _, pattern := range patterns {
		pattern = strings.ToLower(pattern)
		ok, err := filepath.Match(pattern, name)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func defaultMergeIncludes() []string {
	return []string{"*.pdf", "*.png", "*.jpg", "*.jpeg", "*.tif", "*.tiff", "*.webp", "*.gif"}
}

func defaultImageIncludes() []string {
	return []string{"*.png", "*.jpg", "*.jpeg", "*.tif", "*.tiff", "*.webp", "*.gif"}
}

func searchContext(line string, col, length int) string {
	start := col - 40
	if start < 0 {
		start = 0
	}
	end := col + length + 40
	if end > len(line) {
		end = len(line)
	}
	return strings.TrimSpace(line[start:end])
}

func resolvePath(baseDir, path string) string {
	if path == "" || filepath.IsAbs(path) || baseDir == "" {
		return path
	}
	return filepath.Join(baseDir, path)
}

func writeOutputFile(path string, data []byte) error {
	if err := ensureOutputDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ensureOutputDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}
