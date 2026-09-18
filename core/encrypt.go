package core

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha256"
	"fmt"
)

// Package note on the AES-256 (revision 5) handler below: this implements
// Adobe's original AES-256 extension (Adobe Supplement to ISO 32000,
// "Extension Level 3", used with dictionary V=5 R=5) - a single SHA-256
// hash, not the later ISO 32000-2 (PDF 2.0) revision 6 "hardened hash"
// (Algorithm 2.B, which iterates SHA-256/384/512 at least 64 times). R5 is
// what this document layer declares (see document/encryption.go's V/R
// selection), is simpler to implement correctly, and remains supported by
// mainstream PDF readers for backward compatibility, even though newer
// writers generally prefer R6. If R6 support is needed later, it requires a
// distinct hash2B implementation and R=6 in the encryption dictionary - it
// is out of scope here.

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
		return nil, fmt.Errorf("AES-256 uses a different O/U/OE/UE structure than RC4/AES-128; use ComputeAES256SecurityHandler instead of ComputeOwnerPasswordValue")
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

// ComputeEncryptionKey computes the file encryption key per PDF spec Algorithm 2.
func ComputeEncryptionKey(config EncryptionConfig, documentID []byte) ([]byte, error) {
	if config.Algorithm == AES_256 {
		return nil, fmt.Errorf("AES-256's file key is independent of the password/document ID; use GenerateAES256FileKey instead of ComputeEncryptionKey")
	}
	userPad := PadPassword(config.UserPassword)
	oValue, err := ComputeOwnerPasswordValue(config)
	if err != nil {
		return nil, err
	}
	return computeStandardEncryptionKey(userPad, config.Permissions, documentID, config.Algorithm, oValue)
}

func computeStandardEncryptionKey(userPad []byte, permissions uint32, documentID []byte, algorithm EncryptionAlgorithm, oValue ...[]byte) ([]byte, error) {
	h := md5.New()
	h.Write(userPad)
	if len(oValue) > 0 && oValue[0] != nil {
		h.Write(oValue[0])
	}
	// Permissions as little-endian 4 bytes
	p := permissions
	h.Write([]byte{byte(p), byte(p >> 8), byte(p >> 16), byte(p >> 24)})
	h.Write(documentID)
	digest := h.Sum(nil)

	keyLen := 16 // 128-bit for both RC4-128 and AES-128
	// Revision 3: hash 50 more times
	for i := 0; i < 50; i++ {
		tmp := md5.Sum(digest[:keyLen])
		digest = tmp[:]
	}
	return digest[:keyLen], nil
}

// ComputeUserPasswordValue computes the U value per PDF spec Algorithm 4/5.
func ComputeUserPasswordValue(config EncryptionConfig, documentID []byte) ([]byte, error) {
	if config.Algorithm == AES_256 {
		return nil, fmt.Errorf("AES-256 uses a different O/U/OE/UE structure than RC4/AES-128; use ComputeAES256SecurityHandler instead of ComputeUserPasswordValue")
	}
	oValue, err := ComputeOwnerPasswordValue(config)
	if err != nil {
		return nil, err
	}
	key, err := computeStandardEncryptionKey(PadPassword(config.UserPassword), config.Permissions, documentID, config.Algorithm, oValue)
	if err != nil {
		return nil, err
	}
	return computeStandardUserPasswordValue(key, documentID)
}

