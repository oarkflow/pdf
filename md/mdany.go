package md

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/oarkflow/pdf/md/internal/export"
	"github.com/oarkflow/pdf/md/internal/markdown"
)

type Format string

const (
	PDF  Format = "pdf"
	DOCX Format = "docx"
	HTML Format = "html"
)

type Options struct {
	Title, Author, PageSize string
	Margin                  float64
	MaxLineBytes            int
	Theme                   string
	CSS                     string
	TOC                     bool
	Standalone              bool
	SoftHR                  bool
}

func Convert(input []byte, format Format, opt Options) ([]byte, error) {
	return ConvertReader(bytes.NewReader(input), format, opt)
}

func ConvertReader(r io.Reader, format Format, opt Options) ([]byte, error) {
	d, err := markdown.NewParser(markdown.Options{MaxLineBytes: opt.MaxLineBytes}).Parse(r)
	if err != nil {
		return nil, err
	}
	eo := export.Options{Title: opt.Title, Author: opt.Author, PageSize: opt.PageSize, Margin: opt.Margin, Theme: opt.Theme, CSS: opt.CSS, TOC: opt.TOC, Standalone: opt.Standalone, SoftHR: opt.SoftHR}
	switch Format(strings.ToLower(string(format))) {
	case PDF:
		return export.PDF{}.Export(d, eo)
	case DOCX:
		return export.DOCX{}.Export(d, eo)
	case HTML:
		return export.HTML{}.Export(d, eo)
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}
