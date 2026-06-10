package slipstream

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"strings"
)

const nextProto = "slipstream"

func TLSConfigFromPinnedCertPEM(certPEM []byte) (*tls.Config, error) {
	block, rest := pem.Decode(certPEM)
	if block == nil || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errConfig("pinned certificate must contain exactly one PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	pinnedDER := append([]byte(nil), cert.Raw...)

	return &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{nextProto},
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errConfig("server did not send a certificate")
			}
			if !bytes.Equal(rawCerts[0], pinnedDER) {
				return errConfig("server certificate does not match pinned certificate")
			}
			return nil
		},
	}, nil
}

func TLSConfigFromCertSHA256(fingerprint string) (*tls.Config, error) {
	expected, err := parseFingerprint(fingerprint)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{nextProto},
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errConfig("server did not send a certificate")
			}
			sum := sha256.Sum256(rawCerts[0])
			if !bytes.Equal(sum[:], expected) {
				return errConfig("server certificate fingerprint mismatch")
			}
			return nil
		},
	}, nil
}

func InsecureTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{nextProto},
	}
}

func TLSConfigFromKeyPairPEM(certPEM, keyPEM []byte) (*tls.Config, error) {
	if len(bytes.TrimSpace(certPEM)) == 0 {
		return nil, errConfig("server certificate PEM is required")
	}
	if len(bytes.TrimSpace(keyPEM)) == 0 {
		return nil, errConfig("server private key PEM is required")
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{nextProto},
	}, nil
}

func parseFingerprint(fingerprint string) ([]byte, error) {
	cleaned := strings.TrimSpace(fingerprint)
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	if cleaned == "" {
		return nil, errConfig("certificate fingerprint is required")
	}
	if decoded, err := hex.DecodeString(cleaned); err == nil && len(decoded) == sha256.Size {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(fingerprint); err == nil && len(decoded) == sha256.Size {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(fingerprint); err == nil && len(decoded) == sha256.Size {
		return decoded, nil
	}
	return nil, errConfig("certificate fingerprint must be SHA-256 in hex or base64")
}
