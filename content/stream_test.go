package content

import (
	"strings"
	"testing"
)

func TestNewStream(t *testing.T) {
	s := New()
	if s.String() != "" {
		t.Error("new stream should be empty")
	}
}

func TestSaveRestoreState(t *testing.T) {
	s := New()
	s.SaveState()
	s.RestoreState()
	if s.String() != "q\nQ\n" {
		t.Errorf("got %q", s.String())
	}
}

func TestBeginEndText(t *testing.T) {
	s := New()
	s.BeginText()
	s.EndText()
	if s.String() != "BT\nET\n" {
		t.Errorf("got %q", s.String())
	}
}

func TestSetFont(t *testing.T) {
	s := New()
	s.SetFont("F1", 12)
	if !strings.Contains(s.String(), "/F1 12 Tf") {
		t.Errorf("got %q", s.String())
	}
}

func TestShowText(t *testing.T) {
	s := New()
	s.ShowText("Hello")
	if !strings.Contains(s.String(), "(Hello) Tj") {
		t.Errorf("got %q", s.String())
	}
}

func TestShowTextEscape(t *testing.T) {
	s := New()
	s.ShowText("a(b)c")
	out := s.String()
	if !strings.Contains(out, `\(`) || !strings.Contains(out, `\)`) {
		t.Errorf("escaping failed: %q", out)
	}
}

func TestMoveTo(t *testing.T) {
	s := New()
	s.MoveTo(10, 20)
	if !strings.Contains(s.String(), "10 20 m") {
		t.Errorf("got %q", s.String())
	}
}

func TestLineTo(t *testing.T) {
	s := New()
	s.LineTo(30, 40)
	if !strings.Contains(s.String(), "30 40 l") {
		t.Errorf("got %q", s.String())
	}
}

func TestRectangle(t *testing.T) {
	s := New()
	s.Rectangle(0, 0, 100, 50)
	if !strings.Contains(s.String(), "re") {
		t.Errorf("got %q", s.String())
	}
}

func TestSetFillColorRGB(t *testing.T) {
	s := New()
	s.SetFillColorRGB(1, 0, 0)
	if !strings.Contains(s.String(), "1 0 0 rg") {
		t.Errorf("got %q", s.String())
	}
}

func TestStroke(t *testing.T) {
	s := New()
	s.Stroke()
	if !strings.Contains(s.String(), "S\n") {
		t.Errorf("got %q", s.String())
	}
}

func TestFill(t *testing.T) {
	s := New()
	s.Fill()
	if s.String() != "f\n" {
		t.Errorf("got %q", s.String())
	}
}

func TestSetLineWidth(t *testing.T) {
	s := New()
	s.SetLineWidth(2.5)
	if !strings.Contains(s.String(), "2.5 w") {
		t.Errorf("got %q", s.String())
	}
}

func TestDrawXObject(t *testing.T) {
	s := New()
	s.DrawXObject("Im1")
	if !strings.Contains(s.String(), "/Im1 Do") {
		t.Errorf("got %q", s.String())
	}
}

func TestBytes(t *testing.T) {
	s := New()
	s.Fill()
	b := s.Bytes()
	if len(b) == 0 {
		t.Error("Bytes is empty")
	}
}
