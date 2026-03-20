package barcode

// GF(256) finite field arithmetic for Reed-Solomon error correction.
// Uses primitive polynomial 0x11D (x^8 + x^4 + x^3 + x^2 + 1), standard for QR codes.

var gf256Exp [512]byte // antilog table (doubled for convenience)
var gf256Log [256]byte // log table

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gf256Exp[i] = byte(x)
		gf256Log[x] = byte(i)
		x <<= 1
		if x >= 256 {
			x ^= 0x11D
		}
	}
	// Duplicate for easy modular access
	for i := 255; i < 512; i++ {
		gf256Exp[i] = gf256Exp[i-255]
	}
}

// gf256Mul multiplies two elements in GF(256).
func gf256Mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return gf256Exp[int(gf256Log[a])+int(gf256Log[b])]
}

// gf256Div divides a by b in GF(256). b must be non-zero.
func gf256Div(a, b byte) byte {
	if a == 0 {
		return 0
	}
	if b == 0 {
		panic("gf256: division by zero")
	}
	diff := int(gf256Log[a]) - int(gf256Log[b])
	if diff < 0 {
		diff += 255
	}
	return gf256Exp[diff]
}

// gf256PolyMul multiplies two polynomials in GF(256).
// Coefficients are ordered from highest degree to lowest.
func gf256PolyMul(a, b []byte) []byte {
	out := make([]byte, len(a)+len(b)-1)
	for i, ca := range a {
		for j, cb := range b {
			out[i+j] ^= gf256Mul(ca, cb)
		}
	}
	return out
}

// gf256GenPoly returns the generator polynomial for numEC error correction codewords.
// g(x) = (x - alpha^0)(x - alpha^1)...(x - alpha^(numEC-1))
func gf256GenPoly(numEC int) []byte {
	g := []byte{1}
	for i := 0; i < numEC; i++ {
		g = gf256PolyMul(g, []byte{1, gf256Exp[i]})
	}
	return g
}
