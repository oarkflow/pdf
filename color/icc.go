package color

import (
	"bytes"
	"encoding/binary"
	"math"
)

// SRGBProfile returns a minimal sRGB ICC profile suitable for PDF/A OutputIntent.
func SRGBProfile() []byte {
	var buf bytes.Buffer

	// We'll build tags first, then assemble the full profile.
	type tag struct {
		sig  [4]byte
		data []byte
	}

	// Helper: encode XYZ data (tag type 'XYZ ', 20 bytes)
	xyzTag := func(x, y, z float64) []byte {
		var b bytes.Buffer
		b.Write([]byte("XYZ "))              // type signature
		binary.Write(&b, binary.BigEndian, uint32(0)) // reserved
		binary.Write(&b, binary.BigEndian, s15Fixed16(x))
		binary.Write(&b, binary.BigEndian, s15Fixed16(y))
		binary.Write(&b, binary.BigEndian, s15Fixed16(z))
		return b.Bytes()
	}

	// Parametric curve type for gamma 2.2
	// Type 0: Y = X^gamma
	curveTag := func() []byte {
		var b bytes.Buffer
		b.Write([]byte("para"))              // type signature
		binary.Write(&b, binary.BigEndian, uint32(0)) // reserved
		binary.Write(&b, binary.BigEndian, uint16(0)) // function type 0
		binary.Write(&b, binary.BigEndian, uint16(0)) // padding
		binary.Write(&b, binary.BigEndian, s15Fixed16(2.2))
		return b.Bytes()
	}

	// Text description tag (desc)
	descTag := func(text string) []byte {
		var b bytes.Buffer
		b.Write([]byte("desc"))              // type signature
		binary.Write(&b, binary.BigEndian, uint32(0)) // reserved
		// ASCII description
		binary.Write(&b, binary.BigEndian, uint32(len(text)+1))
		b.WriteString(text)
		b.WriteByte(0)
		// Unicode localized (empty)
		binary.Write(&b, binary.BigEndian, uint32(0)) // unicode language code
		binary.Write(&b, binary.BigEndian, uint32(0)) // unicode count
		// ScriptCode (empty)
		binary.Write(&b, binary.BigEndian, uint16(0)) // scriptcode code
		b.WriteByte(0)                                  // scriptcode count
		// 67 bytes of filler for scriptcode string
		b.Write(make([]byte, 67))
		return b.Bytes()
	}

	// Copyright tag (text type)
	textTag := func(text string) []byte {
		var b bytes.Buffer
		b.Write([]byte("text"))
		binary.Write(&b, binary.BigEndian, uint32(0))
		b.WriteString(text)
		b.WriteByte(0)
		return b.Bytes()
	}

	// sRGB primaries and D65 white point
	tags := []tag{
		{sig: [4]byte{'d', 'e', 's', 'c'}, data: descTag("sRGB")},
		{sig: [4]byte{'r', 'X', 'Y', 'Z'}, data: xyzTag(0.4360747, 0.2225045, 0.0139322)},
		{sig: [4]byte{'g', 'X', 'Y', 'Z'}, data: xyzTag(0.3850649, 0.7168786, 0.0971045)},
		{sig: [4]byte{'b', 'X', 'Y', 'Z'}, data: xyzTag(0.1430804, 0.0606169, 0.7141733)},
		{sig: [4]byte{'w', 't', 'p', 't'}, data: xyzTag(0.9504559, 1.0000000, 1.0890058)},
		{sig: [4]byte{'r', 'T', 'R', 'C'}, data: curveTag()},
		{sig: [4]byte{'g', 'T', 'R', 'C'}, data: curveTag()},
		{sig: [4]byte{'b', 'T', 'R', 'C'}, data: curveTag()},
		{sig: [4]byte{'c', 'p', 'r', 't'}, data: textTag("No copyright")},
	}

	// Tag table starts at offset 128
	tagTableOffset := 128
	tagCount := len(tags)
	// Each tag table entry: 4(sig) + 4(offset) + 4(size) = 12 bytes
	dataStart := tagTableOffset + 4 + tagCount*12 // 4 bytes for tag count

	// Calculate offsets, aligning each tag data to 4 bytes
	offsets := make([]int, tagCount)
	sizes := make([]int, tagCount)
	offset := dataStart
	for i, t := range tags {
		offsets[i] = offset
		sizes[i] = len(t.data)
		offset += len(t.data)
		// Pad to 4-byte boundary
		if offset%4 != 0 {
			offset += 4 - offset%4
		}
	}
	profileSize := offset

	// Write 128-byte header
	binary.Write(&buf, binary.BigEndian, uint32(profileSize)) // profile size
	buf.Write([]byte{0, 0, 0, 0})                             // preferred CMM
	binary.Write(&buf, binary.BigEndian, uint32(0x02400000))   // version 2.4.0
	buf.Write([]byte("mntr"))                                  // device class: monitor
	buf.Write([]byte("RGB "))                                  // color space
	buf.Write([]byte("XYZ "))                                  // PCS
	// Date/time: 2024-01-01 00:00:00
	binary.Write(&buf, binary.BigEndian, uint16(2024)) // year
	binary.Write(&buf, binary.BigEndian, uint16(1))    // month
	binary.Write(&buf, binary.BigEndian, uint16(1))    // day
	binary.Write(&buf, binary.BigEndian, uint16(0))    // hour
	binary.Write(&buf, binary.BigEndian, uint16(0))    // minute
	binary.Write(&buf, binary.BigEndian, uint16(0))    // second
	buf.Write([]byte("acsp"))                           // file signature
	buf.Write([]byte{0, 0, 0, 0})                      // primary platform
	binary.Write(&buf, binary.BigEndian, uint32(0))     // profile flags
	buf.Write([]byte{0, 0, 0, 0})                      // device manufacturer
	buf.Write([]byte{0, 0, 0, 0})                      // device model
	binary.Write(&buf, binary.BigEndian, uint64(0))     // device attributes
	binary.Write(&buf, binary.BigEndian, uint32(0))     // rendering intent (perceptual)
	// PCS illuminant (D50): X=0.9642, Y=1.0, Z=0.8249
	binary.Write(&buf, binary.BigEndian, s15Fixed16(0.9642))
	binary.Write(&buf, binary.BigEndian, s15Fixed16(1.0))
	binary.Write(&buf, binary.BigEndian, s15Fixed16(0.8249))
	buf.Write([]byte{0, 0, 0, 0}) // profile creator
	// 16 bytes profile ID (MD5, can be zero)
	buf.Write(make([]byte, 16))
	// Remaining reserved bytes to fill 128
	remaining := 128 - buf.Len()
	buf.Write(make([]byte, remaining))

	// Write tag table
	binary.Write(&buf, binary.BigEndian, uint32(tagCount))
	for i, t := range tags {
		buf.Write(t.sig[:])
		binary.Write(&buf, binary.BigEndian, uint32(offsets[i]))
		binary.Write(&buf, binary.BigEndian, uint32(sizes[i]))
	}

	// Write tag data
	for i, t := range tags {
		buf.Write(t.data)
		// Pad to 4-byte boundary
		pad := offsets[i] + sizes[i]
		if pad%4 != 0 {
			buf.Write(make([]byte, 4-pad%4))
		}
	}

	return buf.Bytes()
}

// s15Fixed16 converts a float64 to ICC s15Fixed16Number (4 bytes).
func s15Fixed16(v float64) int32 {
	return int32(math.Round(v * 65536))
}
