package main

import (
	"bufio"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/oarkflow/pdf/reader"
	"github.com/oarkflow/pdf/sign"
	"golang.org/x/term"
)

func main() {
	if err := runCLI(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func writeJSON(w io.Writer, v interface{}) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func printOutlines(items []reader.OutlineInfo, indent string) {
	for _, item := range items {
		fmt.Printf("%s- %s", indent, item.Title)
		if item.Page > 0 {
			fmt.Printf(" (page %d)", item.Page)
		}
		fmt.Println()
		if len(item.Children) > 0 {
			printOutlines(item.Children, indent+"  ")
		}
	}
}

func passwordValue(password string, prompt bool) string {
	if prompt && password == "" {
		return promptPDFPassword()
	}
	return password
}

func isPasswordError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid password") || strings.Contains(msg, "encrypted pdf")
}

func promptPDFPassword() string {
	return promptPDFPasswordLabel("PDF password: ")
}

func promptPDFPasswordLabel(label string) string {
	fmt.Fprint(os.Stderr, label)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		password, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
			os.Exit(1)
		}
		return string(password)
	}

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		os.Exit(1)
	}
	return strings.TrimRight(line, "\r\n")
}

func loadLocalSigner(keyPath, certPath string) (*sign.LocalSigner, error) {
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}
	key, err := parsePrivateKeyPEM(keyBytes)
	if err != nil {
		return nil, err
	}
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read cert: %w", err)
	}
	certs, err := parseCertificatesPEM(certBytes)
	if err != nil {
		return nil, err
	}
	signer := sign.NewLocalSigner(key, certs)
	if signer == nil {
		return nil, fmt.Errorf("certificate chain is empty")
	}
	return signer, nil
}

func parsePrivateKeyPEM(data []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM private key found")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		return cryptoSignerFromKey(key)
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("unsupported private key PEM type %q", block.Type)
}

func cryptoSignerFromKey(key interface{}) (crypto.Signer, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return k, nil
	case *ecdsa.PrivateKey:
		return k, nil
	case crypto.Signer:
		return k, nil
	default:
		return nil, fmt.Errorf("unsupported private key type %T", key)
	}
}

func parseCertificatesPEM(data []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no PEM certificates found")
	}
	return certs, nil
}
