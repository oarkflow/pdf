package font

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

// Subset creates a minimal TTF file containing only the specified glyphs.
// Glyph ID 0 (.notdef) is always included.

type tableEntry struct {
	tag  string
	data []byte
}

func Subset(data []byte, glyphIDs []uint16) ([]byte, error) {
	if len(data) < 12 {
		return nil, errors.New("font: data too short for TTF")
	}

	dir, err := parseTableDirectory(data)
	if err != nil {
		return nil, err
	}

	// Always include glyph 0.
	gidSet := map[uint16]bool{0: true}
	for _, g := range glyphIDs {
		gidSet[g] = true
	}

	// Resolve composite glyph components.
	if err := resolveComposites(data, dir, gidSet); err != nil {
		return nil, err
	}

	// Build sorted glyph list; remap old GID -> new GID.
	sorted := make([]uint16, 0, len(gidSet))
	for g := range gidSet {
		sorted = append(sorted, g)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	gidMap := make(map[uint16]uint16) // old -> new
	for i, g := range sorted {
		gidMap[g] = uint16(i)
	}
	numGlyphs := uint16(len(sorted))

	// Read original tables we need.
	headData := getTable(data, dir, "head")
	hheaData := getTable(data, dir, "hhea")
	maxpData := getTable(data, dir, "maxp")
	os2Data := getTable(data, dir, "OS/2")
	nameData := getTable(data, dir, "name")
	postData := getTable(data, dir, "post")
	hmtxData := getTable(data, dir, "hmtx")
	locaData := getTable(data, dir, "loca")
	glyfData := getTable(data, dir, "glyf")

	if headData == nil || locaData == nil || glyfData == nil || hmtxData == nil || maxpData == nil || hheaData == nil {
		return nil, errors.New("font: missing required tables for subsetting")
	}

	// Determine loca format (0=short, 1=long) from head table.
	locaFormat := int16(binary.BigEndian.Uint16(headData[50:52]))

	// Read original number of glyphs.
	origNumGlyphs := int(binary.BigEndian.Uint16(maxpData[4:6]))

	// Read number of long hor metrics from hhea.
	numHMetrics := int(binary.BigEndian.Uint16(hheaData[34:36]))

	// Get glyph offsets from loca.
	getOffset := func(gid int) (uint32, uint32) {
		if locaFormat == 0 {
			// Short format: offsets are uint16 * 2
			if (gid+1)*2+1 >= len(locaData) {
				return 0, 0
			}
			off := uint32(binary.BigEndian.Uint16(locaData[gid*2:])) * 2
			next := uint32(binary.BigEndian.Uint16(locaData[(gid+1)*2:])) * 2
			return off, next
		}
		// Long format: offsets are uint32
		if (gid+1)*4+3 >= len(locaData) {
			return 0, 0
		}
		off := binary.BigEndian.Uint32(locaData[gid*4:])
		next := binary.BigEndian.Uint32(locaData[(gid+1)*4:])
		return off, next
	}

	// Build new glyf table and loca table (using long format).
	var newGlyf []byte
	newLoca := make([]byte, (int(numGlyphs)+1)*4)
	for i, oldGID := range sorted {
		binary.BigEndian.PutUint32(newLoca[i*4:], uint32(len(newGlyf)))
		if int(oldGID) >= origNumGlyphs {
			continue
		}
		off, next := getOffset(int(oldGID))
		if next <= off || int(off) >= len(glyfData) || int(next) > len(glyfData) {
			continue // empty glyph
		}
		glyphBytes := make([]byte, next-off)
		copy(glyphBytes, glyfData[off:next])
		// Remap composite glyph component references.
		remapCompositeRefs(glyphBytes, gidMap)
		newGlyf = append(newGlyf, glyphBytes...)
	}
	binary.BigEndian.PutUint32(newLoca[int(numGlyphs)*4:], uint32(len(newGlyf)))

	// Build new hmtx table.
	newHmtx := buildNewHmtx(hmtxData, sorted, numHMetrics)

	// Update head: set loca format to long (1).
	newHead := make([]byte, len(headData))
	copy(newHead, headData)
	binary.BigEndian.PutUint16(newHead[50:52], 1)

	// Update maxp: set numGlyphs.
	newMaxp := make([]byte, len(maxpData))
	copy(newMaxp, maxpData)
	binary.BigEndian.PutUint16(newMaxp[4:6], numGlyphs)

	// Update hhea: set numberOfHMetrics.
	newHhea := make([]byte, len(hheaData))
	copy(newHhea, hheaData)
	binary.BigEndian.PutUint16(newHhea[34:36], numGlyphs)

	// Build a simple cmap (format 4 for BMP).
	newCmap := buildSubsetCmap(sorted)

	// Build a minimal post table (format 3 = no glyph names).
	newPost := []byte{
		0, 3, 0, 0, // format 3.0
		0, 0, 0, 0, // italicAngle
		0, 0, // underlinePosition
		0, 0, // underlineThickness
		0, 0, 0, 0, // isFixedPitch
		0, 0, 0, 0, // minMemType42
		0, 0, 0, 0, // maxMemType42
		0, 0, 0, 0, // minMemType1
		0, 0, 0, 0, // maxMemType1
	}
	if postData != nil && len(postData) >= 32 {
		copy(newPost[4:], postData[4:32]) // preserve italic angle etc.
	}

	// Assemble output tables.
	tables := []tableEntry{
		{"cmap", newCmap},
		{"glyf", newGlyf},
		{"head", newHead},
		{"hhea", newHhea},
		{"hmtx", newHmtx},
		{"loca", newLoca},
		{"maxp", newMaxp},
		{"post", newPost},
	}
	if os2Data != nil {
		tables = append(tables, tableEntry{"OS/2", os2Data})
	}
	if nameData != nil {
		tables = append(tables, tableEntry{"name", nameData})
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].tag < tables[j].tag })

	return assembleTTF(tables)
}

