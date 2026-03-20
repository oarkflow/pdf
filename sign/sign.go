package sign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"os"
)

// Signer abstracts the signing operation for PDF digital signatures.
type Signer interface {
	Certificate() *x509.Certificate
	CertificateChain() []*x509.Certificate
	Sign(digest []byte) ([]byte, error)
	Algorithm() x509.SignatureAlgorithm
}

// Profile represents a PAdES conformance level.
type Profile int

const (
	PAdESBB  Profile = iota // Basic
	PAdESBT                 // With timestamp
	PAdESBLT                // Long-term
)

// Options configures the PDF signing operation.
type Options struct {
	Signer      Signer
	Profile     Profile
	Reason      string
	Location    string
	ContactInfo string
	Name        string
	TSAURL      string             // RFC 3161 timestamp authority URL
	CertStore   []*x509.Certificate // for LTV
}

// LocalSigner implements Signer using a local private key.
type LocalSigner struct {
	key       crypto.Signer
	cert      *x509.Certificate
	chain     []*x509.Certificate
	algorithm x509.SignatureAlgorithm
}

// NewLocalSigner creates a new LocalSigner from a private key and certificate chain.
// The first certificate in chain must be the signing certificate.
func NewLocalSigner(key crypto.Signer, chain []*x509.Certificate) *LocalSigner {
	if len(chain) == 0 {
		return nil
	}
	var alg x509.SignatureAlgorithm
	switch k := key.Public().(type) {
	case *rsa.PublicKey:
		alg = x509.SHA256WithRSA
	case *ecdsa.PublicKey:
		switch k.Curve {
		case elliptic.P384():
			alg = x509.ECDSAWithSHA384
		case elliptic.P521():
			alg = x509.ECDSAWithSHA512
		default:
			alg = x509.ECDSAWithSHA256
		}
	default:
		alg = x509.SHA256WithRSA
	}
	return &LocalSigner{
		key:       key,
		cert:      chain[0],
		chain:     chain,
		algorithm: alg,
	}
}

func (s *LocalSigner) Certificate() *x509.Certificate      { return s.cert }
func (s *LocalSigner) CertificateChain() []*x509.Certificate { return s.chain }
func (s *LocalSigner) Algorithm() x509.SignatureAlgorithm    { return s.algorithm }

func (s *LocalSigner) Sign(digest []byte) ([]byte, error) {
	h := sha256.Sum256(digest)
	return s.key.Sign(rand.Reader, h[:], crypto.SHA256)
}

// SignFile signs an existing PDF file.
func SignFile(inputPath, outputPath string, opts Options) error {
	pdf, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("sign: read input: %w", err)
	}
	signed, err := SignBytes(pdf, opts)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, signed, 0o644)
}

// SignBytes signs PDF bytes in memory.
func SignBytes(pdf []byte, opts Options) ([]byte, error) {
	if opts.Signer == nil {
		return nil, fmt.Errorf("sign: signer is required")
	}
	return applyPAdESProfile(opts.Profile, pdf, opts.Signer, opts)
}
