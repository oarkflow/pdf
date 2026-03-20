package sign

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// signatureContentsSize is the pre-allocated size for the hex-encoded signature.
	// 8192 bytes of hex = 4096 bytes of binary signature data.
	signatureContentsSize = 8192
)

// PreparePDF inserts a signature placeholder into a PDF.
// Returns the modified PDF and the byte range info.
func PreparePDF(pdf []byte, opts Options) (prepared []byte, byteRange [4]int, contentsOffset int, err error) {
	if len(pdf) < 5 || string(pdf[:5]) != "%PDF-" {
		err = fmt.Errorf("placeholder: not a valid PDF")
		return
	}

	// Find the last xref/startxref to locate EOF.
	eofIdx := bytes.LastIndex(pdf, []byte("%%EOF"))
	if eofIdx < 0 {
		err = fmt.Errorf("placeholder: %%EOF not found")
		return
	}

	// Find last trailer to get the root and info refs.
	// We'll do a simple incremental update approach.

	// Find the highest object number.
	maxObj := findMaxObjectNumber(pdf)
	newSigObjNum := maxObj + 1

	// Build the signature dictionary object.
	sigDict := buildSigDict(newSigObjNum, opts)

	// Build incremental update: new object + updated xref + trailer.
	// For simplicity, append the sig object and a minimal cross-reference.
	var buf bytes.Buffer
	buf.Write(pdf)

	// Record where the new object starts.
	sigObjOffset := buf.Len()

	// Write the signature object with placeholder contents.
	sigObjStr := formatSigObject(newSigObjNum, sigDict)
	buf.WriteString(sigObjStr)

	// Now compute byte range and contents offset in the final output.
	// The /Contents field placeholder starts after "<" and before ">".
	fullBytes := buf.Bytes()
	contentsMarker := []byte("/Contents <")
	contentsIdx := bytes.LastIndex(fullBytes, contentsMarker)
	if contentsIdx < 0 {
		err = fmt.Errorf("placeholder: /Contents marker not found")
		return
	}
	contentsOffset = contentsIdx + len(contentsMarker) - 1 // points to '<'

	// ByteRange: [0, contentsOffset, contentsOffset+sigContentsSize+2, remaining]
	afterContents := contentsOffset + signatureContentsSize + 2 // +2 for < and >

	// Write xref and trailer for the incremental update.
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("%d 1\n", newSigObjNum))
	buf.WriteString(fmt.Sprintf("%010d 00000 n \n", sigObjOffset))
	buf.WriteString("trailer\n")

	// Find existing trailer size.
	trailerSize := maxObj + 2 // rough estimate
	prevXref := findStartXref(pdf)
	buf.WriteString(fmt.Sprintf("<< /Size %d /Prev %d >>\n", trailerSize, prevXref))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	prepared = buf.Bytes()
	totalLen := len(prepared)

	byteRange = [4]int{
		0,
		contentsOffset,
		afterContents,
		totalLen - afterContents,
	}

	return
}

func buildSigDict(objNum int, opts Options) string {
	var b strings.Builder
	b.WriteString("/Type /Sig\n")
	b.WriteString("/Filter /Adobe.PPKLite\n")
	b.WriteString("/SubFilter /ETSI.CAdES.detached\n")

	// Placeholder ByteRange - will be filled later.
	b.WriteString("/ByteRange [0 0000000000 0000000000 0000000000]\n")

	// Placeholder Contents.
	b.WriteString("/Contents <")
	b.WriteString(strings.Repeat("0", signatureContentsSize))
	b.WriteString(">\n")

	// Signing time.
	now := time.Now().UTC()
	b.WriteString(fmt.Sprintf("/M (D:%s)\n", now.Format("20060102150405-07'00'")))

	if opts.Reason != "" {
		b.WriteString(fmt.Sprintf("/Reason (%s)\n", pdfEscapeString(opts.Reason)))
	}
	if opts.Location != "" {
		b.WriteString(fmt.Sprintf("/Location (%s)\n", pdfEscapeString(opts.Location)))
	}
	if opts.ContactInfo != "" {
		b.WriteString(fmt.Sprintf("/ContactInfo (%s)\n", pdfEscapeString(opts.ContactInfo)))
	}
	if opts.Name != "" {
		b.WriteString(fmt.Sprintf("/Name (%s)\n", pdfEscapeString(opts.Name)))
	}

	return b.String()
}

func formatSigObject(objNum int, dict string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n%d 0 obj\n<<\n", objNum))
	b.WriteString(dict)
	b.WriteString(">>\nendobj\n")
	return b.String()
}

func pdfEscapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

func findMaxObjectNumber(pdf []byte) int {
	max := 0
	for i := 0; i < len(pdf)-4; i++ {
		if pdf[i] >= '0' && pdf[i] <= '9' {
			// Look for pattern: "N 0 obj"
			j := i
			for j < len(pdf) && pdf[j] >= '0' && pdf[j] <= '9' {
				j++
			}
			if j < len(pdf)-5 && string(pdf[j:j+6]) == " 0 obj" {
				n, err := strconv.Atoi(string(pdf[i:j]))
				if err == nil && n > max {
					max = n
				}
			}
		}
	}
	return max
}

func findStartXref(pdf []byte) int {
	marker := []byte("startxref")
	idx := bytes.LastIndex(pdf, marker)
	if idx < 0 {
		return 0
	}
	// Read the number after "startxref\n".
	start := idx + len(marker)
	for start < len(pdf) && (pdf[start] == '\n' || pdf[start] == '\r' || pdf[start] == ' ') {
		start++
	}
	end := start
	for end < len(pdf) && pdf[end] >= '0' && pdf[end] <= '9' {
		end++
	}
	if start == end {
		return 0
	}
	n, err := strconv.Atoi(string(pdf[start:end]))
	if err != nil {
		return 0
	}
	return n
}
