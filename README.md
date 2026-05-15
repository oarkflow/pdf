# pdf

`github.com/oarkflow/pdf` is a Go library for generating, reading, merging, signing, and converting PDFs. The HTML renderer is designed for controlled business documents such as invoices, reports, letters, and email-style templates. It is not a full browser engine.

## Quick Start

```go
package main

import "github.com/oarkflow/pdf"

func main() {
	if err := pdf.Quick("Hello PDF", "hello.pdf"); err != nil {
		panic(err)
	}
}
```

Generate a PDF from HTML:

```go
package main

import (
	"os"

	"github.com/oarkflow/pdf"
	"github.com/oarkflow/pdf/html"
)

func main() {
	input, err := os.ReadFile("invoice.html")
	if err != nil {
		panic(err)
	}

	err = pdf.FromHTML(string(input), "invoice.pdf", html.Options{
		DefaultFontSize:   10,
		DefaultFontFamily: "Helvetica",
		PageSize:          [2]float64{595.28, 841.89}, // A4
		Margins:           [4]float64{40, 40, 50, 40},
		BaseURL:           "https://example.com/assets/",
		UseTailwind:       true,
	})
	if err != nil {
		panic(err)
	}
}
```

Streaming output:

```go
var out bytes.Buffer
err := pdf.FromHTMLStreaming("<p>Hello</p>", &out)
```

## HTML To PDF Support

The renderer supports a practical subset of HTML/CSS:

| Area | Supported | Notes |
| --- | --- | --- |
| HTML | `div`, headings, paragraphs, spans, links, lists, tables, images, SVG images, `hr`, `blockquote`, `pre` | Form controls are not rendered as interactive HTML widgets. |
| CSS text | font family/size/weight/style, color, alignment, decoration, line height, spacing, transform, whitespace | Complex browser text shaping is limited; use embedded fonts for multilingual output. |
| CSS box model | margin, padding, borders, radius, background color, simple gradients, overflow clipping | Advanced painting is approximated. |
| Layout | block, inline, inline-block, flex, grid, table | Flex/grid are useful subsets, not full CSS layout engines. |
| Pagination | automatic page flow, basic `@page` size/margins, page break properties | Repeated table headers and advanced running headers/footers are future hardening targets. |
| Assets | data URIs, HTTP(S), relative URLs through `BaseURL`, PNG, JPEG, TIFF, WebP, SVG | Remote fetches use timeouts and size limits. |
| Tailwind | common spacing, sizing, typography, color, border, display, flex/grid, opacity, shadow, ring utilities | Variants are approximated by page width and non-visual states are ignored. |
| JavaScript | optional minimal Goja execution and Alpine-style `x-data`, `x-text`, `x-show`, `x-bind` | No browser DOM, layout engine, canvas, or network runtime parity. |

Use `html.Options` to configure:

- `PageSize` and `Margins` for output dimensions.
- `BaseURL` for external CSS/images.
- `UserStylesheet` for injected CSS.
- `UseTailwind` to parse Tailwind utility classes without a browser.
- `EnableJavaScript` for small template-style scripts.
- `TemplateData` for `fasttpl` rendering before HTML parsing.
- `FontFaces` for embedded/custom fonts and fallback.
- `Encryption` for AES-128/RC4 PDF encryption.

## Production Guidance

For production invoice/report generation:

- Keep HTML templates controlled and covered by tests.
- Prefer local/data URI assets, or set a stable `BaseURL`.
- Register fonts explicitly for non-Latin scripts.
- Treat HTML rendering as deterministic document rendering, not browser screenshotting.
- Run `go test ./...` before releases.

Known limitations:

- AES-256 encrypted output and AES-256 encrypted input are not supported yet.
- JavaScript support is intentionally minimal.
- Some placeholder/stub areas remain in form filling, barcode layout, and SVG image element rendering.
- PDF merge is intended for common page/resource merging and should be validated against your input corpus.

## Examples

- `examples/html_invoice` renders a styled invoice, protected invoice, Tailwind newsletter-style document, and a Devanagari letter.
- `examples/template` demonstrates template-driven documents.
- `cmd/pdftohtml` provides a local PDF-to-HTML inspection UI.

## Development

```sh
go test ./...
go test ./html -run TestConvertTailwind -v
go test ./tailwind -run TestParser_Effects -v
```

The module currently targets Go `1.26.1` because this workspace is using that toolchain.

PASSPORT_PAGE_PADDING="18mm 20mm 18mm 20mm" go run ./examples/html_invoice
