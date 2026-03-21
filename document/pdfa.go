package document

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/oarkflow/pdf/core"
)

// buildPDFAObjects adds PDF/A conformance objects (XMP metadata, output intent
// with sRGB ICC profile, document ID) to the writer.
func (d *Document) buildPDFAObjects(w *Writer) error {
	if d.pdfaLevel == nil {
		return nil
	}

	part, conformance := parsePDFALevel(*d.pdfaLevel)

	// 1. XMP metadata stream (must NOT be compressed per PDF/A).
	xmp := buildPDFAXMPMetadata(d.metadata, part, conformance)
	xmpStream := core.NewStream(xmp)
	xmpStream.Dictionary.Set("Type", core.PdfName("Metadata"))
	xmpStream.Dictionary.Set("Subtype", core.PdfName("XML"))
	xmpNum := w.AddObject(xmpStream)
	w.SetMetadataRef(xmpNum)

	// 2. sRGB ICC profile + OutputIntent.
	iccProfile := sRGBICCProfile()
	iccStream := core.NewStream(iccProfile)
	if err := iccStream.Compress(); err != nil {
		return fmt.Errorf("compressing ICC profile: %w", err)
	}
	iccStream.Dictionary.Set("N", core.PdfInteger(3))
	iccNum := w.AddObject(iccStream)

	outputIntent := core.NewDictionary()
	outputIntent.Set("Type", core.PdfName("OutputIntent"))
	outputIntent.Set("S", core.PdfName("GTS_PDFA1"))
	outputIntent.Set("OutputConditionIdentifier", core.PdfString("sRGB"))
	outputIntent.Set("RegistryName", core.PdfString("http://www.color.org"))
	outputIntent.Set("Info", core.PdfString("sRGB IEC61966-2.1"))
	outputIntent.Set("DestOutputProfile", ref(iccNum))
	oiNum := w.AddObject(outputIntent)
	w.SetOutputIntents([]int{oiNum})

	// 3. Document ID for trailer.
	h := md5.Sum([]byte(fmt.Sprintf("%s-%d", d.metadata.Title, time.Now().UnixNano())))
	w.SetDocumentID(core.PdfHexString(h[:]), core.PdfHexString(h[:]))
	return nil
}

func parsePDFALevel(level PDFALevel) (int, string) {
	switch level {
	case PDFA2b:
		return 2, "B"
	default:
		return 1, "B"
	}
}

// xmlEscape escapes XML special characters.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// buildPDFAXMPMetadata generates XMP metadata with PDF/A identification.
func buildPDFAXMPMetadata(meta Metadata, part int, conformance string) []byte {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	title := meta.Title
	if title == "" {
		title = "Untitled"
	}
	creator := meta.Author
	if creator == "" {
		creator = "Unknown"
	}
	producer := meta.Producer
	if producer == "" {
		producer = "github.com/oarkflow/pdf"
	}

	xmp := fmt.Sprintf(`<?xpacket begin="%s" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description rdf:about=""
        xmlns:dc="http://purl.org/dc/elements/1.1/"
        xmlns:xmp="http://ns.adobe.com/xap/1.0/"
        xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
        xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/">
      <dc:title>
        <rdf:Alt>
          <rdf:li xml:lang="x-default">%s</rdf:li>
        </rdf:Alt>
      </dc:title>
      <dc:creator>
        <rdf:Seq>
          <rdf:li>%s</rdf:li>
        </rdf:Seq>
      </dc:creator>
      <xmp:CreateDate>%s</xmp:CreateDate>
      <xmp:ModifyDate>%s</xmp:ModifyDate>
      <pdf:Producer>%s</pdf:Producer>
      <pdfaid:part>%d</pdfaid:part>
      <pdfaid:conformance>%s</pdfaid:conformance>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`,
		"\xEF\xBB\xBF",
		xmlEscape(title),
		xmlEscape(creator),
		now, now,
		xmlEscape(producer),
		part,
		conformance,
	)

	return []byte(xmp)
}

