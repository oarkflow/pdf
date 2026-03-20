package document

import (
	"fmt"

	"github.com/oarkflow/pdf/core"
)

// buildInfoDict creates a PDF info dictionary from Metadata.
func buildInfoDict(meta Metadata) *core.PdfDictionary {
	d := core.NewDictionary()
	if meta.Title != "" {
		d.Set("Title", core.PdfString(meta.Title))
	}
	if meta.Author != "" {
		d.Set("Author", core.PdfString(meta.Author))
	}
	if meta.Subject != "" {
		d.Set("Subject", core.PdfString(meta.Subject))
	}
	if meta.Keywords != "" {
		d.Set("Keywords", core.PdfString(meta.Keywords))
	}
	if meta.Creator != "" {
		d.Set("Creator", core.PdfString(meta.Creator))
	}
	if meta.Producer != "" {
		d.Set("Producer", core.PdfString(meta.Producer))
	}
	return d
}

// buildXMPMetadata generates an XMP XML metadata packet from Metadata.
func buildXMPMetadata(meta Metadata) []byte {
	xmp := `<?xpacket begin="\xEF\xBB\xBF" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
<rdf:Description rdf:about=""
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:xmp="http://ns.adobe.com/xap/1.0/"
  xmlns:pdf="http://ns.adobe.com/pdf/1.3/">
`
	if meta.Title != "" {
		xmp += fmt.Sprintf("  <dc:title><rdf:Alt><rdf:li xml:lang=\"x-default\">%s</rdf:li></rdf:Alt></dc:title>\n", meta.Title)
	}
	if meta.Author != "" {
		xmp += fmt.Sprintf("  <dc:creator><rdf:Seq><rdf:li>%s</rdf:li></rdf:Seq></dc:creator>\n", meta.Author)
	}
	if meta.Subject != "" {
		xmp += fmt.Sprintf("  <dc:description><rdf:Alt><rdf:li xml:lang=\"x-default\">%s</rdf:li></rdf:Alt></dc:description>\n", meta.Subject)
	}
	if meta.Keywords != "" {
		xmp += fmt.Sprintf("  <pdf:Keywords>%s</pdf:Keywords>\n", meta.Keywords)
	}
	if meta.Creator != "" {
		xmp += fmt.Sprintf("  <xmp:CreatorTool>%s</xmp:CreatorTool>\n", meta.Creator)
	}
	if meta.Producer != "" {
		xmp += fmt.Sprintf("  <pdf:Producer>%s</pdf:Producer>\n", meta.Producer)
	}
	xmp += `</rdf:Description>
</rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
	return []byte(xmp)
}
