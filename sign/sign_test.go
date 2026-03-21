package sign

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"strings"
	"testing"
	"time"
)

func selfSignedCert(key interface{}) *x509.Certificate {
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	var pub interface{}
	switch k := key.(type) {
	case *rsa.PrivateKey:
		pub = &k.PublicKey
	case *ecdsa.PrivateKey:
		pub = &k.PublicKey
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, key)
	cert, _ := x509.ParseCertificate(der)
	return cert
}

func TestNewLocalSignerRSA(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert := selfSignedCert(key)
	s := NewLocalSigner(key, []*x509.Certificate{cert})
	if s == nil {
		t.Fatal("NewLocalSigner returned nil")
	}
	if s.Certificate() != cert {
		t.Error("Certificate mismatch")
	}
	if s.Algorithm() != x509.SHA256WithRSA {
		t.Errorf("Algorithm = %v", s.Algorithm())
	}
}

func TestNewLocalSignerECDSA(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	cert := selfSignedCert(key)
	s := NewLocalSigner(key, []*x509.Certificate{cert})
	if s == nil {
		t.Fatal("nil")
	}
	if s.Algorithm() != x509.ECDSAWithSHA256 {
		t.Errorf("Algorithm = %v", s.Algorithm())
	}
}

func TestNewLocalSignerEmptyChain(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	s := NewLocalSigner(key, nil)
	if s != nil {
		t.Error("expected nil for empty chain")
	}
}

func TestLocalSignerSign(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert := selfSignedCert(key)
	s := NewLocalSigner(key, []*x509.Certificate{cert})
	sig, err := s.Sign([]byte("test data"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) == 0 {
		t.Error("empty signature")
	}
}

func TestPreparePDFInvalid(t *testing.T) {
	_, _, _, err := PreparePDF([]byte("not a pdf"), Options{})
	if err == nil {
		t.Error("expected error for invalid PDF")
	}
}

func TestPreparePDFByteRange(t *testing.T) {
	// Minimal valid-ish PDF
	pdf := []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog >>
endobj
xref
0 2
0000000000 65535 f
0000000009 00000 n
trailer
<< /Size 2 /Root 1 0 R >>
startxref
52
%%EOF
`)
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	cert := selfSignedCert(key)
	signer := NewLocalSigner(key, []*x509.Certificate{cert})

	prepared, byteRange, _, err := PreparePDF(pdf, Options{Signer: signer, Reason: "Test"})
	if err != nil {
		t.Fatal(err)
	}
	// byteRange[0] should be 0
	if byteRange[0] != 0 {
		t.Errorf("byteRange[0] = %d", byteRange[0])
	}
	// byteRange[2] should be after byteRange[1]
	if byteRange[2] <= byteRange[1] {
		t.Error("byteRange[2] should be > byteRange[1]")
	}
	// Total covered should equal len(prepared)
	total := byteRange[1] + (len(prepared) - byteRange[2])
	if byteRange[3] != len(prepared)-byteRange[2] {
		t.Errorf("byteRange[3] = %d, want %d", byteRange[3], len(prepared)-byteRange[2])
	}
	_ = total
}

func TestSignBytesNilSigner(t *testing.T) {
	_, err := SignBytes([]byte("%PDF-1.4"), Options{})
	if err == nil {
		t.Error("expected error for nil signer")
	}
}

func TestBuildSigDict(t *testing.T) {
	dict := buildSigDict(1, Options{Reason: "Approved", Location: "NYC", Name: "Test User"})
	if !strings.Contains(dict, "/Reason (Approved)") {
		t.Error("missing Reason")
	}
	if !strings.Contains(dict, "/Location (NYC)") {
		t.Error("missing Location")
	}
	if !strings.Contains(dict, "/Name (Test User)") {
		t.Error("missing Name")
	}
}
