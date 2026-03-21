package barcode

import "testing"

func TestRSEncodeLength(t *testing.T) {
	data := []byte{32, 91, 11, 120, 209, 114, 220, 77}
	ec := RSEncode(data, 10)
	if len(ec) != 10 {
		t.Errorf("got %d bytes, want 10", len(ec))
	}
}

func TestRSEncodeNonZero(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	ec := RSEncode(data, 4)
	allZero := true
	for _, b := range ec {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("all zeros")
	}
}

func TestRSEncodeDeterministic(t *testing.T) {
	data := []byte{72, 101, 108, 108, 111}
	ec1 := RSEncode(data, 6)
	ec2 := RSEncode(data, 6)
	for i := range ec1 {
		if ec1[i] != ec2[i] {
			t.Fatal("not deterministic")
		}
	}
}

func TestRSEncodeDifferentData(t *testing.T) {
	ec1 := RSEncode([]byte{1, 2, 3}, 4)
	ec2 := RSEncode([]byte{3, 2, 1}, 4)
	same := true
	for i := range ec1 {
		if ec1[i] != ec2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different data produced same EC")
	}
}

func TestRSEncodeSingleByte(t *testing.T) {
	ec := RSEncode([]byte{42}, 2)
	if len(ec) != 2 {
		t.Errorf("got %d EC bytes", len(ec))
	}
}
