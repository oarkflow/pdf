package font

import (
	"testing"

	"golang.org/x/image/math/fixed"
)

func TestLoadTrueType_InvalidData(t *testing.T) {
	_, err := LoadTrueType([]byte("not a font"))
	if err == nil {
		t.Error("expected error for invalid font data")
	}
}

func TestLoadTrueType_EmptyData(t *testing.T) {
	_, err := LoadTrueType(nil)
	if err == nil {
		t.Error("expected error for nil data")
	}
}

func TestLoadTrueType_ShortData(t *testing.T) {
	_, err := LoadTrueType([]byte{0, 1, 0, 0})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestLoadTrueTypeFile_NonExistent(t *testing.T) {
	_, err := LoadTrueTypeFile("/nonexistent/font.ttf")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFix266ToInt(t *testing.T) {
	tests := []struct {
		input fixed.Int26_6
		want  int
	}{
		{0, 0},
		{64, 1},   // 1.0 in 26.6
		{128, 2},  // 2.0
		{96, 2},   // 1.5 rounds to 2
	}
	for _, tt := range tests {
		got := fix266ToInt(tt.input)
		if got != tt.want {
			t.Errorf("fix266ToInt(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
