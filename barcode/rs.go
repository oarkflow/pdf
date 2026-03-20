package barcode

// RSEncode computes Reed-Solomon error correction codewords over GF(256).
// Returns numEC error correction bytes for the given data.
func RSEncode(data []byte, numEC int) []byte {
	gen := gf256GenPoly(numEC)

	// Polynomial long division
	feedback := make([]byte, len(data)+numEC)
	copy(feedback, data)

	for i := 0; i < len(data); i++ {
		coef := feedback[i]
		if coef != 0 {
			for j := 1; j < len(gen); j++ {
				feedback[i+j] ^= gf256Mul(gen[j], coef)
			}
		}
	}

	return feedback[len(data):]
}
