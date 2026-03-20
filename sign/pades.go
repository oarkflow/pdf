package sign

import (
	"fmt"
)

// applyPAdESProfile orchestrates PDF signing according to the specified PAdES profile.
// It follows the sequence: prepare -> sign -> embed timestamp (B-T) -> embed LTV (B-LT).
func applyPAdESProfile(profile Profile, pdf []byte, signer Signer, opts Options) ([]byte, error) {
	// Step 1: Insert signature placeholder.
	prepared, byteRange, contentsOffset, err := PreparePDF(pdf, opts)
	if err != nil {
		return nil, fmt.Errorf("pades: prepare: %w", err)
	}

	// Step 2: Compute the data to be signed (bytes covered by ByteRange).
	signedData := extractSignedBytes(prepared, byteRange)

	// Step 3: Build CMS signature.
	cmsSig, err := buildCMSSignedData(signer, signedData, opts)
	if err != nil {
		return nil, fmt.Errorf("pades: build cms: %w", err)
	}

	// Step 4: Embed signature into PDF.
	signed, err := AppendSignature(prepared, cmsSig, byteRange, contentsOffset)
	if err != nil {
		return nil, fmt.Errorf("pades: append signature: %w", err)
	}

	// Step 5: For PAdES B-LT, embed LTV data.
	if profile >= PAdESBLT {
		certs := signer.CertificateChain()
		if len(opts.CertStore) > 0 {
			certs = append(certs, opts.CertStore...)
		}
		signed, err = embedLTVData(signed, certs)
		if err != nil {
			return nil, fmt.Errorf("pades: embed ltv: %w", err)
		}
	}

	return signed, nil
}

// extractSignedBytes returns the concatenation of byte ranges that are signed.
func extractSignedBytes(pdf []byte, byteRange [4]int) []byte {
	part1 := pdf[byteRange[0] : byteRange[0]+byteRange[1]]
	part2 := pdf[byteRange[2] : byteRange[2]+byteRange[3]]
	result := make([]byte, len(part1)+len(part2))
	copy(result, part1)
	copy(result[len(part1):], part2)
	return result
}
