package export

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/oarkflow/pdf/md/internal/markdown"
)

type DOCX struct{}

func (DOCX) Export(d markdown.Doc, o Options) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	add := func(name, body string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(body))
		return err
	}
	if err := add("[Content_Types].xml", contentTypes); err != nil {
		return nil, err
	}
	if err := add("_rels/.rels", rels); err != nil {
		return nil, err
	}
	if err := add("docProps/core.xml", coreXML(o)); err != nil {
		return nil, err
	}
	if err := add("docProps/app.xml", appXML); err != nil {
		return nil, err
	}
	if err := add("word/_rels/document.xml.rels", docRels); err != nil {
		return nil, err
	}
	if err := add("word/styles.xml", stylesXML); err != nil {
		return nil, err
	}
	if err := add("word/numbering.xml", numberingXML); err != nil {
		return nil, err
	}
	if err := add("word/document.xml", documentXML(d)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func documentXML(d markdown.Doc) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, n := range d.Nodes {
		writeDocxNode(&b, n)
	}
	b.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr></w:body></w:document>`)
	return b.String()
}
func writeDocxNode(b *strings.Builder, n markdown.Node) {
	switch n.Kind {
	case markdown.Heading:
		style := fmt.Sprintf("Heading%d", n.Level)
		if n.Level < 1 || n.Level > 6 {
			style = "Heading1"
		}
		para(b, style, plainInline(n.Text))
	case markdown.Paragraph:
		para(b, "Normal", plainInline(n.Text))
	case markdown.BlockQuote:
		para(b, "Quote", plainInline(n.Text))
	case markdown.Alert:
		para(b, "Quote", strings.ToUpper(defaultStr(n.Info, "note"))+": "+plainInline(n.Text))
	case markdown.HorizontalRule:
		// Soft section separator: do not render a visible line by default.
	case markdown.CodeBlock:
		for _, ln := range strings.Split(n.Text, "\n") {
			para(b, "Code", ln)
		}
	case markdown.BulletList:
		for _, c := range n.Children {
			listPara(b, "ListBullet", 1, taskPrefix(c.Checked)+plainInline(c.Text))
		}
	case markdown.OrderedList:
		for _, c := range n.Children {
			listPara(b, "ListNumber", 2, plainInline(c.Text))
		}
	case markdown.Table:
		table(b, n.Rows)
	}
}
func para(b *strings.Builder, style, text string) {
	b.WriteString(`<w:p><w:pPr><w:pStyle w:val="`)
	b.WriteString(style)
	b.WriteString(`"/></w:pPr><w:r><w:t xml:space="preserve">`)
	xml.EscapeText((*builderWriter)(b), []byte(text))
	b.WriteString(`</w:t></w:r></w:p>`)
}

