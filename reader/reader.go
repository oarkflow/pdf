package reader

import (
	"errors"
	"fmt"
	"os"

	"github.com/oarkflow/pdf/core"
)

// PageInfo holds information about a single PDF page.
type PageInfo struct {
	MediaBox  [4]float64
	Contents  []byte // raw content stream (decompressed)
	Resources map[string]interface{}
	Rotation  int
}

// Reader provides high-level access to a parsed PDF document.
type Reader struct {
	resolver *Resolver
	trailer  map[string]interface{}
	catalog  map[string]interface{}
	pages    []map[string]interface{}
}

// Open parses the PDF data and returns a Reader.
func Open(data []byte) (*Reader, error) {
	return OpenWithPassword(data, "")
}

// OpenWithPassword parses the PDF data and authenticates encrypted documents
// using the supplied user or owner password.
func OpenWithPassword(data []byte, password string) (*Reader, error) {
	if len(data) == 0 {
		return nil, errors.New("reader: data is empty")
	}
	resolver, err := NewResolver(data)
	if err != nil {
		return nil, fmt.Errorf("reader: %w", err)
	}

	trailer, err := resolver.Trailer()
	if err != nil {
		return nil, fmt.Errorf("reader: %w", err)
	}

	if err := configureDecryption(resolver, trailer, password); err != nil {
		return nil, fmt.Errorf("reader: %w", err)
	}

	rootRef := trailer["/Root"]
	rootObj, err := resolver.ResolveReference(rootRef)
	if err != nil {
		return nil, fmt.Errorf("reader: resolving catalog: %w", err)
	}
	catalog, ok := rootObj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("reader: catalog is not a dictionary")
	}

	r := &Reader{
		resolver: resolver,
		trailer:  trailer,
		catalog:  catalog,
	}

	if err := r.buildPageList(); err != nil {
		return nil, fmt.Errorf("reader: building page list: %w", err)
	}

	return r, nil
}

// OpenFile reads a PDF file and returns a Reader.
func OpenFile(path string) (*Reader, error) {
	return OpenFileWithPassword(path, "")
}

// OpenFileWithPassword reads a PDF file and returns a Reader.
func OpenFileWithPassword(path string, password string) (*Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return OpenWithPassword(data, password)
}

// NumPages returns the number of pages in the document.
func (r *Reader) NumPages() int {
	return len(r.pages)
}

// Page returns information for the 0-indexed page.
func (r *Reader) Page(n int) (*PageInfo, error) {
	if n < 0 || n >= len(r.pages) {
		return nil, fmt.Errorf("page %d out of range [0, %d)", n, len(r.pages))
	}

	pageDict := r.pages[n]

	info := &PageInfo{}

	// MediaBox.
	mb, err := r.inheritedAttr(pageDict, "/MediaBox")
	if err != nil {
		return nil, fmt.Errorf("reader: page %d: resolving /MediaBox: %w", n, err)
	}
	if arr, ok := mb.([]interface{}); ok && len(arr) >= 4 {
		for i := 0; i < 4; i++ {
			info.MediaBox[i] = toFloat(arr[i])
		}
	}

	// Rotation.
	rot, err := r.inheritedAttr(pageDict, "/Rotate")
	if err != nil {
		return nil, fmt.Errorf("reader: page %d: resolving /Rotate: %w", n, err)
	}
	if v, ok := rot.(int64); ok {
		info.Rotation = int(v)
	}

	// Resources.
	res, err := r.inheritedAttr(pageDict, "/Resources")
	if err != nil {
		return nil, fmt.Errorf("reader: page %d: resolving /Resources: %w", n, err)
	}
	if resRef, ok := res.(IndirectRef); ok {
		resolved, err := r.resolver.ResolveObject(resRef.ObjNum)
		if err != nil {
			return nil, fmt.Errorf("reader: failed to resolve resources: %w", err)
		}
		if d, ok := resolved.(map[string]interface{}); ok {
			res = d
		}
	}
	if d, ok := res.(map[string]interface{}); ok {
		info.Resources = d
	} else {
		info.Resources = make(map[string]interface{})
	}

	// Contents.
	contents, err := r.resolver.ResolveReference(pageDict["/Contents"])
	if err != nil {
		return nil, fmt.Errorf("reader: failed to resolve /Contents: %w", err)
	}
	if contents != nil {
		data, err := r.extractContents(contents)
		if err != nil {
			return nil, fmt.Errorf("extracting contents for page %d: %w", n, err)
		}
		info.Contents = data
	}

	return info, nil
}

func (r *Reader) extractContents(contents interface{}) ([]byte, error) {
	switch c := contents.(type) {
	case *StreamObject:
		return r.resolver.DecompressStream(c.Dict, c.Data)
	case IndirectRef:
		obj, err := r.resolver.ResolveObject(c.ObjNum)
		if err != nil {
			return nil, err
		}
		return r.extractContents(obj)
	case []interface{}:
		// Array of content streams.
		var all []byte
		for _, item := range c {
			resolved, err := r.resolver.ResolveReference(item)
			if err != nil {
				return nil, fmt.Errorf("reader: failed to resolve content stream item: %w", err)
			}
			part, err := r.extractContents(resolved)
			if err != nil {
				return nil, err
			}
			if len(all) > 0 {
				all = append(all, '\n')
			}
			all = append(all, part...)
		}
		return all, nil
	}
	return nil, nil
}

