package trust

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

const lifetime = 50 * 365 * 24 * time.Hour

func LoadIdentity(path string) (tls.Certificate, error) {
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		cert, err := tls.X509KeyPair(raw, raw)
		if err == nil {
			return cert, nil
		}
		slog.Warn("the peer certificate is unreadable, making a new one", "path", path, "err", err)
	case !errors.Is(err, fs.ErrNotExist):
		return tls.Certificate{}, err
	}
	raw, err = newIdentity()
	if err != nil {
		return tls.Certificate{}, err
	}
	cert, err := tls.X509KeyPair(raw, raw)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return cert, fmt.Errorf("trust: keeping the certificate: %w", err)
	}
	if err := safe.WriteFile(path, raw, 0o600); err != nil {
		return cert, fmt.Errorf("trust: keeping the certificate: %w", err)
	}
	return cert, nil
}

func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

func newIdentity() ([]byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "countertop"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(lifetime),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("trust: making the certificate: %w", err)
	}
	priv, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	out := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return append(out, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: priv})...), nil
}
