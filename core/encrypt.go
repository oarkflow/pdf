package core

// EncryptionAlgorithm identifies the encryption algorithm for a PDF document.
type EncryptionAlgorithm int

const (
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