// Metadata returns the document info dictionary as a string map.
func (r *Reader) Metadata() map[string]string {
	result := make(map[string]string)
	infoRef, ok := r.trailer["/Info"]
	if !ok {
		return result
	}
	infoObj, err := r.resolver.ResolveReference(infoRef)
	if err != nil {
		return result
	}
	infoDict, ok := infoObj.(map[string]interface{})
	if !ok {
		return result
	}
	for k, v := range infoDict {
		if s, ok := v.(string); ok {
			// Strip the leading / from the key for the result map.
			key := k
			if len(key) > 0 && key[0] == '/' {
				key = key[1:]
			}
			result[key] = s
		}
	}
	return result
}

// Trailer returns the raw trailer dictionary.
func (r *Reader) Trailer() map[string]interface{} {
	return r.trailer
}

// Catalog returns the document catalog dictionary.
func (r *Reader) Catalog() map[string]interface{} {
	return r.catalog
}

// GetResolver returns the underlying resolver for advanced use.
func (r *Reader) GetResolver() *Resolver {
	return r.resolver
}

func configureDecryption(resolver *Resolver, trailer map[string]interface{}, password string) error {
	encRef, ok := trailer["/Encrypt"]
	if !ok {
		return nil
	}

	ref, ok := encRef.(IndirectRef)
	if !ok {
		return fmt.Errorf("invalid /Encrypt reference")
	}
	encObj, err := resolver.ResolveObject(ref.ObjNum)
	if err != nil {
		return fmt.Errorf("resolving /Encrypt: %w", err)
	}
	encDict, ok := encObj.(map[string]interface{})
	if !ok {
		return fmt.Errorf("/Encrypt is not a dictionary")
	}

	v, _ := getInt(encDict, "/V")
	if v == 5 {
		return fmt.Errorf("AES-256 encrypted PDFs are not supported yet")
	}

	var algorithm core.EncryptionAlgorithm
	switch v {
	case 2:
		algorithm = core.RC4_128
	case 4:
		algorithm = core.AES_128
	default:
		return fmt.Errorf("unsupported encryption version %d", v)
	}

	oValue, ok := encDict["/O"].(string)
	if !ok {
		return fmt.Errorf("missing /O in encryption dictionary")
	}
	uValue, ok := encDict["/U"].(string)
	if !ok {
		return fmt.Errorf("missing /U in encryption dictionary")
	}
	pInt, ok := getInt(encDict, "/P")
	if !ok {
		return fmt.Errorf("missing /P in encryption dictionary")
	}

	idArr, ok := trailer["/ID"].([]interface{})
	if !ok || len(idArr) == 0 {
		return fmt.Errorf("missing trailer /ID")
	}
	docID, ok := idArr[0].(string)
	if !ok {
		return fmt.Errorf("invalid trailer /ID")
	}

	key, matched, err := core.AuthenticateUserPassword(password, algorithm, []byte(oValue), []byte(uValue), uint32(pInt), []byte(docID))
	if err != nil {
		return err
	}
	if !matched {
		key, matched, err = core.AuthenticateOwnerPassword(password, algorithm, []byte(oValue), []byte(uValue), uint32(pInt), []byte(docID))
		if err != nil {
			return err
		}
	}
	if !matched {
		return fmt.Errorf("invalid password for encrypted PDF")
	}

	resolver.crypt = &decryptState{
		key:           key,
		algorithm:     algorithm,
		encryptObjNum: ref.ObjNum,
	}
	return nil
}

func (r *Reader) buildPageList() error {
	pagesRef := r.catalog["/Pages"]
	pagesObj, err := r.resolver.ResolveReference(pagesRef)
	if err != nil {
		return err
	}
	pagesDict, ok := pagesObj.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Pages is not a dictionary")
	}
	return r.collectPages(pagesDict)
}

func (r *Reader) collectPages(node map[string]interface{}) error {
	typ, _ := node["/Type"].(string)

	if typ == "/Page" {
		r.pages = append(r.pages, node)
		return nil
	}

	// Pages node.
	kids, _ := node["/Kids"].([]interface{})
	for _, kid := range kids {
		kidObj, err := r.resolver.ResolveReference(kid)
		if err != nil {
			continue
		}
		kidDict, ok := kidObj.(map[string]interface{})
		if !ok {
			continue
		}
		// Store parent reference for inheritance.
		if _, exists := kidDict["_parent"]; !exists {
			kidDict["_parent"] = node
		}
		if err := r.collectPages(kidDict); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reader) inheritedAttr(pageDict map[string]interface{}, key string) (interface{}, error) {
	node := pageDict
	for node != nil {
		if v, ok := node[key]; ok {
			resolved, err := r.resolver.ResolveReference(v)
			if err != nil {
				return nil, fmt.Errorf("reader: failed to resolve inherited attr %s: %w", key, err)
			}
			return resolved, nil
		}
		parent, _ := node["_parent"].(map[string]interface{})
		node = parent
	}
	return nil, nil
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case int64:
		return float64(n)
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}
