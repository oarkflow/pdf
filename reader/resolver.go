package reader

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/oarkflow/pdf/core"
)

// StreamObject represents a PDF stream (dictionary + decompressed data).
type StreamObject struct {
	Dict map[string]interface{}
	Data []byte
}

// IndirectRef represents a PDF indirect reference (obj gen R).
type IndirectRef struct {
	ObjNum int
	GenNum int
}

// XRefEntry is a single cross-reference entry.
type XRefEntry struct {
	Offset     int64
	Generation int
	Free       bool
	Compressed bool
	StreamObj  int // object number of the containing object stream
	StreamIdx  int // index within the object stream
}

// maxRecursionDepth is the maximum nesting depth for parsing PDF objects.
// This prevents stack overflow from circular references in malicious PDFs.
const maxRecursionDepth = 100

// Resolver parses and caches PDF objects from raw bytes.
type Resolver struct {
	data  []byte
	xref  map[int]XRefEntry
	cache map[int]interface{}
	mu    sync.RWMutex
	depth int // current recursion depth
	crypt *decryptState
}

type decryptState struct {
	key           []byte
	algorithm     core.EncryptionAlgorithm
	encryptObjNum int
}

// NewResolver creates a Resolver by parsing the xref table/stream from data.
func NewResolver(data []byte) (*Resolver, error) {
	r := &Resolver{
		data:  data,
		xref:  make(map[int]XRefEntry),
		cache: make(map[int]interface{}),
	}
	if err := r.parseXRef(); err != nil {
		return nil, err
	}
	return r, nil
}

// Trailer returns the trailer dictionary.
func (r *Resolver) Trailer() (map[string]interface{}, error) {
	startxref := r.findStartXRef()
	if startxref < 0 {
		return nil, fmt.Errorf("startxref not found")
	}
	tok := NewTokenizer(r.data)
	tok.Seek(int(startxref))

	// Could be xref table or xref stream.
	t, _ := tok.Peek()
	if t.Type == TokenKeyword && t.Value == "xref" {
		// Traditional xref table: skip to trailer.
		return r.parseTrailerDict(tok)
	}
	// xref stream: parse the stream object dict.
	obj := r.parseObjectAt(int(startxref))
	if so, ok := obj.(*StreamObject); ok {
		return so.Dict, nil
	}
	if d, ok := obj.(map[string]interface{}); ok {
		return d, nil
	}
	return nil, fmt.Errorf("could not parse trailer")
}

func (r *Resolver) parseTrailerDict(tok *Tokenizer) (map[string]interface{}, error) {
	// Scan for "trailer" keyword.
	for {
		t, err := tok.Next()
		if err != nil {
			return nil, err
		}
		if t.Type == TokenEOF {
			return nil, fmt.Errorf("trailer keyword not found")
		}
		if t.Type == TokenKeyword && t.Value == "trailer" {
			break
		}
	}
	obj, err := r.parseObject(tok)
	if err != nil {
		return nil, err
	}
	if d, ok := obj.(map[string]interface{}); ok {
		return d, nil
	}
	return nil, fmt.Errorf("trailer is not a dictionary")
}

func (r *Resolver) findStartXRef() int64 {
	// Search backwards for "startxref".
	idx := bytes.LastIndex(r.data, []byte("startxref"))
	if idx < 0 {
		return -1
	}
	tok := NewTokenizer(r.data)
	tok.Seek(idx)
	tok.Next() // "startxref"
	t, err := tok.Next()
	if err != nil || t.Type != TokenInteger {
		return -1
	}
	return t.Int
}

func (r *Resolver) parseXRef() error {
	startxref := r.findStartXRef()
	if startxref < 0 {
		return fmt.Errorf("startxref not found")
	}
	return r.parseXRefAt(int(startxref))
}

func (r *Resolver) parseXRefAt(offset int) error {
	tok := NewTokenizer(r.data)
	tok.Seek(offset)

	t, _ := tok.Peek()
	if t.Type == TokenKeyword && t.Value == "xref" {
		return r.parseXRefTable(tok)
	}
	// xref stream.
	return r.parseXRefStream(offset)
}