type tableRecord struct {
	tag      string
	checksum uint32
	offset   uint32
	length   uint32
}

func parseTableDirectory(data []byte) ([]tableRecord, error) {
	if len(data) < 12 {
		return nil, errors.New("font: truncated table directory")
	}
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	if len(data) < 12+numTables*16 {
		return nil, errors.New("font: truncated table records")
	}
	records := make([]tableRecord, numTables)
	for i := 0; i < numTables; i++ {
		off := 12 + i*16
		records[i] = tableRecord{
			tag:      string(data[off : off+4]),
			checksum: binary.BigEndian.Uint32(data[off+4:]),
			offset:   binary.BigEndian.Uint32(data[off+8:]),
			length:   binary.BigEndian.Uint32(data[off+12:]),
		}
	}
	return records, nil
}

func getTable(data []byte, dir []tableRecord, tag string) []byte {
	for _, r := range dir {
		if r.tag == tag {
			end := r.offset + r.length
			if int(end) > len(data) {
				return nil
			}
			return data[r.offset:end]
		}
	}
	return nil
}

func resolveComposites(data []byte, dir []tableRecord, gidSet map[uint16]bool) error {
	glyfData := getTable(data, dir, "glyf")
	locaData := getTable(data, dir, "loca")
	headData := getTable(data, dir, "head")
	maxpData := getTable(data, dir, "maxp")
	if glyfData == nil || locaData == nil || headData == nil || maxpData == nil {
		return nil
	}

	locaFormat := int16(binary.BigEndian.Uint16(headData[50:52]))
	origNumGlyphs := int(binary.BigEndian.Uint16(maxpData[4:6]))

	getOff := func(gid int) (uint32, uint32) {
		if locaFormat == 0 {
			if (gid+1)*2+1 >= len(locaData) {
				return 0, 0
			}
			return uint32(binary.BigEndian.Uint16(locaData[gid*2:])) * 2,
				uint32(binary.BigEndian.Uint16(locaData[(gid+1)*2:])) * 2
		}
		if (gid+1)*4+3 >= len(locaData) {
			return 0, 0
		}
		return binary.BigEndian.Uint32(locaData[gid*4:]),
			binary.BigEndian.Uint32(locaData[(gid+1)*4:])
	}

	// Iterate until no new glyphs added.
	for {
		added := false
		for g := range gidSet {
			if int(g) >= origNumGlyphs {
				continue
			}
			off, next := getOff(int(g))
			if next <= off || int(next) > len(glyfData) {
				continue
			}
			glyph := glyfData[off:next]
			if len(glyph) < 10 {
				continue
			}
			numContours := int16(binary.BigEndian.Uint16(glyph[0:2]))
			if numContours >= 0 {
				continue // simple glyph
			}
			// Composite glyph: parse component references.
			i := 10
			for {
				if i+4 > len(glyph) {
					break
				}
				flags := binary.BigEndian.Uint16(glyph[i:])
				compGID := binary.BigEndian.Uint16(glyph[i+2:])
				if !gidSet[compGID] {
					gidSet[compGID] = true
					added = true
				}
				i += 4
				// Skip arguments.
				if flags&1 != 0 { // ARG_1_AND_2_ARE_WORDS
					i += 4
				} else {
					i += 2
				}
				// Skip transform.
				if flags&8 != 0 { // WE_HAVE_A_SCALE
					i += 2
				} else if flags&64 != 0 { // WE_HAVE_AN_X_AND_Y_SCALE
					i += 4
				} else if flags&128 != 0 { // WE_HAVE_A_TWO_BY_TWO
					i += 8
				}
				if flags&0x20 == 0 { // MORE_COMPONENTS
					break
				}
			}
		}
		if !added {
			break
		}
	}
	return nil
}