func computeStandardUserPasswordValue(key, documentID []byte) ([]byte, error) {
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

// AuthenticateUserPassword validates a Standard Security handler user password
// for RC4-128 or AES-128 documents and returns the file encryption key.
func AuthenticateUserPassword(password string, algorithm EncryptionAlgorithm, oValue, uValue []byte, permissions uint32, documentID []byte) ([]byte, bool, error) {
	if algorithm == AES_256 {
		return nil, false, fmt.Errorf("AES-256 uses UE in addition to U; use AuthenticateAES256UserPassword instead of AuthenticateUserPassword")
	}
	key, err := computeStandardEncryptionKey(PadPassword(password), permissions, documentID, algorithm, oValue)
	if err != nil {
		return nil, false, err
	}
	expectedU, err := computeStandardUserPasswordValue(key, documentID)
	if err != nil {
		return nil, false, err
	}
	if len(uValue) >= 16 && bytes.Equal(expectedU[:16], uValue[:16]) {
		return key, true, nil
	}
	return nil, false, nil
}

// AuthenticateOwnerPassword validates an owner password for RC4-128 or AES-128
// documents and returns the file encryption key.
func AuthenticateOwnerPassword(password string, algorithm EncryptionAlgorithm, oValue, uValue []byte, permissions uint32, documentID []byte) ([]byte, bool, error) {
	if algorithm == AES_256 {
		return nil, false, fmt.Errorf("AES-256 uses OE in addition to O/U; use AuthenticateAES256OwnerPassword instead of AuthenticateOwnerPassword")
	}
	userPad, err := recoverUserPadFromOwnerPassword(password, oValue)
	if err != nil {
		return nil, false, err
	}
	key, err := computeStandardEncryptionKey(userPad, permissions, documentID, algorithm, oValue)
	if err != nil {
		return nil, false, err
	}
	expectedU, err := computeStandardUserPasswordValue(key, documentID)
	if err != nil {
		return nil, false, err
	}
	if len(uValue) >= 16 && bytes.Equal(expectedU[:16], uValue[:16]) {
		return key, true, nil
	}
	return nil, false, nil
}

func recoverUserPadFromOwnerPassword(password string, oValue []byte) ([]byte, error) {
	ownerPad := PadPassword(password)
	h := md5.Sum(ownerPad)
	digest := h[:]
	for i := 0; i < 50; i++ {
		tmp := md5.Sum(digest)
		digest = tmp[:]
	}
	key := digest[:16]

	decrypted := make([]byte, len(oValue))
	copy(decrypted, oValue)
	for i := 19; i >= 0; i-- {
		tmpKey := make([]byte, len(key))
		for j := range tmpKey {
			tmpKey[j] = key[j] ^ byte(i)
		}
		c, err := rc4.NewCipher(tmpKey)
		if err != nil {
			return nil, fmt.Errorf("rc4 owner decrypt round %d: %w", i, err)
		}
		c.XORKeyStream(decrypted, decrypted)
	}
	return decrypted, nil
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

// ---------------------------------------------------------------------------
// AES-256 (Standard Security Handler revision 5)
// ---------------------------------------------------------------------------

// GenerateAES256FileKey returns a random 32-byte file encryption key for the
// Standard Security Handler revision 5 (AES-256). Unlike RC4/AES-128, the
// AES-256 file key is generated independently of any password; O/U/OE/UE
// merely wrap it so it can be recovered given a correct password.
func GenerateAES256FileKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate AES-256 file key: %w", err)
	}
	return key, nil
}

// truncatePassword returns the UTF-8 bytes of password, truncated to 127
// bytes per the Standard Security Handler revision 5 password length limit -
// unlike PadPassword's fixed 32-byte padding used by revisions 2-4.
func truncatePassword(password string) []byte {
	b := []byte(password)
	if len(b) > 127 {
		return b[:127]
	}
	return b
}

// ComputeAES256SecurityHandler computes the O, U, OE, UE, and Perms entries
// for the Standard Security Handler revision 5 (Adobe's AES-256 extension -
// see the package-level note above EncryptionAlgorithm for why this is R5,
// not the later ISO 32000-2 revision 6). fileKey must be a random 32-byte
// file encryption key (see GenerateAES256FileKey).
func ComputeAES256SecurityHandler(config EncryptionConfig, fileKey []byte) (o, u, oe, ue, perms []byte, err error) {
	if len(fileKey) != 32 {
		return nil, nil, nil, nil, nil, fmt.Errorf("AES-256 file key must be 32 bytes, got %d", len(fileKey))
	}
	userPassword := truncatePassword(config.UserPassword)
	ownerPassword := truncatePassword(config.OwnerPassword)

	uValidationSalt := make([]byte, 8)
	uKeySalt := make([]byte, 8)
	if _, err = rand.Read(uValidationSalt); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("generate U validation salt: %w", err)
	}
	if _, err = rand.Read(uKeySalt); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("generate U key salt: %w", err)
	}
	uHash := sha256.Sum256(concatBytes(userPassword, uValidationSalt))
	u = concatBytes(uHash[:], uValidationSalt, uKeySalt)

	uIntermediateKey := sha256.Sum256(concatBytes(userPassword, uKeySalt))
	ue, err = aesCBCNoPad(fileKey, uIntermediateKey[:], make([]byte, 16))
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("compute UE: %w", err)
	}

	oValidationSalt := make([]byte, 8)
	oKeySalt := make([]byte, 8)
	if _, err = rand.Read(oValidationSalt); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("generate O validation salt: %w", err)
	}
	if _, err = rand.Read(oKeySalt); err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("generate O key salt: %w", err)
	}
	oHash := sha256.Sum256(concatBytes(ownerPassword, oValidationSalt, u))
	o = concatBytes(oHash[:], oValidationSalt, oKeySalt)

	oIntermediateKey := sha256.Sum256(concatBytes(ownerPassword, oKeySalt, u))
	oe, err = aesCBCNoPad(fileKey, oIntermediateKey[:], make([]byte, 16))
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("compute OE: %w", err)
	}

	perms, err = computePerms(config.Permissions, fileKey)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("compute Perms: %w", err)
	}
	return o, u, oe, ue, perms, nil
}

