package sign

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"math/big"
	"time"
)

// OID constants for CMS/PKCS#7 construction.
var (
	oidSignedData     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidData           = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidSHA256         = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	oidRSAEncryption  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	oidSigningTime    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}
	oidMessageDigest  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	oidContentType    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidTimestampToken = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 14}
)

// ASN.1 structures for CMS SignedData.

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,tag:0"`
}

type signedData struct {
	Version                    int
	DigestAlgorithmIdentifiers asn1.RawValue `asn1:"set"`
	EncapContentInfo           encapContentInfo
	Certificates               asn1.RawValue `asn1:"optional,tag:0"`
	SignerInfos                asn1.RawValue `asn1:"set"`
}

type encapContentInfo struct {
	EContentType asn1.ObjectIdentifier
}

type algorithmIdentifier struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.RawValue `asn1:"optional"`
}

type issuerAndSerialNumber struct {
	Issuer       asn1.RawValue
	SerialNumber *big.Int
}

type attribute struct {
	Type   asn1.ObjectIdentifier
	Values asn1.RawValue `asn1:"set"`
}

type signerInfo struct {
	Version            int
	SID                issuerAndSerialNumber
	DigestAlgorithm    algorithmIdentifier
	SignedAttrs        asn1.RawValue `asn1:"optional,tag:0"`
	SignatureAlgorithm algorithmIdentifier
	Signature          []byte
	UnsignedAttrs      asn1.RawValue `asn1:"optional,tag:1"`
}

// buildCMSSignedData creates a detached CMS/PKCS#7 SignedData signature for PDF signing.
func buildCMSSignedData(signer Signer, data []byte, opts Options) ([]byte, error) {
	cert := signer.Certificate()
	if cert == nil {
		return nil, fmt.Errorf("cms: signer has no certificate")
	}

	// Compute message digest.
	digest := sha256.Sum256(data)

	// Build signed attributes.
	signedAttrs, err := buildSignedAttrs(digest[:])
	if err != nil {
		return nil, fmt.Errorf("cms: build signed attrs: %w", err)
	}

	// Sign the DER-encoded signed attributes.
	sig, err := signer.Sign(signedAttrs)
	if err != nil {
		return nil, fmt.Errorf("cms: sign: %w", err)
	}

	// Encode signed attributes as implicit SET for inclusion in SignerInfo.
	signedAttrsRaw := asn1.RawValue{
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
		Bytes:      signedAttrs[2:], // strip outer SEQUENCE tag+length, use SET contents
	}
	// We need the raw SET OF bytes. Re-marshal the attributes as a SET.
	signedAttrsForSI, err := marshalSignedAttrsImplicit(digest[:])
	if err != nil {
		return nil, fmt.Errorf("cms: marshal signed attrs implicit: %w", err)
	}
	signedAttrsRaw = asn1.RawValue{
		Class:      asn1.ClassContextSpecific,
		Tag:        0,
		IsCompound: true,
		Bytes:      signedAttrsForSI,
	}

	// Build SignerInfo.
	sha256Alg := algorithmIdentifier{
		Algorithm:  oidSHA256,
		Parameters: asn1.RawValue{Tag: asn1.TagNull},
	}
	rsaAlg := algorithmIdentifier{
		Algorithm:  encryptionOIDForAlg(signer.Algorithm()),
		Parameters: asn1.RawValue{Tag: asn1.TagNull},
	}

	si := signerInfo{
		Version: 1,
		SID: issuerAndSerialNumber{
			Issuer:       asn1.RawValue{FullBytes: cert.RawIssuer},
			SerialNumber: cert.SerialNumber,
		},
		DigestAlgorithm:    sha256Alg,
		SignedAttrs:        signedAttrsRaw,
		SignatureAlgorithm: rsaAlg,
		Signature:          sig,
	}

	// Handle timestamp for PAdES B-T.
	if opts.Profile >= PAdESBT && opts.TSAURL != "" {
		tsToken, tsErr := requestTimestamp(opts.TSAURL, digest[:])
		if tsErr == nil && len(tsToken) > 0 {
			unsignedAttr, uErr := buildTimestampUnsignedAttr(tsToken)
			if uErr == nil {
				si.UnsignedAttrs = asn1.RawValue{
					Class:      asn1.ClassContextSpecific,
					Tag:        1,
					IsCompound: true,
					Bytes:      unsignedAttr,
				}
			}
		}
	}

	siBytes, err := asn1.Marshal(si)
	if err != nil {
		return nil, fmt.Errorf("cms: marshal signer info: %w", err)
	}

	// Build certificate set.
	var certBytes []byte
	for _, c := range signer.CertificateChain() {
		certBytes = append(certBytes, c.Raw...)
	}

	// Digest algorithm set.
	digestAlgBytes, err := asn1.Marshal(sha256Alg)
	if err != nil {
		return nil, fmt.Errorf("cms: marshal digest alg: %w", err)
	}

	sd := signedData{
		Version: 1,
		DigestAlgorithmIdentifiers: asn1.RawValue{
			Class:      asn1.ClassUniversal,
			Tag:        asn1.TagSet,
			IsCompound: true,
			Bytes:      digestAlgBytes,
		},
		EncapContentInfo: encapContentInfo{EContentType: oidData},
		Certificates: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      certBytes,
		},
		SignerInfos: asn1.RawValue{
			Class:      asn1.ClassUniversal,
			Tag:        asn1.TagSet,
			IsCompound: true,
			Bytes:      siBytes,
		},
	}

	sdBytes, err := asn1.Marshal(sd)
	if err != nil {
		return nil, fmt.Errorf("cms: marshal signed data: %w", err)
	}

	ci := contentInfo{
		ContentType: oidSignedData,
		Content: asn1.RawValue{
			Class:      asn1.ClassContextSpecific,
			Tag:        0,
			IsCompound: true,
			Bytes:      sdBytes,
		},
	}

	return asn1.Marshal(ci)
}

