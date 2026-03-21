package document

import (
	"crypto/rand"
	"fmt"

	"github.com/oarkflow/pdf/core"
)

// applyEncryption adds encryption objects to the writer.
func (d *Document) applyEncryption(w *Writer) error {
	cfg := *d.encConfig

	// 1. Generate a random 16-byte document ID.
	docID := make([]byte, 16)
	if _, err := rand.Read(docID); err != nil {
		return fmt.Errorf("generate document ID: %w", err)
	}

	// 2. Compute O and U values.
	oValue, err := core.ComputeOwnerPasswordValue(cfg)
	if err != nil {
		return fmt.Errorf("compute owner password: %w", err)
	}
	uValue, err := core.ComputeUserPasswordValue(cfg, docID)
	if err != nil {
		return fmt.Errorf("compute user password: %w", err)
	}

	// 3. Compute encryption key.
	encKey, err := core.ComputeEncryptionKey(cfg, docID)
	if err != nil {
		return fmt.Errorf("compute encryption key: %w", err)
	}

	// 4. Determine V, R, Length, SubFilter based on algorithm.
	var v, r, keyLength int
	var subFilter string
	switch cfg.Algorithm {
	case core.RC4_128:
		v = 2
		r = 3
		keyLength = 128
		subFilter = "Standard"
	case core.AES_128:
		v = 4
		r = 4
		keyLength = 128
		subFilter = "Standard"
	case core.AES_256:
		v = 5
		r = 5
		keyLength = 256
		subFilter = "Standard"
	}

	// 5. Build /Encrypt dictionary.
	encDict := core.NewDictionary()
	encDict.Set("Filter", core.PdfName("Standard"))
	encDict.Set("SubFilter", core.PdfName(subFilter))
	encDict.Set("V", core.PdfInteger(v))
	encDict.Set("Length", core.PdfInteger(keyLength))
	encDict.Set("R", core.PdfInteger(r))
	encDict.Set("O", core.PdfHexString(oValue))
	encDict.Set("U", core.PdfHexString(uValue))
	encDict.Set("P", core.PdfInteger(int64(cfg.Permissions)))

	if cfg.Algorithm == AES_128 || cfg.Algorithm == AES_256 {
		// Add CF (crypt filter) dictionary for AES
		stdCF := core.NewDictionary()
		stdCF.Set("Type", core.PdfName("CryptFilter"))
		if cfg.Algorithm == AES_128 {
			stdCF.Set("CFM", core.PdfName("AESV2"))
		} else {
			stdCF.Set("CFM", core.PdfName("AESV3"))
		}
		stdCF.Set("AuthEvent", core.PdfName("DocOpen"))
		stdCF.Set("Length", core.PdfInteger(keyLength/8))

		cfDict := core.NewDictionary()
		cfDict.Set("StdCF", stdCF)
		encDict.Set("CF", cfDict)
		encDict.Set("StmF", core.PdfName("StdCF"))
		encDict.Set("StrF", core.PdfName("StdCF"))
	}

	// 6. Add encrypt dict to writer as an indirect object.
	encObjNum := w.AddObject(encDict)

	// 7. Store encryption info on writer for stream/string encryption during WriteTo.
	w.SetEncryption(encKey, cfg.Algorithm, encObjNum, docID)
	return nil
}

// AES_128 and AES_256 are re-exported for convenience within the package.
const (
	AES_128 = core.AES_128
	AES_256 = core.AES_256
)