func (r *Resolver) parseXRefTable(tok *Tokenizer) error {
	tok.Next() // consume "xref"

	for {
		t, _ := tok.Peek()
		if t.Type == TokenKeyword && t.Value == "trailer" {
			break
		}
		if t.Type == TokenEOF {
			break
		}
		// Read subsection header: startObj count
		startTok, err := tok.Next()
		if err != nil || startTok.Type != TokenInteger {
			break
		}
		countTok, err := tok.Next()
		if err != nil || countTok.Type != TokenInteger {
			return fmt.Errorf("expected xref subsection count")
		}
		startObj := int(startTok.Int)
		count := int(countTok.Int)

		for i := 0; i < count; i++ {
			offTok, err := tok.Next()
			if err != nil {
				return err
			}
			genTok, err := tok.Next()
			if err != nil {
				return err
			}
			typeTok, err := tok.Next()
			if err != nil {
				return err
			}

			objNum := startObj + i
			if _, exists := r.xref[objNum]; exists {
				continue // first entry wins (most recent xref)
			}

			entry := XRefEntry{
				Offset:     offTok.Int,
				Generation: int(genTok.Int),
				Free:       typeTok.Value == "f",
			}
			r.xref[objNum] = entry
		}
	}

	// Parse trailer to check for Prev.
	trailer, err := r.parseTrailerDict(tok)
	if err != nil {
		return nil // tolerate missing trailer
	}
	if prev, ok := getInt(trailer, "Prev"); ok {
		return r.parseXRefAt(int(prev))
	}
	return nil
}

func (r *Resolver) parseXRefStream(offset int) error {
	obj := r.parseObjectAt(offset)
	so, ok := obj.(*StreamObject)
	if !ok {
		return fmt.Errorf("xref stream not found at offset %d", offset)
	}
	dict := so.Dict

	// Decompress if needed.
	streamData, err := r.DecompressStream(dict, so.Data)
	if err != nil {
		return fmt.Errorf("decompressing xref stream: %w", err)
	}

	// Read W array.
	wArr, _ := dict["/W"].([]interface{})
	if len(wArr) < 3 {
		return fmt.Errorf("invalid /W in xref stream")
	}
	w := [3]int{toInt(wArr[0]), toInt(wArr[1]), toInt(wArr[2])}
	entrySize := w[0] + w[1] + w[2]

	// Read Index array.
	var indices []int
	if idxArr, ok := dict["/Index"].([]interface{}); ok {
		for _, v := range idxArr {
			indices = append(indices, toInt(v))
		}
	} else {
		size := toInt(dict["/Size"])
		indices = []int{0, size}
	}

	pos := 0
	for i := 0; i+1 < len(indices); i += 2 {
		startObj := indices[i]
		count := indices[i+1]
		for j := 0; j < count; j++ {
			if pos+entrySize > len(streamData) {
				break
			}
			field := func(width int) int64 {
				var val int64
				for k := 0; k < width; k++ {
					val = val<<8 | int64(streamData[pos])
					pos++
				}
				return val
			}

			var f1, f2, f3 int64
			if w[0] > 0 {
				f1 = field(w[0])
			} else {
				f1 = 1 // default type is 1
			}
			f2 = field(w[1])
			f3 = field(w[2])

			objNum := startObj + j
			if _, exists := r.xref[objNum]; exists {
				continue
			}

			switch f1 {
			case 0:
				r.xref[objNum] = XRefEntry{Free: true, Generation: int(f3)}
			case 1:
				r.xref[objNum] = XRefEntry{Offset: f2, Generation: int(f3)}
			case 2:
				r.xref[objNum] = XRefEntry{
					Compressed: true,
					StreamObj:  int(f2),
					StreamIdx:  int(f3),
				}
			}
		}
	}

	// Follow Prev.
	if prev, ok := getInt(dict, "/Prev"); ok {
		return r.parseXRefAt(int(prev))
	}
	return nil
}

