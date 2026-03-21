package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha256"
	"fmt"
)

// EncryptionAlgorithm identifies the encryption algorithm for a PDF document.
type EncryptionAlgorithm int

const (
	// Deprecated: RC4_128 is cryptographically broken. Use AES_128 or AES_256 instead.
	RC4_128 EncryptionAlgorithm = iota
	AES_128
	AES_256
)

// EncryptionConfig holds the parameters needed to encrypt a PDF.
type EncryptionConfig struct {
	Algorithm     EncryptionAlgorithm
	OwnerPassword string
	UserPassword  string
	Permissions   uint32
}

// passwordPadding is the standard 32-byte padding defined in PDF spec Table 3.19.
var passwordPadding = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

// PadPassword pads or truncates a password to exactly 32 bytes using the
// standard PDF password padding string.
func PadPassword(password string) []byte {
	b := []byte(password)
	if len(b) >= 32 {
		return b[:32]
	}
	padded := make([]byte, 32)
	copy(padded, b)
	copy(padded[len(b):], passwordPadding)
	return padded
}

// ComputeOwnerPasswordValue computes the O value per PDF spec Algorithm 3.
func ComputeOwnerPasswordValue(config EncryptionConfig) ([]byte, error) {
	if config.Algorithm == AES_256 {
		return computeOwnerPasswordAES256(config)
	}
	// Algorithm 3 (revision 3)
	ownerPad := PadPassword(config.OwnerPassword)
	h := md5.Sum(ownerPad)
	digest := h[:]
	// Revision 3: hash 50 more times
	for i := 0; i < 50; i++ {
		tmp := md5.Sum(digest)
		digest = tmp[:]
	}
	keyLen := 16 // 128-bit
	key := digest[:keyLen]

	userPad := PadPassword(config.UserPassword)
	c, err := rc4.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("rc4 owner password: %w", err)
	}
	encrypted := make([]byte, 32)
	c.XORKeyStream(encrypted, userPad)

	// Revision 3: 19 additional rounds
	for i := 1; i <= 19; i++ {
		tmpKey := make([]byte, keyLen)
		for j := range tmpKey {
			tmpKey[j] = key[j] ^ byte(i)
		}
		c2, err := rc4.NewCipher(tmpKey)
		if err != nil {
			return nil, fmt.Errorf("rc4 owner password round %d: %w", i, err)
		}
		c2.XORKeyStream(encrypted, encrypted)
	}
	return encrypted, nil
}

func computeOwnerPasswordAES256(config EncryptionConfig) ([]byte, error) {
	// PDF 2.0 / Extension level 5 simplified O computation.
	// O = SHA-256(userPad + ownerValidationSalt + U)
	// For generation we embed random validation+key salts.
	oVal := make([]byte, 48) // 32 hash + 8 validation salt + 8 key salt
	if _, err := rand.Read(oVal[32:]); err != nil {
		return nil, fmt.Errorf("generate owner salt: %w", err)
	}
	// We'll fill the hash portion after U is known; caller handles this.
	// For now, return placeholder that gets completed in the document layer.
	return oVal, nil
}

// ComputeEncryptionKey computes the file encryption key per PDF spec Algorithm 2.
func ComputeEncryptionKey(config EncryptionConfig, documentID []byte) ([]byte, error) {
	if config.Algorithm == AES_256 {
		return computeEncryptionKeyAES256(config)
	}
	userPad := PadPassword(config.UserPassword)
	h := md5.New()
	h.Write(userPad)
	oValue, err := ComputeOwnerPasswordValue(config)
	if err != nil {
		return nil, err
	}
	h.Write(oValue)
	// Permissions as little-endian 4 bytes
	p := config.Permissions
	h.Write([]byte{byte(p), byte(p >> 8), byte(p >> 16), byte(p >> 24)})
	h.Write(documentID)
	if config.Algorithm == AES_128 {
		// Revision 4: include additional AES marker
		h.Write([]byte{0xFF, 0xFF, 0xFF, 0xFF})
	}
	digest := h.Sum(nil)

	keyLen := 16 // 128-bit for both RC4-128 and AES-128
	// Revision 3: hash 50 more times
	for i := 0; i < 50; i++ {
		tmp := md5.Sum(digest[:keyLen])
		digest = tmp[:]
	}
	return digest[:keyLen], nil
}

func computeEncryptionKeyAES256(_ EncryptionConfig) ([]byte, error) {
	// AES-256: the file encryption key is a random 32-byte key.
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate AES-256 key: %w", err)
	}
	return key, nil
}

// ComputeUserPasswordValue computes the U value per PDF spec Algorithm 4/5.
func ComputeUserPasswordValue(config EncryptionConfig, documentID []byte) ([]byte, error) {
	if config.Algorithm == AES_256 {
		return computeUserPasswordAES256(config)
	}
	key, err := ComputeEncryptionKey(config, documentID)
	if err != nil {
		return nil, err
	}

	// Algorithm 5 (revision 3): MD5 of padding + document ID, then RC4 rounds
	h := md5.New()
	h.Write(passwordPadding)
	h.Write(documentID)
	digest := h.Sum(nil)

	c, err := rc4.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("rc4 user password: %w", err)
	}
	c.XORKeyStream(digest, digest)

	for i := 1; i <= 19; i++ {
		tmpKey := make([]byte, len(key))
		for j := range tmpKey {
			tmpKey[j] = key[j] ^ byte(i)
		}
		c2, err := rc4.NewCipher(tmpKey)
		if err != nil {
			return nil, fmt.Errorf("rc4 user password round %d: %w", i, err)
		}
		c2.XORKeyStream(digest, digest)
	}
	// Pad to 32 bytes with arbitrary data
	result := make([]byte, 32)
	copy(result, digest)
	if _, err := rand.Read(result[16:]); err != nil {
		return nil, fmt.Errorf("generate user password padding: %w", err)
	}
	return result, nil
}

