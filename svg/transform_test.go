package svg

import (
	"math"
	"testing"
)

func TestIdentity(t *testing.T) {
	m := Identity()
	if m != (Matrix{1, 0, 0, 1, 0, 0}) {
		t.Errorf("Identity = %v", m)
	}
}

func TestTransformPoint(t *testing.T) {
	m := Matrix{1, 0, 0, 1, 10, 20} // translate(10,20)
	x, y := m.TransformPoint(5, 5)
	if x != 15 || y != 25 {
		t.Errorf("got (%f, %f), want (15, 25)", x, y)
	}
}

func TestParseTransformTranslate(t *testing.T) {
	m := parseTransform("translate(10, 20)")
	if m[4] != 10 || m[5] != 20 {
		t.Errorf("translate: %v", m)
	}
}

func TestParseTransformScale(t *testing.T) {
	m := parseTransform("scale(2, 3)")
	if m[0] != 2 || m[3] != 3 {
		t.Errorf("scale: %v", m)
	}
}

func TestParseTransformScaleUniform(t *testing.T) {
	m := parseTransform("scale(5)")
	if m[0] != 5 || m[3] != 5 {
		t.Errorf("scale(5): %v", m)
	}
}

func TestParseTransformRotate(t *testing.T) {
	m := parseTransform("rotate(90)")
	// cos(90) ~ 0, sin(90) ~ 1
	if math.Abs(m[0]) > 1e-9 || math.Abs(m[1]-1) > 1e-9 {
		t.Errorf("rotate(90): %v", m)
	}
}

func TestParseTransformMatrix(t *testing.T) {
	m := parseTransform("matrix(1 0 0 1 50 60)")
	if m[4] != 50 || m[5] != 60 {
		t.Errorf("matrix: %v", m)
	}
}

func TestParseTransformChained(t *testing.T) {
	m := parseTransform("translate(10, 0) scale(2)")
	// translate then scale: the resulting e should be 10
	x, y := m.TransformPoint(0, 0)
	if math.Abs(x-10) > 1e-9 || math.Abs(y) > 1e-9 {
		t.Errorf("chained: origin -> (%f, %f)", x, y)
	}
}

func TestParseTransformEmpty(t *testing.T) {
	m := parseTransform("")
	if m != Identity() {
		t.Errorf("empty transform should be identity, got %v", m)
	}
}

func TestMultiplyIdentity(t *testing.T) {
	m := Matrix{2, 0, 0, 3, 10, 20}
	id := Identity()
	if m.Multiply(id) != m {
		t.Error("M * I != M")
	}
	if id.Multiply(m) != m {
		t.Error("I * M != M")
	}
}

func TestToPDFOperator(t *testing.T) {
	m := Identity()
	s := m.ToPDFOperator()
	if s != "1 0 0 1 0 0 cm" {
		t.Errorf("ToPDFOperator = %q", s)
	}
}
