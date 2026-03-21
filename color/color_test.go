package color

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestFromHex6(t *testing.T) {
	c, err := FromHex("#FF0000")
	if err != nil {
		t.Fatal(err)
	}
	if !approx(c.R, 1) || !approx(c.G, 0) || !approx(c.B, 0) {
		t.Errorf("got %+v", c)
	}
}

func TestFromHex3(t *testing.T) {
	c, err := FromHex("#F00")
	if err != nil {
		t.Fatal(err)
	}
	if !approx(c.R, 1) || !approx(c.G, 0) || !approx(c.B, 0) {
		t.Errorf("got %+v", c)
	}
}

func TestFromHexNoHash(t *testing.T) {
	c, err := FromHex("00FF00")
	if err != nil {
		t.Fatal(err)
	}
	if !approx(c.G, 1) {
		t.Errorf("G = %f", c.G)
	}
}

func TestFromHexInvalid(t *testing.T) {
	_, err := FromHex("ZZZZZZ")
	if err == nil {
		t.Error("expected error")
	}
	_, err = FromHex("#12")
	if err == nil {
		t.Error("expected error for 2-char hex")
	}
}

func TestFromRGBi(t *testing.T) {
	c := FromRGBi(255, 128, 0)
	if !approx(c.R, 1) || !approx(c.B, 0) {
		t.Errorf("got %+v", c)
	}
}

func TestRGBToCMYK(t *testing.T) {
	c := RGB{1, 0, 0}.ToCMYK()
	if !approx(c.C, 0) || !approx(c.M, 1) || !approx(c.Y, 1) || !approx(c.K, 0) {
		t.Errorf("red CMYK = %+v", c)
	}
}

func TestRGBToGray(t *testing.T) {
	g := RGB{1, 1, 1}.ToGray()
	if !approx(g.G, 1) {
		t.Errorf("white gray = %f", g.G)
	}
}

func TestBlackToCMYK(t *testing.T) {
	c := RGB{0, 0, 0}.ToCMYK()
	if !approx(c.K, 1) {
		t.Errorf("black CMYK = %+v", c)
	}
}

func TestCMYKToRGB(t *testing.T) {
	c := CMYK{0, 0, 0, 0}.ToRGB()
	if !approx(c.R, 1) || !approx(c.G, 1) || !approx(c.B, 1) {
		t.Errorf("got %+v", c)
	}
}

func TestGrayToRGB(t *testing.T) {
	c := Gray{0.5}.ToRGB()
	if !approx(c.R, 0.5) || !approx(c.G, 0.5) || !approx(c.B, 0.5) {
		t.Errorf("got %+v", c)
	}
}

func TestHSLToRGB(t *testing.T) {
	// Pure red: H=0, S=1, L=0.5
	c := HSL{0, 1, 0.5}.ToRGB()
	if !approx(c.R, 1) || !approx(c.G, 0) || !approx(c.B, 0) {
		t.Errorf("red HSL -> %+v", c)
	}
}

func TestHSLGray(t *testing.T) {
	// S=0 means gray
	c := HSL{0, 0, 0.5}.ToRGB()
	if !approx(c.R, 0.5) || !approx(c.G, 0.5) || !approx(c.B, 0.5) {
		t.Errorf("gray HSL -> %+v", c)
	}
}

func TestNamedColors(t *testing.T) {
	if !approx(Black.R, 0) || !approx(Black.G, 0) || !approx(Black.B, 0) {
		t.Error("Black is wrong")
	}
	if !approx(Blue.B, 1) {
		t.Error("Blue is wrong")
	}
}

func TestColorInterface(t *testing.T) {
	// All types implement Color
	var _ Color = RGB{}
	var _ Color = CMYK{}
	var _ Color = Gray{}
	var _ Color = HSL{}
}