// AuthenticateAES256UserPassword validates a Standard Security Handler
// revision 5 user password against U and, if it matches, returns the file
// encryption key recovered from UE.
func AuthenticateAES256UserPassword(password string, u, ue []byte) ([]byte, bool, error) {
	if len(u) < 48 {
		return nil, false, fmt.Errorf("U must be at least 48 bytes, got %d", len(u))
	}
	if len(ue) != 32 {
		return nil, false, fmt.Errorf("UE must be 32 bytes, got %d", len(ue))
	}
	pw := truncatePassword(password)
	validationSalt := u[32:40]
	keySalt := u[40:48]
	expectedHash := sha256.Sum256(concatBytes(pw, validationSalt))
	if !bytes.Equal(expectedHash[:], u[:32]) {
		return nil, false, nil
	}
	intermediateKey := sha256.Sum256(concatBytes(pw, keySalt))
	fileKey, err := aesCBCNoPadDecrypt(ue, intermediateKey[:], make([]byte, 16))
	if err != nil {
		return nil, false, err
	}
	return fileKey, true, nil
}

// AuthenticateAES256OwnerPassword validates a Standard Security Handler
// revision 5 owner password against O and, if it matches, returns the file
// encryption key recovered from OE.
func AuthenticateAES256OwnerPassword(password string, o, u, oe []byte) ([]byte, bool, error) {
	if len(o) < 48 {
		return nil, false, fmt.Errorf("O must be at least 48 bytes, got %d", len(o))
	}
	if len(u) < 48 {
		return nil, false, fmt.Errorf("U must be at least 48 bytes, got %d", len(u))
	}
	if len(oe) != 32 {
		return nil, false, fmt.Errorf("OE must be 32 bytes, got %d", len(oe))
	}
	pw := truncatePassword(password)
	validationSalt := o[32:40]
	keySalt := o[40:48]
	expectedHash := sha256.Sum256(concatBytes(pw, validationSalt, u[:48]))
	if !bytes.Equal(expectedHash[:], o[:32]) {
		return nil, false, nil
	}
	intermediateKey := sha256.Sum256(concatBytes(pw, keySalt, u[:48]))
	fileKey, err := aesCBCNoPadDecrypt(oe, intermediateKey[:], make([]byte, 16))
	if err != nil {
		return nil, false, err
	}
	return fileKey, true, nil
}

// computePerms encrypts the /Perms entry per Algorithm 10: the permission
// flags, an "extend to full range" marker, the EncryptMetadata flag, and a
// fixed filler, as a single AES-256 ECB (no chaining - exactly one 16-byte
// block, so no padding/IV is needed) encryption under the file key.
func computePerms(permissions uint32, fileKey []byte) ([]byte, error) {
	plain := make([]byte, 16)
	plain[0] = byte(permissions)
	plain[1] = byte(permissions >> 8)
	plain[2] = byte(permissions >> 16)
	plain[3] = byte(permissions >> 24)
	plain[4], plain[5], plain[6], plain[7] = 0xFF, 0xFF, 0xFF, 0xFF
	plain[8] = 'T' // EncryptMetadata - this library always encrypts metadata
	plain[9], plain[10], plain[11] = 'a', 'd', 'b'
	if _, err := rand.Read(plain[12:16]); err != nil {
		return nil, fmt.Errorf("generate Perms filler: %w", err)
	}
	block, err := aes.NewCipher(fileKey)
	if err != nil {
		return nil, fmt.Errorf("aes cipher for Perms: %w", err)
	}
	out := make([]byte, 16)
	block.Encrypt(out, plain) // ECB: single block, no chaining needed
	return out, nil
}

// aesCBCNoPad encrypts data (whose length must already be a multiple of the
// AES block size) with AES-CBC and no padding - used for UE/OE, which are
// always exactly 32 bytes (two blocks), so PKCS#7 padding would be wrong.
func aesCBCNoPad(data, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("aesCBCNoPad: data length %d is not a multiple of block size %d", len(data), block.BlockSize())
	}
	out := make([]byte, len(data))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, data)
	return out, nil
}

func aesCBCNoPadDecrypt(data, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(data)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("aesCBCNoPadDecrypt: data length %d is not a multiple of block size %d", len(data), block.BlockSize())
	}
	out := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, data)
	return out, nil
}

func concatBytes(parts ...[]byte) []byte {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	out := make([]byte, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
