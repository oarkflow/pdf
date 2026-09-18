package core

import (
	"bytes"
	"testing"
)

func TestEncryptionAlgorithmConstants(t *testing.T) {
	if RC4_128 != 0 {
		t.Errorf("RC4_128 = %d, want 0", RC4_128)
	}
	if AES_128 != 1 {
		t.Errorf("AES_128 = %d, want 1", AES_128)
	}
	if AES_256 != 2 {
		t.Errorf("AES_256 = %d, want 2", AES_256)
	}
}

func TestEncryptionConfigDefaults(t *testing.T) {
	cfg := EncryptionConfig{}
	if cfg.Algorithm != RC4_128 {
		t.Error("default algorithm should be RC4_128")
	}
	if cfg.OwnerPassword != "" || cfg.UserPassword != "" {
		t.Error("default passwords should be empty")
	}
	if cfg.Permissions != 0 {
		t.Error("default permissions should be 0")
	}
}

func TestEncryptionConfigWithValues(t *testing.T) {
	cfg := EncryptionConfig{
		Algorithm:     AES_256,
		OwnerPassword: "owner",
		UserPassword:  "user",
		Permissions:   0xFFFFF0C4,
	}
	if cfg.Algorithm != AES_256 {
		t.Error("algorithm mismatch")
	}
	if cfg.OwnerPassword != "owner" {
		t.Error("owner password mismatch")
	}
	if cfg.UserPassword != "user" {
		t.Error("user password mismatch")
	}
	if cfg.Permissions != 0xFFFFF0C4 {
		t.Error("permissions mismatch")
	}
}

func TestEncryptionConfigAES128(t *testing.T) {
	cfg := EncryptionConfig{Algorithm: AES_128}
	if cfg.Algorithm != AES_128 {
		t.Errorf("algorithm = %d, want AES_128(%d)", cfg.Algorithm, AES_128)
	}
}

func TestEncryptionConfigEmptyPasswords(t *testing.T) {
	cfg := EncryptionConfig{
		Algorithm:     AES_256,
		OwnerPassword: "",
		UserPassword:  "",
	}
	if cfg.OwnerPassword != "" || cfg.UserPassword != "" {
		t.Error("passwords should be empty")
	}
}

func TestPadPassword(t *testing.T) {
	// Empty password should be the full padding
	p := PadPassword("")
	if len(p) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(p))
	}
	if !bytes.Equal(p, passwordPadding) {
		t.Error("empty password should equal padding string")
	}

	// Short password
	p2 := PadPassword("test")
	if len(p2) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(p2))
	}
	if string(p2[:4]) != "test" {
		t.Error("password prefix mismatch")
	}
	if !bytes.Equal(p2[4:], passwordPadding[:28]) {
		t.Error("padding suffix mismatch")
	}

	// Long password (>32 bytes) should be truncated
	long := "abcdefghijklmnopqrstuvwxyz1234567890"
	p3 := PadPassword(long)
	if len(p3) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(p3))
	}
	if string(p3) != long[:32] {
		t.Error("long password should be truncated to 32 bytes")
	}
}

func TestComputeOwnerPasswordValue(t *testing.T) {
	cfg := EncryptionConfig{
		Algorithm:     RC4_128,
		OwnerPassword: "owner",
		UserPassword:  "user",
		Permissions:   0xFFFFF0C4,
	}
	o, err := ComputeOwnerPasswordValue(cfg)
	if err != nil {
		t.Fatalf("ComputeOwnerPasswordValue: %v", err)
	}
	if len(o) != 32 {
		t.Fatalf("O value should be 32 bytes, got %d", len(o))
	}

	// Same inputs should produce same output (deterministic)
	o2, err := ComputeOwnerPasswordValue(cfg)
	if err != nil {
		t.Fatalf("ComputeOwnerPasswordValue: %v", err)
	}
	if !bytes.Equal(o, o2) {
		t.Error("O value should be deterministic for same inputs")
	}

	// Different passwords should produce different O values
	cfg2 := cfg
	cfg2.OwnerPassword = "different"
	o3, err := ComputeOwnerPasswordValue(cfg2)
	if err != nil {
		t.Fatalf("ComputeOwnerPasswordValue: %v", err)
	}
	if bytes.Equal(o, o3) {
		t.Error("different owner passwords should produce different O values")
	}
}

func TestComputeUserPasswordValue(t *testing.T) {
	docID := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10}
	cfg := EncryptionConfig{
		Algorithm:     RC4_128,
		OwnerPassword: "owner",
		UserPassword:  "user",
		Permissions:   0xFFFFF0C4,
	}
	u, err := ComputeUserPasswordValue(cfg, docID)
	if err != nil {
		t.Fatalf("ComputeUserPasswordValue: %v", err)
	}
	if len(u) != 32 {
		t.Fatalf("U value should be 32 bytes, got %d", len(u))
	}

	// First 16 bytes should be deterministic
	u2, err := ComputeUserPasswordValue(cfg, docID)
	if err != nil {
		t.Fatalf("ComputeUserPasswordValue: %v", err)
	}
	if !bytes.Equal(u[:16], u2[:16]) {
		t.Error("first 16 bytes of U should be deterministic")
	}
}