func listPara(b *strings.Builder, style string, numID int, text string) {
	b.WriteString(`<w:p><w:pPr><w:pStyle w:val="`)
	b.WriteString(style)
	b.WriteString(`"/><w:numPr><w:ilvl w:val="0"/><w:numId w:val="`)
	b.WriteString(fmt.Sprint(numID))
	b.WriteString(`"/></w:numPr></w:pPr><w:r><w:t xml:space="preserve">`)
	xml.EscapeText((*builderWriter)(b), []byte(text))
	b.WriteString(`</w:t></w:r></w:p>`)
}
func horizontalRule(b *strings.Builder) {
	b.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="D8DEE4"/></w:pBdr><w:spacing w:before="120" w:after="120"/></w:pPr></w:p>`)
}

func table(b *strings.Builder, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	cols := 1
	for _, row := range rows {
		if len(row) > cols {
			cols = len(row)
		}
	}
	cellW := 9360 / cols
	if cellW < 1200 {
		cellW = 1200
	}
	b.WriteString(`<w:tbl><w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="0" w:type="auto"/><w:tblLook w:firstRow="1" w:noHBand="0" w:noVBand="1"/><w:tblBorders><w:top w:val="single" w:sz="6" w:color="CBD5E1"/><w:left w:val="single" w:sz="6" w:color="CBD5E1"/><w:bottom w:val="single" w:sz="6" w:color="CBD5E1"/><w:right w:val="single" w:sz="6" w:color="CBD5E1"/><w:insideH w:val="single" w:sz="4" w:color="E2E8F0"/><w:insideV w:val="single" w:sz="4" w:color="E2E8F0"/></w:tblBorders><w:tblCellMar><w:top w:w="120" w:type="dxa"/><w:left w:w="120" w:type="dxa"/><w:bottom w:w="120" w:type="dxa"/><w:right w:w="120" w:type="dxa"/></w:tblCellMar></w:tblPr>`)
	for r, row := range rows {
		b.WriteString(`<w:tr>`)
		if r == 0 {
			b.WriteString(`<w:trPr><w:tblHeader/></w:trPr>`)
		}
		for _, cell := range row {
			fill := "FFFFFF"
			if r == 0 {
				fill = "E2E8F0"
			} else if r%2 == 0 {
				fill = "F8FAFC"
			}
			b.WriteString(`<w:tc><w:tcPr><w:tcW w:w="`)
			b.WriteString(fmt.Sprint(cellW))
			b.WriteString(`" w:type="dxa"/><w:shd w:fill="`)
			b.WriteString(fill)
			b.WriteString(`"/></w:tcPr>`)
			if r == 0 {
				para(b, "TableHeader", plainInline(cell))
			} else {
				para(b, "Normal", plainInline(cell))
			}
			b.WriteString(`</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
}

func taskPrefix(v *bool) string {
	if v == nil {
		return ""
	}
	if *v {
		return "[x] "
	}
	return "[ ] "
}

type builderWriter strings.Builder

func (w *builderWriter) Write(p []byte) (int, error) {
	(*strings.Builder)(w).Write(p)
	return len(p), nil
}

func coreXML(o Options) string {
	title := o.Title
	if title == "" {
		title = "Markdown Document"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:dcmitype="http://purl.org/dc/dcmitype/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>%s</dc:title><dc:creator>%s</dc:creator><cp:lastModifiedBy>%s</cp:lastModifiedBy><dcterms:created xsi:type="dcterms:W3CDTF">%s</dcterms:created><dcterms:modified xsi:type="dcterms:W3CDTF">%s</dcterms:modified></cp:coreProperties>`, esc(title), esc(defaultStr(o.Author, "mdany")), esc(defaultStr(o.Author, "mdany")), time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339))
}
func esc(s string) string { var b bytes.Buffer; xml.EscapeText(&b, []byte(s)); return b.String() }
func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/><Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/></Types>`
const rels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
const docRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/></Relationships>`
const appXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>mdany</Application></Properties>`
const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/><w:rPr><w:sz w:val="22"/></w:rPr><w:pPr><w:spacing w:after="160"/></w:pPr></w:style><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:rPr><w:b/><w:sz w:val="36"/></w:rPr><w:pPr><w:spacing w:before="360" w:after="160"/></w:pPr></w:style><w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:qFormat/><w:rPr><w:b/><w:sz w:val="30"/></w:rPr><w:pPr><w:spacing w:before="300" w:after="140"/></w:pPr></w:style><w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="heading 3"/><w:basedOn w:val="Normal"/><w:qFormat/><w:rPr><w:b/><w:sz w:val="26"/></w:rPr><w:pPr><w:spacing w:before="240" w:after="120"/></w:pPr></w:style><w:style w:type="paragraph" w:styleId="Heading4"><w:name w:val="heading 4"/><w:basedOn w:val="Normal"/><w:qFormat/><w:rPr><w:b/><w:sz w:val="24"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading5"><w:name w:val="heading 5"/><w:basedOn w:val="Normal"/><w:qFormat/><w:rPr><w:b/><w:sz w:val="22"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading6"><w:name w:val="heading 6"/><w:basedOn w:val="Normal"/><w:qFormat/><w:rPr><w:b/><w:sz w:val="20"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:left="720"/><w:pBdr><w:left w:val="single" w:sz="12" w:space="4"/></w:pBdr></w:pPr><w:rPr><w:i/><w:color w:val="666666"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Code"><w:name w:val="Code"/><w:basedOn w:val="Normal"/><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/><w:sz w:val="18"/></w:rPr><w:pPr><w:shd w:fill="F6F8FA"/></w:pPr></w:style><w:style w:type="paragraph" w:styleId="ListBullet"><w:name w:val="List Bullet"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr></w:style><w:style w:type="paragraph" w:styleId="ListNumber"><w:name w:val="List Number"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr></w:style><w:style w:type="table" w:styleId="TableGrid"><w:name w:val="Table Grid"/><w:tblPr><w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders></w:tblPr></w:style><w:style w:type="paragraph" w:styleId="TableHeader"><w:name w:val="Table Header"/><w:basedOn w:val="Normal"/><w:rPr><w:b/><w:color w:val="0F172A"/></w:rPr><w:pPr><w:spacing w:after="40"/></w:pPr></w:style></w:styles>`

const numberingXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:abstractNum w:abstractNumId="1"><w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="•"/><w:lvlJc w:val="left"/><w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr><w:rPr><w:rFonts w:ascii="Symbol" w:hAnsi="Symbol" w:hint="default"/></w:rPr></w:lvl></w:abstractNum><w:abstractNum w:abstractNumId="2"><w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="decimal"/><w:lvlText w:val="%1."/><w:lvlJc w:val="left"/><w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr></w:lvl></w:abstractNum><w:num w:numId="1"><w:abstractNumId w:val="1"/></w:num><w:num w:numId="2"><w:abstractNumId w:val="2"/></w:num></w:numbering>`
