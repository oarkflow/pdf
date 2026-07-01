package export

import "github.com/oarkflow/pdf/md/internal/markdown"

type Options struct {
	Title      string
	Author     string
	PageSize   string
	Margin     float64
	Theme      string
	CSS        string
	TOC        bool
	Standalone bool
	SoftHR     bool
}

type Exporter interface {
	Export(markdown.Doc, Options) ([]byte, error)
}
