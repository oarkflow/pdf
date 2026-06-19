# PDF Tools Example

This example creates its own sample inputs and demonstrates the higher-level PDF
tooling APIs:

- JSON HTML template filling
- existing PDF form field discovery and filling
- signature image placement
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

Equivalent CLI workflows:

```sh
pdf fill-template -data offer.json -o offer-filled.pdf offer.html
pdf form-fields application-form.pdf
pdf fill-form -data application-data.json -o application-filled.pdf application-form.pdf
pdf signature-image -image signature.png -page 1 -x 72 -y 96 -width 120 -o signature-image.pdf source.pdf
pdf scan -o scanned-images.pdf scan-page.png
pdf compress -o compressed.pdf source.pdf
pdf redact -text Secret -region 1,72,700,100,16 -o redacted.pdf source.pdf
pdf translate -dict translations.json -o translated.pdf source.pdf
pdf compare source.pdf compressed.pdf
pdf print-prep -page-numbers -title "Print Prepared Example" -o print-ready.pdf source.pdf
pdf archive -o archive-copy.pdf source.pdf
pdf graph --json source.pdf signature-image.pdf offer-filled.pdf
```
