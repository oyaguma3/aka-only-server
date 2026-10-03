// Package certs は証明書の生成・解析・フィンガープリント計算を行う。
package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"
)

// ErrInvalid は、証明書または秘密鍵を解釈できない、有効期間外である、
// 秘密鍵が証明書と対応していない、のいずれかを表す。
var ErrInvalid = errors.New("invalid certificate")

// Fingerprint は証明書（DER）の SHA-256 を 16進小文字で返す。
func Fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}

// ParsePEM は PEM の先頭にある証明書を解析する。
func ParsePEM(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("%w: no PEM certificate found", ErrInvalid)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return cert, nil
}

// CheckValidity は証明書が現在有効期間内かを確認する。
func CheckValidity(cert *x509.Certificate) error {
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("%w: not yet valid", ErrInvalid)
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("%w: expired", ErrInvalid)
	}
	return nil
}

// SelfSignedOptions は自己署名証明書の生成条件。
type SelfSignedOptions struct {
	CommonName string
	// Hosts はサーバー証明書の SAN。IP アドレスとして解釈できるものは IP、それ以外は DNS 名として入れる。
	Hosts []string
	// Client が true ならクライアント認証用、false ならサーバー認証用の証明書を作る。
	Client   bool
	ValidFor time.Duration
}

// SelfSigned は ECDSA P-256 の鍵と自己署名証明書を生成し、PEM で返す。
func SelfSigned(opts SelfSignedOptions) (certPEM, keyPEM []byte, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, nil, fmt.Errorf("generate serial: %w", err)
	}

	notBefore := time.Now().Add(-5 * time.Minute)
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: opts.CommonName},
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(opts.ValidFor),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	if opts.Client {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		for _, h := range opts.Hosts {
			if ip := net.ParseIP(h); ip != nil {
				tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
			} else {
				tmpl.DNSNames = append(tmpl.DNSNames, h)
			}
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, fmt.Errorf("create certificate: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal key: %w", err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM, nil
}
