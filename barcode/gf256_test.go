package barcode

import "testing"

func TestGF256MulIdentity(t *testing.T) {
	for i := 0; i < 256; i++ {
		got := gf256Mul(byte(i), 1)
		if got != byte(i) {
			t.Errorf("gf256Mul(%d, 1) = %d, want %d", i, got, i)
		}
	}
}

func TestGF256MulZero(t *testing.T) {
	for i := 0; i < 256; i++ {
		if got := gf256Mul(byte(i), 0); got != 0 {
			t.Errorf("gf256Mul(%d, 0) = %d, want 0", i, got)
		}
		if got := gf256Mul(0, byte(i)); got != 0 {
			t.Errorf("gf256Mul(0, %d) = %d, want 0", i, got)
		}
	}
}

func TestGF256MulCommutative(t *testing.T) {
	for a := 1; a < 256; a++ {
		for b := 1; b < 256; b++ {
			ab := gf256Mul(byte(a), byte(b))
			ba := gf256Mul(byte(b), byte(a))
			if ab != ba {
				t.Fatalf("gf256Mul(%d,%d)=%d != gf256Mul(%d,%d)=%d", a, b, ab, b, a, ba)
			}
		}
	}
}

func TestGF256DivInverse(t *testing.T) {
	for a := 1; a < 256; a++ {
		got, err := gf256Div(byte(a), byte(a))
		if err != nil {
			t.Fatalf("gf256Div(%d, %d) error: %v", a, a, err)
		}
		if got != 1 {
			t.Errorf("gf256Div(%d, %d) = %d, want 1", a, a, got)
		}
	}
}

func TestGF256DivMulRoundTrip(t *testing.T) {
	for a := 1; a < 256; a += 17 {
		for b := 1; b < 256; b += 13 {
			product := gf256Mul(byte(a), byte(b))
			got, err := gf256Div(product, byte(b))
			if err != nil {
				t.Fatalf("gf256Div error: %v", err)
			}
			if got != byte(a) {
				t.Errorf("gf256Div(gf256Mul(%d,%d), %d) = %d, want %d", a, b, b, got, a)
			}
		}
	}
}

func TestGF256DivByZeroReturnsError(t *testing.T) {
	_, err := gf256Div(1, 0)
	if err == nil {
		t.Error("gf256Div(1, 0) should return error")
	}
}

func TestGF256DivZeroNumerator(t *testing.T) {
	for b := 1; b < 256; b++ {
		got, err := gf256Div(0, byte(b))
		if err != nil {
			t.Fatalf("gf256Div(0, %d) error: %v", b, err)
		}
		if got != 0 {
			t.Errorf("gf256Div(0, %d) = %d, want 0", b, got)
		}
	}
}

func TestGF256GenPoly(t *testing.T) {
	g := gf256GenPoly(1)
	if len(g) != 2 || g[0] != 1 || g[1] != 1 {
		t.Errorf("gf256GenPoly(1) = %v, want [1, 1]", g)
	}
}

func TestGF256PolyMulIdentity(t *testing.T) {
	p := []byte{3, 7, 42}
	got := gf256PolyMul(p, []byte{1})
	if len(got) != len(p) {
		t.Fatalf("length mismatch: %d vs %d", len(got), len(p))
	}
	for i := range p {
		if got[i] != p[i] {
			t.Errorf("index %d: got %d, want %d", i, got[i], p[i])
		}
	}
}

func TestGF256ExpLogConsistency(t *testing.T) {
	for x := 1; x < 256; x++ {
		if int(gf256Exp[gf256Log[x]]) != x {
			t.Errorf("exp[log[%d]] = %d", x, gf256Exp[gf256Log[x]])
		}
	}
}