// sRGBICCProfile returns a minimal valid sRGB ICC profile.
func sRGBICCProfile() []byte {
	fixed := func(v float64) uint32 {
		return uint32(int32(v * 65536.0))
	}

	type xyz struct{ x, y, z float64 }
	wp := xyz{0.9505, 1.0000, 1.0890}
	rXYZ := xyz{0.4124, 0.2126, 0.0193}
	gXYZ := xyz{0.3576, 0.7152, 0.1192}
	bXYZ := xyz{0.1805, 0.0722, 0.9505}

	makeXYZ := func(v xyz) []byte {
		b := make([]byte, 20)
		copy(b[0:4], "XYZ ")
		binary.BigEndian.PutUint32(b[8:12], fixed(v.x))
		binary.BigEndian.PutUint32(b[12:16], fixed(v.y))
		binary.BigEndian.PutUint32(b[16:20], fixed(v.z))
		return b
	}

	makeTRC := func() []byte {
		b := make([]byte, 14)
		copy(b[0:4], "curv")
		binary.BigEndian.PutUint32(b[8:12], 1)
		// gamma 2.2 as u8Fixed8: 0x0233
		b[12] = 0x02
		b[13] = 0x33
		// Pad to 4 bytes.
		b = append(b, 0x00, 0x00)
		return b
	}

	makeDesc := func(s string) []byte {
		b := make([]byte, 12)
		copy(b[0:4], "desc")
		binary.BigEndian.PutUint32(b[8:12], uint32(len(s)+1))
		b = append(b, []byte(s)...)
		b = append(b, 0) // null terminator
		// Unicode count=0, scriptcode count=0.
		b = append(b, make([]byte, 4+4+2+1+67)...)
		for len(b)%4 != 0 {
			b = append(b, 0)
		}
		return b
	}

	descData := makeDesc("sRGB IEC61966-2.1")
	wpData := makeXYZ(wp)
	rXYZData := makeXYZ(rXYZ)
	gXYZData := makeXYZ(gXYZ)
	bXYZData := makeXYZ(bXYZ)
	trcData := makeTRC()
	cprtData := makeDesc("No copyright")

	tagCount := 9
	tagTableSize := 4 + tagCount*12
	dataOffset := 128 + tagTableSize

	type tagInfo struct {
		sig  string
		data []byte
	}
	allTags := []tagInfo{
		{"desc", descData},
		{"wtpt", wpData},
		{"rXYZ", rXYZData},
		{"gXYZ", gXYZData},
		{"bXYZ", bXYZData},
		{"rTRC", trcData},
		{"gTRC", trcData},
		{"bTRC", trcData},
		{"cprt", cprtData},
	}

	type tagEntry struct {
		sig            string
		offset, length int
	}
	var entries []tagEntry
	offset := dataOffset
	trcOffset := 0
	var tagData []byte
	for _, t := range allTags {
		if (t.sig == "gTRC" || t.sig == "bTRC") && trcOffset > 0 {
			entries = append(entries, tagEntry{t.sig, trcOffset, len(t.data)})
			continue
		}
		entries = append(entries, tagEntry{t.sig, offset, len(t.data)})
		if t.sig == "rTRC" {
			trcOffset = offset
		}
		tagData = append(tagData, t.data...)
		offset += len(t.data)
	}

	profileSize := offset

	// Header (128 bytes).
	header := make([]byte, 128)
	binary.BigEndian.PutUint32(header[0:4], uint32(profileSize))
	copy(header[4:8], "none")
	binary.BigEndian.PutUint32(header[8:12], 0x02100000) // v2.1.0
	copy(header[12:16], "mntr")
	copy(header[16:20], "RGB ")
	copy(header[20:24], "XYZ ")
	binary.BigEndian.PutUint16(header[24:26], 2000)
	binary.BigEndian.PutUint16(header[26:28], 1)
	binary.BigEndian.PutUint16(header[28:30], 1)
	copy(header[36:40], "acsp")
	copy(header[40:44], "none")
	binary.BigEndian.PutUint32(header[68:72], fixed(0.9642))
	binary.BigEndian.PutUint32(header[72:76], fixed(1.0))
	binary.BigEndian.PutUint32(header[76:80], fixed(0.8249))

	// Tag table.
	table := make([]byte, 4)
	binary.BigEndian.PutUint32(table[0:4], uint32(tagCount))
	for _, e := range entries {
		rec := make([]byte, 12)
		copy(rec[0:4], e.sig)
		binary.BigEndian.PutUint32(rec[4:8], uint32(e.offset))
		binary.BigEndian.PutUint32(rec[8:12], uint32(e.length))
		table = append(table, rec...)
	}

	// Assemble profile.
	profile := make([]byte, 0, profileSize)
	profile = append(profile, header...)
	profile = append(profile, table...)
	profile = append(profile, tagData...)

	// Profile ID (MD5).
	id := md5.Sum(profile)
	copy(profile[84:100], id[:])

	return profile
}
