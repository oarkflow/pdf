package font

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// DecodeWOFF decompresses WOFF1 data into standard sfnt (TTF/OTF) data.
func DecodeWOFF(data []byte) ([]byte, error) {
	if len(data) < 44 {
		return nil, errors.New("font: WOFF data too short")
	}

	sig := binary.BigEndian.Uint32(data[0:4])
	if sig != 0x774F4646 {
		return nil, fmt.Errorf("font: not a WOFF file (signature %08X)", sig)
	}

	flavor := binary.BigEndian.Uint32(data[4:8])
	// length := binary.BigEndian.Uint32(data[8:12]) // total WOFF size
	numTables := int(binary.BigEndian.Uint16(data[12:14]))
	// reserved := binary.BigEndian.Uint16(data[14:16])
	// totalSfntSize := binary.BigEndian.Uint32(data[16:20])
	// ... other header fields we don't need

	if len(data) < 44+numTables*20 {
		return nil, errors.New("font: WOFF table directory truncated")
	}

	type woffTable struct {
		tag        [4]byte
		offset     uint32
		compLength uint32
		origLength uint32
		origChksum uint32
	}

	tables := make([]woffTable, numTables)
	for i := 0; i < numTables; i++ {
		off := 44 + i*20
		copy(tables[i].tag[:], data[off:off+4])
		tables[i].offset = binary.BigEndian.Uint32(data[off+4:])
		tables[i].compLength = binary.BigEndian.Uint32(data[off+8:])
		tables[i].origLength = binary.BigEndian.Uint32(data[off+12:])
		tables[i].origChksum = binary.BigEndian.Uint32(data[off+16:])
	}

	// Calculate sfnt output size.
	headerSize := 12 + numTables*16
	sfntSize := headerSize
	for _, t := range tables {
		sfntSize += int(t.origLength)
		if sfntSize%4 != 0 {
			sfntSize += 4 - sfntSize%4
		}
	}

	// Build sfnt header.
	searchRange := 1
	for searchRange*2 <= numTables {
		searchRange *= 2
	}
	searchRange *= 16
	entrySelector := 0
	for s := searchRange / 16; s > 1; s >>= 1 {
		entrySelector++
	}
	rangeShift := numTables*16 - searchRange

	out := make([]byte, 0, sfntSize)
	out = appendU32(out, flavor)
	out = appendU16(out, uint16(numTables))
	out = appendU16(out, uint16(searchRange))
	out = appendU16(out, uint16(entrySelector))
	out = appendU16(out, uint16(rangeShift))

	// Calculate table offsets in output.
	tableOffset := uint32(headerSize)
	type outEntry struct {
		tag    [4]byte
		chksum uint32
		offset uint32
		length uint32
	}
	entries := make([]outEntry, numTables)
	for i, t := range tables {
		entries[i] = outEntry{
			tag:    t.tag,
			chksum: t.origChksum,
			offset: tableOffset,
			length: t.origLength,
		}
		tableOffset += t.origLength
		if tableOffset%4 != 0 {
			tableOffset += 4 - tableOffset%4
		}
	}

	// Write table records.
	for _, e := range entries {
		out = append(out, e.tag[0], e.tag[1], e.tag[2], e.tag[3])
		out = appendU32(out, e.chksum)
		out = appendU32(out, e.offset)
		out = appendU32(out, e.length)
	}

	// Write table data.
	for _, t := range tables {
		tableData := data[t.offset : t.offset+t.compLength]
		var origData []byte

		if t.compLength < t.origLength {
			// zlib compressed.
			r, err := zlib.NewReader(bytes.NewReader(tableData))
			if err != nil {
				return nil, fmt.Errorf("font: zlib decompress table %s: %w", string(t.tag[:]), err)
			}
			origData, err = io.ReadAll(r)
			r.Close()
			if err != nil {
				return nil, fmt.Errorf("font: zlib read table %s: %w", string(t.tag[:]), err)
			}
			if uint32(len(origData)) != t.origLength {
				return nil, fmt.Errorf("font: table %s decompressed size mismatch", string(t.tag[:]))
			}
		} else {
			origData = tableData[:t.origLength]
		}

		out = append(out, origData...)
		// Pad to 4-byte boundary.
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}

	return out, nil
}
