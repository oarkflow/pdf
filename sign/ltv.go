package sign

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"strings"
)

// embedLTVData adds a DSS (Document Security Store) dictionary to the PDF
// for long-term validation. This includes certificates, OCSP responses,
// and CRL data needed to validate the signature offline.
func embedLTVData(pdf []byte, certs []*x509.Certificate) ([]byte, error) {
	if len(certs) == 0 {
		return pdf, nil
	}

	// Find the highest object number for new objects.
	maxObj := findMaxObjectNumber(pdf)

	var buf bytes.Buffer
	buf.Write(pdf)

	// Remove trailing %%EOF if present so we can append.
	content := buf.Bytes()
	eofIdx := bytes.LastIndex(content, []byte("%%EOF"))
	if eofIdx >= 0 {
		buf.Reset()
		buf.Write(content[:eofIdx])
	}

	// Create certificate stream objects.
	certObjNums := make([]int, 0, len(certs))
	certOffsets := make([]int, 0, len(certs))
	for _, cert := range certs {
		maxObj++
		objNum := maxObj
		certObjNums = append(certObjNums, objNum)
		certOffsets = append(certOffsets, buf.Len())
		buf.WriteString(fmt.Sprintf("\n%d 0 obj\n<< /Length %d >>\nstream\n", objNum, len(cert.Raw)))
		buf.Write(cert.Raw)
		buf.WriteString("\nendstream\nendobj\n")
	}

	// Create the DSS dictionary object.
	maxObj++
	dssObjNum := maxObj
	dssOffset := buf.Len()

	var certRefs []string
	for _, n := range certObjNums {
		certRefs = append(certRefs, fmt.Sprintf("%d 0 R", n))
	}

	buf.WriteString(fmt.Sprintf("\n%d 0 obj\n<<\n/Type /DSS\n/Certs [%s]\n>>\nendobj\n",
		dssObjNum, strings.Join(certRefs, " ")))

	// Write xref table.
	xrefOffset := buf.Len()
	totalNewObjs := len(certObjNums) + 1 // certs + DSS
	firstNewObj := certObjNums[0]
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("%d %d\n", firstNewObj, totalNewObjs))
	for _, off := range certOffsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", off))
	}
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", dssOffset))

	// Trailer.
	prevXref := findStartXref(pdf)
	trailerSize := maxObj + 1
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Prev %d >>\n", trailerSize, prevXref))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	return buf.Bytes(), nil
}