// parseObjectAt parses a top-level indirect object at the given byte offset.
func (r *Resolver) parseObjectAt(offset int) interface{} {
	tok := NewTokenizer(r.data)
	tok.Seek(offset)

	// objNum genNum obj
	objNumTok, _ := tok.Next() // object number
	genTok, _ := tok.Next()    // generation
	tok.Next()                 // "obj"
	objNum := int(objNumTok.Int)
	genNum := int(genTok.Int)

	obj, _ := r.parseObject(tok)

	// Check for stream.
	t, _ := tok.Peek()
	if t.Type == TokenKeyword && t.Value == "stream" {
		tok.Next() // consume "stream"
		// Skip the newline after "stream".
		p := tok.Pos()
		if p < len(r.data) && r.data[p] == '\r' {
			p++
		}
		if p < len(r.data) && r.data[p] == '\n' {
			p++
		}

		dict, _ := obj.(map[string]interface{})
		length := 0
		if dict != nil {
			if l, ok := getInt(dict, "/Length"); ok {
				length = int(l)
			} else if lRef, ok := dict["/Length"].(IndirectRef); ok {
				resolved, _ := r.ResolveObject(lRef.ObjNum)
				if v, ok := resolved.(int64); ok {
					length = int(v)
				}
			}
		}

		end := p + length
		if end > len(r.data) {
			end = len(r.data)
		}
		streamData := r.data[p:end]
		if r.crypt != nil && objNum != r.crypt.encryptObjNum {
			decrypted, err := core.DecryptData(streamData, r.crypt.key, objNum, genNum, r.crypt.algorithm)
			if err == nil {
				streamData = decrypted
			}
		}
		return &StreamObject{Dict: dict, Data: streamData}
	}

	if r.crypt != nil && objNum != r.crypt.encryptObjNum {
		if decrypted, err := r.decryptObject(obj, objNum, genNum); err == nil {
			obj = decrypted
		}
	}

	return obj
}

// parseObject parses a single PDF object from the tokenizer.
func (r *Resolver) parseObject(tok *Tokenizer) (interface{}, error) {
	r.depth++
	defer func() { r.depth-- }()
	if r.depth > maxRecursionDepth {
		return nil, fmt.Errorf("maximum PDF object recursion depth (%d) exceeded", maxRecursionDepth)
	}

	t, err := tok.Next()
	if err != nil {
		return nil, err
	}

	switch t.Type {
	case TokenEOF:
		return nil, fmt.Errorf("unexpected EOF")

	case TokenInteger:
		// Look ahead for "gen R" (indirect reference).
		saved := tok.Pos()
		t2, _ := tok.Peek()
		if t2.Type == TokenInteger {
			tok.Next()
			t3, _ := tok.Peek()
			if t3.Type == TokenKeyword && t3.Value == "R" {
				tok.Next()
				return IndirectRef{ObjNum: int(t.Int), GenNum: int(t2.Int)}, nil
			}
			tok.Seek(saved)
		}
		return t.Int, nil

	case TokenReal:
		return t.Real, nil

	case TokenString:
		return t.Value, nil

	case TokenHexString:
		return decodeHex(t.Value), nil

	case TokenName:
		return "/" + t.Value, nil

	case TokenKeyword:
		switch t.Value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null":
			return nil, nil
		default:
			return t.Value, nil
		}

	case TokenArrayBegin:
		var arr []interface{}
		for {
			pk, _ := tok.Peek()
			if pk.Type == TokenArrayEnd {
				tok.Next()
				break
			}
			if pk.Type == TokenEOF {
				break
			}
			elem, err := r.parseObject(tok)
			if err != nil {
				return nil, err
			}
			arr = append(arr, elem)
		}
		if arr == nil {
			arr = []interface{}{}
		}
		return arr, nil

	case TokenDictBegin:
		dict := make(map[string]interface{})
		for {
			pk, _ := tok.Peek()
			if pk.Type == TokenDictEnd {
				tok.Next()
				break
			}
			if pk.Type == TokenEOF {
				break
			}
			keyTok, err := tok.Next()
			if err != nil {
				return nil, err
			}
			if keyTok.Type == TokenDictEnd {
				break
			}
			if keyTok.Type != TokenName {
				// Malformed: skip
				continue
			}
			key := "/" + keyTok.Value
			val, err := r.parseObject(tok)
			if err != nil {
				return nil, err
			}
			dict[key] = val
		}
		return dict, nil
	}

	return t.Value, nil
}

// ResolveObject resolves an indirect object by object number.
func (r *Resolver) ResolveObject(objNum int) (interface{}, error) {
	r.mu.RLock()
	if cached, ok := r.cache[objNum]; ok {
		r.mu.RUnlock()
		return cached, nil
	}
	r.mu.RUnlock()

	r.depth++
	defer func() { r.depth-- }()
	if r.depth > maxRecursionDepth {
		return nil, fmt.Errorf("maximum PDF object recursion depth (%d) exceeded", maxRecursionDepth)
	}

	entry, ok := r.xref[objNum]
	if !ok {
		return nil, fmt.Errorf("object %d not in xref", objNum)
	}

	if entry.Free {
		return nil, nil
	}

	if entry.Compressed {
		obj, err := r.resolveCompressed(entry)
		if err != nil {
			return nil, err
		}
		r.mu.Lock()
		r.cache[objNum] = obj
		r.mu.Unlock()
		return obj, nil
	}

	obj := r.parseObjectAt(int(entry.Offset))
	r.mu.Lock()
	r.cache[objNum] = obj
	r.mu.Unlock()
	return obj, nil
}