func computeUserPasswordAES256(config EncryptionConfig) ([]byte, error) {
	// Simplified AES-256: U = SHA-256(password + validationSalt) + salts
	userPad := PadPassword(config.UserPassword)
	uVal := make([]byte, 48)
	if _, err := rand.Read(uVal[32:]); err != nil {
		return nil, fmt.Errorf("generate user salt: %w", err)
	}
	hash := sha256.Sum256(append(userPad, uVal[32:40]...))
	copy(uVal, hash[:])
	return uVal, nil
}

// EncryptData encrypts data using the file encryption key per Algorithm 1.
// objNum and genNum are used to derive the per-object key.
func EncryptData(data, key []byte, objNum, genNum int, algorithm EncryptionAlgorithm) ([]byte, error) {
	if algorithm == AES_256 {
		// AES-256: use key directly (no per-object derivation)
		return encryptAES(data, key)
	}

	// Algorithm 1: derive per-object key
	h := md5.New()
	h.Write(key)
	// Object number as 3 little-endian bytes
	h.Write([]byte{byte(objNum), byte(objNum >> 8), byte(objNum >> 16)})
	// Generation number as 2 little-endian bytes
	h.Write([]byte{byte(genNum), byte(genNum >> 8)})
	if algorithm == AES_128 {
		h.Write([]byte("sAlT")) // AES salt marker per spec
	}
	objKey := h.Sum(nil)
	// Key length: min(keyLen+5, 16)
	objKeyLen := len(key) + 5
	if objKeyLen > 16 {
		objKeyLen = 16
	}
	objKey = objKey[:objKeyLen]

	switch algorithm {
	case RC4_128:
		c, err := rc4.NewCipher(objKey)
		if err != nil {
			return nil, fmt.Errorf("rc4 encrypt: %w", err)
		}
		out := make([]byte, len(data))
		c.XORKeyStream(out, data)
		return out, nil
	case AES_128:
		return encryptAES(data, objKey)
	default:
		return data, nil
	}
}

// encryptAES encrypts data with AES-CBC, prepending a random 16-byte IV.
// Data is PKCS#7 padded to a multiple of the block size.
func encryptAES(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes encrypt: %w", err)
	}
	// PKCS#7 padding
	blockSize := block.BlockSize()
	padLen := blockSize - (len(data) % blockSize)
	padded := make([]byte, len(data)+padLen)
	copy(padded, data)
	for i := len(data); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	// Random IV
	iv := make([]byte, blockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("generate AES IV: %w", err)
	}

	cbc := cipher.NewCBCEncrypter(block, iv)
	cbc.CryptBlocks(padded, padded)

	// Prepend IV
	result := make([]byte, blockSize+len(padded))
	copy(result, iv)
	copy(result[blockSize:], padded)
	return result, nil
}

// DecryptData decrypts data that was encrypted with EncryptData.
// This is useful for testing round-trips.
func DecryptData(data, key []byte, objNum, genNum int, algorithm EncryptionAlgorithm) ([]byte, error) {
	if algorithm == AES_256 {
		return decryptAES(data, key)
	}

	// Derive per-object key (same as EncryptData)
	h := md5.New()
	h.Write(key)
	h.Write([]byte{byte(objNum), byte(objNum >> 8), byte(objNum >> 16)})
	h.Write([]byte{byte(genNum), byte(genNum >> 8)})
	if algorithm == AES_128 {
		h.Write([]byte("sAlT"))
	}
	objKey := h.Sum(nil)
	objKeyLen := len(key) + 5
	if objKeyLen > 16 {
		objKeyLen = 16
	}
	objKey = objKey[:objKeyLen]

	switch algorithm {
	case RC4_128:
		c, err := rc4.NewCipher(objKey)
		if err != nil {
			return nil, fmt.Errorf("rc4 decrypt: %w", err)
		}
		out := make([]byte, len(data))
		c.XORKeyStream(out, data)
		return out, nil
	case AES_128:
		return decryptAES(data, objKey)
	default:
		return data, nil
	}
}

func decryptAES(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes decrypt: %w", err)
	}
	blockSize := block.BlockSize()
	if len(data) < 2*blockSize {
		return nil, fmt.Errorf("aes decrypt: ciphertext too short")
	}
	iv := data[:blockSize]
	ct := make([]byte, len(data)-blockSize)
	copy(ct, data[blockSize:])

	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(ct, ct)

	// Remove PKCS#7 padding
	if len(ct) > 0 {
		padLen := int(ct[len(ct)-1])
		if padLen > 0 && padLen <= blockSize && padLen <= len(ct) {
			ct = ct[:len(ct)-padLen]
		}
	}
	return ct, nil
}