func remapCompositeRefs(glyph []byte, gidMap map[uint16]uint16) {
	if len(glyph) < 10 {
		return
	}
	numContours := int16(binary.BigEndian.Uint16(glyph[0:2]))
	if numContours >= 0 {
		return
	}
	i := 10
	for {
		if i+4 > len(glyph) {
			break
		}
		flags := binary.BigEndian.Uint16(glyph[i:])
		oldGID := binary.BigEndian.Uint16(glyph[i+2:])
		if newGID, ok := gidMap[oldGID]; ok {
			binary.BigEndian.PutUint16(glyph[i+2:], newGID)
		}
		i += 4
		if flags&1 != 0 {
			i += 4
		} else {
			i += 2
		}
		if flags&8 != 0 {
			i += 2
		} else if flags&64 != 0 {
			i += 4
		} else if flags&128 != 0 {
			i += 8
		}
		if flags&0x20 == 0 {
			break
		}
	}
}

func buildNewHmtx(hmtxData []byte, sorted []uint16, numHMetrics int) []byte {
	// Each long metric: 2 bytes advanceWidth + 2 bytes lsb.
	// After that, only 2-byte lsb entries.
	getMetric := func(gid uint16) (uint16, int16) {
		idx := int(gid)
		if idx < numHMetrics {
			off := idx * 4
			if off+3 >= len(hmtxData) {
				return 0, 0
			}
			aw := binary.BigEndian.Uint16(hmtxData[off:])
			lsb := int16(binary.BigEndian.Uint16(hmtxData[off+2:]))
			return aw, lsb
		}
		// Use last advance width.
		var aw uint16
		if numHMetrics > 0 {
			off := (numHMetrics - 1) * 4
			if off+1 < len(hmtxData) {
				aw = binary.BigEndian.Uint16(hmtxData[off:])
			}
		}
		lsbOff := numHMetrics*4 + (idx-numHMetrics)*2
		var lsb int16
		if lsbOff+1 < len(hmtxData) {
			lsb = int16(binary.BigEndian.Uint16(hmtxData[lsbOff:]))
		}
		return aw, lsb
	}

	buf := make([]byte, len(sorted)*4)
	for i, gid := range sorted {
		aw, lsb := getMetric(gid)
		binary.BigEndian.PutUint16(buf[i*4:], aw)
		binary.BigEndian.PutUint16(buf[i*4+2:], uint16(lsb))
	}
	return buf
}