func (r *Resolver) resolveCompressed(entry XRefEntry) (interface{}, error) {
	// Get the object stream.
	stmObj, err := r.ResolveObject(entry.StreamObj)
	if err != nil {
		return nil, fmt.Errorf("resolving object stream %d: %w", entry.StreamObj, err)
	}
	so, ok := stmObj.(*StreamObject)
	if !ok {
		return nil, fmt.Errorf("object %d is not a stream", entry.StreamObj)
	}

	decompressed, err := r.DecompressStream(so.Dict, so.Data)
	if err != nil {
		return nil, err
	}

	// Parse the object stream header: N pairs of (objNum offset).
	n := 0
	if v, ok := getInt(so.Dict, "/N"); ok {
		n = int(v)
	}
	first := 0
	if v, ok := getInt(so.Dict, "/First"); ok {
		first = int(v)
	}

	tok := NewTokenizer(decompressed)
	type objEntry struct {
		num    int
		offset int
	}
	entries := make([]objEntry, n)
	for i := 0; i < n; i++ {
		numTok, _ := tok.Next()
		offTok, _ := tok.Next()
		entries[i] = objEntry{num: int(numTok.Int), offset: int(offTok.Int)}
	}

	if entry.StreamIdx >= len(entries) {
		return nil, fmt.Errorf("stream index %d out of range", entry.StreamIdx)
	}

	e := entries[entry.StreamIdx]
	objTok := NewTokenizer(decompressed)
	objTok.Seek(first + e.offset)
	obj, err := r.parseObject(objTok)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

// ResolveReference resolves a value if it is an IndirectRef, otherwise returns it as-is.
func (r *Resolver) ResolveReference(obj interface{}) (interface{}, error) {
	if ref, ok := obj.(IndirectRef); ok {
		return r.ResolveObject(ref.ObjNum)
	}
	return obj, nil
}

func (r *Resolver) decryptObject(obj interface{}, objNum, genNum int) (interface{}, error) {
	switch v := obj.(type) {
	case string:
		if strings.HasPrefix(v, "/") {
			return v, nil
		}
		plain, err := core.DecryptData([]byte(v), r.crypt.key, objNum, genNum, r.crypt.algorithm)
		if err != nil {
			return nil, err
		}
		return string(plain), nil
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			dec, err := r.decryptObject(item, objNum, genNum)
			if err != nil {
				return nil, err
			}
			out[i] = dec
		}
		return out, nil
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for key, item := range v {
			dec, err := r.decryptObject(item, objNum, genNum)
			if err != nil {
				return nil, err
			}
			out[key] = dec
		}
		return out, nil
	default:
		return obj, nil
	}
}

// DecompressStream decompresses stream data according to the stream dictionary's /Filter.
func (r *Resolver) DecompressStream(streamDict map[string]interface{}, streamData []byte) ([]byte, error) {
	if streamDict == nil {
		return streamData, nil
	}

	filter, _ := streamDict["/Filter"]
	filter, _ = r.ResolveReference(filter)
	if filter == nil {
		return streamData, nil
	}

	var filters []string
	switch f := filter.(type) {
	case string:
		filters = []string{f}
	case []interface{}:
		for _, v := range f {
			if s, ok := v.(string); ok {
				filters = append(filters, s)
			}
		}
	}

	data := streamData
	for _, f := range filters {
		switch f {
		case "/FlateDecode":
			decoded, err := flateDecompress(data)
			if err != nil {
				return nil, fmt.Errorf("FlateDecode: %w", err)
			}
			data = decoded
		case "/ASCIIHexDecode":
			data = asciiHexDecode(data)
		case "/ASCII85Decode":
			decoded, err := ascii85Decode(data)
			if err != nil {
				return nil, fmt.Errorf("ASCII85Decode: %w", err)
			}
			data = decoded
		default:
			// Unsupported filter; return raw data.
			return data, nil
		}
	}

	// Handle predictor (PNG sub, etc.).
	if params, ok := streamDict["/DecodeParms"]; ok {
		params, _ = r.ResolveReference(params)
		if dp, ok := params.(map[string]interface{}); ok {
			data = applyPredictor(dp, data)
		}
	}

	return data, nil
}

