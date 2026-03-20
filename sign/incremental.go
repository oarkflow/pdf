package sign

import (
	"encoding/hex"
	"fmt"
)

// AppendSignature performs an incremental update to embed the signature
// into the pre-allocated /Contents field without modifying signed bytes.
func AppendSignature(original []byte, sig []byte, byteRange [4]int, contentsOffset int) ([]byte, error) {
	// Validate byte range.
	if byteRange[0] != 0 {
		return nil, fmt.Errorf("incremental: byte range must start at 0")
	}
	if byteRange[1]+byteRange[3]+byteRange[2] > len(original) {
		return nil, fmt.Errorf("incremental: byte range exceeds PDF size")
	}

	// Hex-encode the signature.
	hexSig := hex.EncodeToString(sig)

	// The hex signature must fit within the allocated contents size.
	if len(hexSig) > signatureContentsSize {
		return nil, fmt.Errorf("incremental: signature too large (%d > %d)", len(hexSig), signatureContentsSize)
	}

	// Pad with zeros.
	for len(hexSig) < signatureContentsSize {
		hexSig += "0"
	}

	// Build the output by copying the original and replacing the contents placeholder.
	result := make([]byte, len(original))
	copy(result, original)

	// The contents field is at contentsOffset: <hex...>
	// contentsOffset points to '<', so hex data starts at contentsOffset+1.
	copy(result[contentsOffset+1:contentsOffset+1+signatureContentsSize], []byte(hexSig))

	// Update the ByteRange values in the PDF.
	updateByteRange(result, byteRange)

	return result, nil
}

// updateByteRange finds and updates the /ByteRange placeholder in the PDF.
func updateByteRange(pdf []byte, byteRange [4]int) {
	// Search for the ByteRange placeholder pattern.
	marker := []byte("/ByteRange [0 0000000000 0000000000 0000000000]")
	for i := 0; i <= len(pdf)-len(marker); i++ {
		if string(pdf[i:i+len(marker)]) == string(marker) {
			replacement := fmt.Sprintf("/ByteRange [%d %010d %010d %010d]",
				byteRange[0], byteRange[1], byteRange[2], byteRange[3])
			copy(pdf[i:i+len(marker)], []byte(replacement))
			return
		}
	}
}