// buildSignedAttrs creates the DER-encoded signed attributes.
func buildSignedAttrs(digest []byte) ([]byte, error) {
	now := time.Now().UTC()

	contentTypeAttr, err := marshalAttribute(oidContentType, oidData)
	if err != nil {
		return nil, err
	}
	signingTimeAttr, err := marshalAttribute(oidSigningTime, now)
	if err != nil {
		return nil, err
	}
	messageDigestAttr, err := marshalAttribute(oidMessageDigest, digest)
	if err != nil {
		return nil, err
	}

	var attrs []byte
	attrs = append(attrs, contentTypeAttr...)
	attrs = append(attrs, signingTimeAttr...)
	attrs = append(attrs, messageDigestAttr...)

	// Wrap as SET OF.
	return asn1.Marshal(asn1.RawValue{
		Class:      asn1.ClassUniversal,
		Tag:        asn1.TagSet,
		IsCompound: true,
		Bytes:      attrs,
	})
}

func marshalSignedAttrsImplicit(digest []byte) ([]byte, error) {
	now := time.Now().UTC()

	contentTypeAttr, err := marshalAttribute(oidContentType, oidData)
	if err != nil {
		return nil, err
	}
	signingTimeAttr, err := marshalAttribute(oidSigningTime, now)
	if err != nil {
		return nil, err
	}
	messageDigestAttr, err := marshalAttribute(oidMessageDigest, digest)
	if err != nil {
		return nil, err
	}

	var attrs []byte
	attrs = append(attrs, contentTypeAttr...)
	attrs = append(attrs, signingTimeAttr...)
	attrs = append(attrs, messageDigestAttr...)
	return attrs, nil
}

func marshalAttribute(oid asn1.ObjectIdentifier, value interface{}) ([]byte, error) {
	valBytes, err := asn1.Marshal(value)
	if err != nil {
		return nil, err
	}
	attr := attribute{
		Type: oid,
		Values: asn1.RawValue{
			Class:      asn1.ClassUniversal,
			Tag:        asn1.TagSet,
			IsCompound: true,
			Bytes:      valBytes,
		},
	}
	return asn1.Marshal(attr)
}

func buildTimestampUnsignedAttr(tsToken []byte) ([]byte, error) {
	attr := attribute{
		Type: oidTimestampToken,
		Values: asn1.RawValue{
			Class:      asn1.ClassUniversal,
			Tag:        asn1.TagSet,
			IsCompound: true,
			Bytes:      tsToken,
		},
	}
	return asn1.Marshal(attr)
}

func encryptionOIDForAlg(alg x509.SignatureAlgorithm) asn1.ObjectIdentifier {
	switch alg {
	case x509.ECDSAWithSHA256, x509.ECDSAWithSHA384, x509.ECDSAWithSHA512:
		return asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2} // ecdsa-with-SHA256
	default:
		return oidRSAEncryption
	}
}
