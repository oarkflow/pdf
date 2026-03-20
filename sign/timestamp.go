package sign

import (
	"bytes"
	"crypto/sha256"
	"encoding/asn1"
	"fmt"
	"io"
	"math/big"
	"net/http"
)

// TSA request/response ASN.1 structures.

type timeStampReq struct {
	Version        int
	MessageImprint messageImprint
	Nonce          *big.Int `asn1:"optional"`
	CertReq        bool     `asn1:"optional"`
}

type messageImprint struct {
	HashAlgorithm algorithmIdentifier
	HashedMessage []byte
}

type timeStampResp struct {
	Status         pkiStatusInfo
	TimeStampToken asn1.RawValue `asn1:"optional"`
}

type pkiStatusInfo struct {
	Status int
}

// requestTimestamp sends an RFC 3161 timestamp request to the given TSA URL.
func requestTimestamp(tsaURL string, hash []byte) ([]byte, error) {
	if tsaURL == "" {
		return nil, fmt.Errorf("timestamp: no TSA URL provided")
	}

	// Hash the signature for the timestamp.
	h := sha256.Sum256(hash)

	req := timeStampReq{
		Version: 1,
		MessageImprint: messageImprint{
			HashAlgorithm: algorithmIdentifier{
				Algorithm:  oidSHA256,
				Parameters: asn1.RawValue{Tag: asn1.TagNull},
			},
			HashedMessage: h[:],
		},
		CertReq: true,
	}

	reqBytes, err := asn1.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("timestamp: marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", tsaURL, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("timestamp: create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/timestamp-query")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("timestamp: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("timestamp: TSA returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("timestamp: read response: %w", err)
	}

	var tsResp timeStampResp
	_, err = asn1.Unmarshal(body, &tsResp)
	if err != nil {
		return nil, fmt.Errorf("timestamp: unmarshal response: %w", err)
	}

	if tsResp.Status.Status != 0 && tsResp.Status.Status != 1 {
		return nil, fmt.Errorf("timestamp: TSA rejected request (status %d)", tsResp.Status.Status)
	}

	return tsResp.TimeStampToken.FullBytes, nil
}