func buildSubsetCmap(sorted []uint16) []byte {
	// Build a cmap table with format 4 subtable.
	// For simplicity, map each new GID to itself as a character code.
	// This is an identity mapping.

	segCount := len(sorted) + 1 // +1 for terminal segment
	searchRange := 1
	for searchRange*2 <= segCount {
		searchRange *= 2
	}
	searchRange *= 2
	entrySelector := 0
	for s := searchRange / 2; s > 1; s >>= 1 {
		entrySelector++
	}
	rangeShift := segCount*2 - searchRange

	// Format 4 subtable.
	length := 14 + segCount*8 // header + endCode + pad + startCode + idDelta + idRangeOffset
	var sub []byte
	sub = appendU16(sub, 4)                  // format
	sub = appendU16(sub, uint16(length))     // length
	sub = appendU16(sub, 0)                  // language
	sub = appendU16(sub, uint16(segCount*2)) // segCountX2
	sub = appendU16(sub, uint16(searchRange))
	sub = appendU16(sub, uint16(entrySelector))
	sub = appendU16(sub, uint16(rangeShift))

	// endCode
	for _, gid := range sorted {
		sub = appendU16(sub, gid)
	}
	sub = appendU16(sub, 0xFFFF) // terminal
	sub = appendU16(sub, 0)      // reservedPad

	// startCode
	for _, gid := range sorted {
		sub = appendU16(sub, gid)
	}
	sub = appendU16(sub, 0xFFFF)

	// idDelta (identity: delta = 0 for each)
	for range sorted {
		sub = appendU16(sub, 0)
	}
	sub = appendU16(sub, 1) // terminal delta

	// idRangeOffset (all 0)
	for i := 0; i <= len(sorted); i++ {
		sub = appendU16(sub, 0)
	}

	// cmap header: version=0, numTables=1, platformID=3(Windows), encodingID=1(BMP), offset=12
	var cmap []byte
	cmap = appendU16(cmap, 0)  // version
	cmap = appendU16(cmap, 1)  // numTables
	cmap = appendU16(cmap, 3)  // platformID
	cmap = appendU16(cmap, 1)  // encodingID
	cmap = appendU32(cmap, 12) // offset to subtable
	cmap = append(cmap, sub...)

	return cmap
}

func assembleTTF(tables []tableEntry) ([]byte, error) {
	numTables := len(tables)
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

	headerSize := 12 + numTables*16
	// Calculate offsets.
	offset := uint32(headerSize)
	offsets := make([]uint32, numTables)
	for i, t := range tables {
		offsets[i] = offset
		offset += uint32(len(t.data))
		// Pad to 4-byte boundary.
		if offset%4 != 0 {
			offset += 4 - offset%4
		}
	}

	var out []byte
	// sfVersion (TrueType)
	out = appendU32(out, 0x00010000)
	out = appendU16(out, uint16(numTables))
	out = appendU16(out, uint16(searchRange))
	out = appendU16(out, uint16(entrySelector))
	out = appendU16(out, uint16(rangeShift))

	// Table records.
	for i, t := range tables {
		if len(t.tag) != 4 {
			return nil, fmt.Errorf("font: invalid table tag %q", t.tag)
		}
		out = append(out, t.tag[0], t.tag[1], t.tag[2], t.tag[3])
		out = appendU32(out, calcChecksum(t.data))
		out = appendU32(out, offsets[i])
		out = appendU32(out, uint32(len(t.data)))
	}

	// Table data.
	for _, t := range tables {
		out = append(out, t.data...)
		// Pad to 4-byte boundary.
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}

	return out, nil
}

func calcChecksum(data []byte) uint32 {
	var sum uint32
	// Pad to 4 bytes.
	padded := make([]byte, len(data))
	copy(padded, data)
	for len(padded)%4 != 0 {
		padded = append(padded, 0)
	}
	for i := 0; i < len(padded); i += 4 {
		sum += binary.BigEndian.Uint32(padded[i:])
	}
	return sum
}

func appendU16(b []byte, v uint16) []byte {
	return append(b, byte(v>>8), byte(v))
}

func appendU32(b []byte, v uint32) []byte {
	return append(b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}
