# PDF Tools Example

This example creates its own sample inputs and demonstrates the higher-level PDF
tooling APIs:

- JSON HTML template filling
- existing PDF form field discovery and filling
- signature QR image placement from `qr-code.png`
- stamp image placement from `logo.png`
- scanned image batch to PDF
- PDF rewrite/compression
- text and region redaction
- dictionary-based translation
- PDF comparison
- print preparation
- archive copy validation
- related PDF graph extraction

Run it from the repository root:

```sh
go run ./examples/pdf_tools
```

To choose an output directory:

```sh
go run ./examples/pdf_tools /tmp/pdf-tools-example
```

Open `agreement-filled.pdf` for the main clean agreement layout. It renders the
JSON-filled names, dates, signature QR images, and stamp images inside the
document flow. `agreement-signed-visual.pdf` additionally demonstrates the
post-generation image stamping API using placement coordinates from the same
JSON file.

Equivalent CLI workflows:

```sh
pdf fill-template -data agreement-data.json -o agreement-filled.pdf agreement.html
pdf form-fields application-form.pdf
pdf fill-form -data application-data.json -o application-filled.pdf application-form.pdf
pdf signature-image -image qr-code.png -page 1 -x 82 -y 144 -width 54 -o agreement-signature-step.pdf agreement-filled.pdf
pdf stamp-image -image logo.png -page 1 -x 82 -y 58 -width 124 -o agreement-signed-visual.pdf agreement-signature-step.pdf
pdf scan -o scanned-images.pdf scan-page.png
pdf compress -o compressed.pdf source.pdf
pdf redact -text Secret -region 1,72,700,100,16 -o redacted.pdf source.pdf
pdf translate -dict translations.json -o translated.pdf source.pdf
pdf compare source.pdf compressed.pdf
pdf print-prep -page-numbers -title "Print Prepared Example" -o print-ready.pdf source.pdf
pdf archive -o archive-copy.pdf source.pdf
pdf graph --json source.pdf agreement-signed-visual.pdf agreement-filled.pdf
```

The Go example writes `agreement-data.json` with `stamp_file`,
`stamp_placement`, `signature_file`, and `signature_placement` fields, then
uses those JSON-defined values to place the visible stamp and QR signature
images.