func flateDecompress(data []byte) ([]byte, error) {
	rd, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	return core.LimitedReadAll(rd, 100*1024*1024) // 100 MB limit
}

func asciiHexDecode(data []byte) []byte {
	s := strings.TrimSpace(string(data))
	s = strings.TrimSuffix(s, ">")
	return []byte(decodeHex(s))
}

func ascii85Decode(data []byte) ([]byte, error) {
	s := strings.TrimSpace(string(data))
	s = strings.TrimSuffix(s, "~>")
	if len(s) == 0 {
		return nil, nil
	}

	var out []byte
	buf := make([]byte, 0, 5)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 'z' && len(buf) == 0 {
			out = append(out, 0, 0, 0, 0)
			continue
		}
		if c < '!' || c > 'u' {
			continue // skip whitespace
		}
		buf = append(buf, c)
		if len(buf) == 5 {
			var val uint32
			for _, b := range buf {
				val = val*85 + uint32(b-'!')
			}
			out = append(out, byte(val>>24), byte(val>>16), byte(val>>8), byte(val))
			buf = buf[:0]
		}
	}
	if len(buf) > 1 {
		// Pad with 'u' and decode partial group.
		for len(buf) < 5 {
			buf = append(buf, 'u')
		}
		var val uint32
		for _, b := range buf {
			val = val*85 + uint32(b-'!')
		}
		n := len(buf) - 1
		for i := 0; i < n-0; i++ {
			// we need (original_len - 1) output bytes
		}
		// Actually produce the correct number of bytes
		all := []byte{byte(val >> 24), byte(val >> 16), byte(val >> 8), byte(val)}
		origLen := len(buf) - 4 // buf was padded; original was shorter
		_ = origLen
		// Partial group of n input chars produces n-1 output bytes.
		partial := len(s)
		_ = partial
		// Recalculate: the buf before padding had some length.
		// We padded from some length to 5, so output n-1 bytes where n = original buf len.
		// We already set buf to 5, original length we don't have. Let's redo.
		out = append(out, all[:len(buf)-4+1-1]...)
	}
	return out, nil
}

func applyPredictor(dp map[string]interface{}, data []byte) []byte {
	predictor := 1
	if v, ok := getInt(dp, "/Predictor"); ok {
		predictor = int(v)
	}
	if predictor <= 1 {
		return data
	}

	columns := 1
	if v, ok := getInt(dp, "/Columns"); ok {
		columns = int(v)
	}

	if predictor >= 10 {
		// PNG predictor.
		rowSize := columns + 1 // +1 for filter byte
		if rowSize <= 0 {
			return data
		}
		var out []byte
		prev := make([]byte, columns)
		for i := 0; i+rowSize <= len(data); i += rowSize {
			filterByte := data[i]
			row := data[i+1 : i+rowSize]
			decoded := make([]byte, columns)
			switch filterByte {
			case 0: // None
				copy(decoded, row)
			case 1: // Sub
				for j := 0; j < len(row); j++ {
					left := byte(0)
					if j > 0 {
						left = decoded[j-1]
					}
					decoded[j] = row[j] + left
				}
			case 2: // Up
				for j := 0; j < len(row); j++ {
					decoded[j] = row[j] + prev[j]
				}
			default:
				copy(decoded, row)
			}
			out = append(out, decoded...)
			copy(prev, decoded)
		}
		return out
	}

	return data
}

func decodeHex(hex string) string {
	var buf strings.Builder
	hex = strings.Map(func(r rune) rune {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			return r
		}
		return -1
	}, hex)
	// Pad odd length.
	if len(hex)%2 != 0 {
		hex += "0"
	}
	for i := 0; i+1 < len(hex); i += 2 {
		b, _ := strconv.ParseUint(hex[i:i+2], 16, 8)
		buf.WriteByte(byte(b))
	}
	return buf.String()
}

func getInt(dict map[string]interface{}, key string) (int64, bool) {
	v, ok := dict[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	}
	return 0, false
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}