func TestAES128KeyDerivationMatchesRC4WhenMetadataEncrypted(t *testing.T) {
	docID := []byte("1234567890abcdef")

	rc4Cfg := EncryptionConfig{
		Algorithm:     RC4_128,
		OwnerPassword: "owner",
		UserPassword:  "user",
		Permissions:   0xFFFFF0C4,
	}
	aesCfg := rc4Cfg
	aesCfg.Algorithm = AES_128

	rc4Key, err := ComputeEncryptionKey(rc4Cfg, docID)
	if err != nil {
		t.Fatalf("ComputeEncryptionKey RC4: %v", err)
	}
	aesKey, err := ComputeEncryptionKey(aesCfg, docID)
	if err != nil {
		t.Fatalf("ComputeEncryptionKey AES-128: %v", err)
	}
	if !bytes.Equal(rc4Key, aesKey) {
		t.Fatal("expected AES-128 key derivation to match RC4 when EncryptMetadata=true")
	}
}

func TestEncryptDecryptRC4(t *testing.T) {
	key := make([]byte, 16)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("Hello, World! This is a test of RC4 encryption.")

	encrypted, err := EncryptData(plaintext, key, 1, 0, RC4_128)
	if err != nil {
		t.Fatalf("EncryptData: %v", err)
	}
	if bytes.Equal(encrypted, plaintext) {
		t.Error("encrypted data should differ from plaintext")
	}

	decrypted, err := DecryptData(encrypted, key, 1, 0, RC4_128)
	if err != nil {
		t.Fatalf("DecryptData: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecryptAES128(t *testing.T) {
	key := make([]byte, 16)
	for i := range key {
		key[i] = byte(i * 3)
	}
	plaintext := []byte("AES-128 encryption test data for PDF.")

	encrypted, err := EncryptData(plaintext, key, 5, 0, AES_128)
	if err != nil {
		t.Fatalf("EncryptData: %v", err)
	}
	if bytes.Equal(encrypted, plaintext) {
		t.Error("encrypted data should differ from plaintext")
	}
	if len(encrypted) < 16 {
		t.Fatal("encrypted data too short for AES (missing IV)")
	}

	decrypted, err := DecryptData(encrypted, key, 5, 0, AES_128)
	if err != nil {
		t.Fatalf("DecryptData: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("AES-128 round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecryptAES256(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i * 7)
	}
	plaintext := []byte("AES-256 encryption test with a longer key size.")

	encrypted, err := EncryptData(plaintext, key, 10, 0, AES_256)
	if err != nil {
		t.Fatalf("EncryptData: %v", err)
	}
	if bytes.Equal(encrypted, plaintext) {
		t.Error("encrypted data should differ from plaintext")
	}

	decrypted, err := DecryptData(encrypted, key, 10, 0, AES_256)
	if err != nil {
		t.Fatalf("DecryptData: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("AES-256 round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptionKeyLength(t *testing.T) {
	docID := make([]byte, 16)
	cfg := EncryptionConfig{
		Algorithm:     RC4_128,
		OwnerPassword: "owner",
		UserPassword:  "user",
		Permissions:   0xFFFFF0C4,
	}
	key, err := ComputeEncryptionKey(cfg, docID)
	if err != nil {
		t.Fatalf("ComputeEncryptionKey: %v", err)
	}
	if len(key) != 16 {
		t.Errorf("RC4-128 key should be 16 bytes, got %d", len(key))
	}

	cfg.Algorithm = AES_128
	key, err = ComputeEncryptionKey(cfg, docID)
	if err != nil {
		t.Fatalf("ComputeEncryptionKey: %v", err)
	}
	if len(key) != 16 {
		t.Errorf("AES-128 key should be 16 bytes, got %d", len(key))
	}

	// AES-256 no longer goes through ComputeEncryptionKey - its file key is
	// independent of the password/document ID (see GenerateAES256FileKey and
	// TestAES256SecurityHandlerRoundTrip below). Calling it with AES_256
	// should fail clearly rather than silently return a placeholder key.
	cfg.Algorithm = AES_256
	if _, err = ComputeEncryptionKey(cfg, docID); err == nil {
		t.Fatal("expected ComputeEncryptionKey to reject AES_256, got nil error")
	}
}

// TestAES256SecurityHandlerRoundTrip is the primary correctness test for
// the AES-256 (revision 5) Standard Security Handler: it generates a file
// key, computes O/U/OE/UE/Perms for distinct user/owner passwords, and
// verifies that both the correct user password and the correct owner
// password each independently recover the exact same file key, while an
// incorrect password recovers nothing.
func TestAES256SecurityHandlerRoundTrip(t *testing.T) {
	fileKey, err := GenerateAES256FileKey()
	if err != nil {
		t.Fatalf("GenerateAES256FileKey: %v", err)
	}
	if len(fileKey) != 32 {
		t.Fatalf("file key should be 32 bytes, got %d", len(fileKey))
	}

	cfg := EncryptionConfig{
		Algorithm:     AES_256,
		OwnerPassword: "owner-secret",
		UserPassword:  "user-secret",
		Permissions:   0xFFFFF0C4,
	}
	o, u, oe, ue, perms, err := ComputeAES256SecurityHandler(cfg, fileKey)
	if err != nil {
		t.Fatalf("ComputeAES256SecurityHandler: %v", err)
	}
	for name, v := range map[string][]byte{"O": o, "U": u} {
		if len(v) != 48 {
			t.Fatalf("%s should be 48 bytes, got %d", name, len(v))
		}
	}
	for name, v := range map[string][]byte{"OE": oe, "UE": ue} {
		if len(v) != 32 {
			t.Fatalf("%s should be 32 bytes, got %d", name, len(v))
		}
	}
	if len(perms) != 16 {
		t.Fatalf("Perms should be 16 bytes, got %d", len(perms))
	}

	// Correct user password recovers the exact file key.
	recoveredByUser, matched, err := AuthenticateAES256UserPassword(cfg.UserPassword, u, ue)
	if err != nil {
		t.Fatalf("AuthenticateAES256UserPassword: %v", err)
	}
	if !matched {
		t.Fatal("expected correct user password to authenticate")
	}
	if !bytes.Equal(recoveredByUser, fileKey) {
		t.Fatalf("user password recovered wrong file key: got %x, want %x", recoveredByUser, fileKey)
	}

	// Correct owner password recovers the same file key.
	recoveredByOwner, matched, err := AuthenticateAES256OwnerPassword(cfg.OwnerPassword, o, u, oe)
	if err != nil {
		t.Fatalf("AuthenticateAES256OwnerPassword: %v", err)
	}
	if !matched {
		t.Fatal("expected correct owner password to authenticate")
	}
	if !bytes.Equal(recoveredByOwner, fileKey) {
		t.Fatalf("owner password recovered wrong file key: got %x, want %x", recoveredByOwner, fileKey)
	}

	// Wrong passwords must not authenticate.
	if _, matched, err := AuthenticateAES256UserPassword("wrong-password", u, ue); err != nil || matched {
		t.Fatalf("expected wrong user password to fail authentication, matched=%v err=%v", matched, err)
	}
	if _, matched, err := AuthenticateAES256OwnerPassword("wrong-password", o, u, oe); err != nil || matched {
		t.Fatalf("expected wrong owner password to fail authentication, matched=%v err=%v", matched, err)
	}

	// The recovered file key must actually decrypt content encrypted with it
	// (exercising the same EncryptData/DecryptData path the document/reader
	// layers use for stream and string content).
	plaintext := []byte("AES-256 R5 content encrypted under the recovered file key.")
	encrypted, err := EncryptData(plaintext, fileKey, 1, 0, AES_256)
	if err != nil {
		t.Fatalf("EncryptData: %v", err)
	}
	decrypted, err := DecryptData(encrypted, recoveredByUser, 1, 0, AES_256)
	if err != nil {
		t.Fatalf("DecryptData: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("content round-trip with recovered key failed: got %q, want %q", decrypted, plaintext)
	}
}

// TestAES256SecurityHandlerEmptyUserPassword covers the common case of an
// empty user password (readable by anyone) with a non-empty owner password
// (needed to change permissions/re-encrypt) - the most frequent real-world
// AES-256 PDF configuration.
func TestAES256SecurityHandlerEmptyUserPassword(t *testing.T) {
	fileKey, err := GenerateAES256FileKey()
	if err != nil {
		t.Fatalf("GenerateAES256FileKey: %v", err)
	}
	cfg := EncryptionConfig{
		Algorithm:     AES_256,
		OwnerPassword: "owner-only-secret",
		UserPassword:  "",
		Permissions:   0xFFFFFFFF,
	}
	o, u, oe, ue, _, err := ComputeAES256SecurityHandler(cfg, fileKey)
	if err != nil {
		t.Fatalf("ComputeAES256SecurityHandler: %v", err)
	}
	recovered, matched, err := AuthenticateAES256UserPassword("", u, ue)
	if err != nil || !matched {
		t.Fatalf("expected empty user password to authenticate, matched=%v err=%v", matched, err)
	}
	if !bytes.Equal(recovered, fileKey) {
		t.Fatal("empty user password recovered wrong file key")
	}
	if _, matched, _ := AuthenticateAES256OwnerPassword("", o, u, oe); matched {
		t.Fatal("empty password should not authenticate as the (non-empty) owner password")
	}
}

// TestAES256SecurityHandlerRejectsWrongKeyLength guards the input
// validation on ComputeAES256SecurityHandler.
func TestAES256SecurityHandlerRejectsWrongKeyLength(t *testing.T) {
	cfg := EncryptionConfig{Algorithm: AES_256, UserPassword: "u", OwnerPassword: "o"}
	if _, _, _, _, _, err := ComputeAES256SecurityHandler(cfg, make([]byte, 16)); err == nil {
		t.Fatal("expected an error for a non-32-byte file key")
	}
}
