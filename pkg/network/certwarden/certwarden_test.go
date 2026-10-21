package certwarden

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type memory struct{ v schema.Certs }

func (m *memory) Read() schema.Certs { return m.v }

func (m *memory) Update(f func(*schema.Certs)) error {
	f(&m.v)
	return nil
}

func issue(t *testing.T, name string, until time.Time) (keyPEM, certPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{name},
		NotBefore:    until.Add(-48 * time.Hour),
		NotAfter:     until,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: raw}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func server(t *testing.T, keyPEM, certPEM []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != chainPath+"kitchen-cert" ||
			r.Header.Get("X-API-Key") != "c-secret.k-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write(keyPEM)
		_, _ = w.Write(certPEM)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func configured(url string) *memory {
	return &memory{v: schema.Certs{
		Server:     url + "/",
		Cert:       "kitchen-cert",
		CertAPIKey: "c-secret",
		KeyAPIKey:  "k-secret",
	}}
}

func TestAPulledCertificateIsServedAndKept(t *testing.T) {
	keyPEM, certPEM := issue(t, "kitchen.home.example", time.Now().Add(60*24*time.Hour))
	srv := server(t, keyPEM, certPEM)
	path := filepath.Join(t.TempDir(), "https.pem")
	settings := configured(srv.URL)

	w := New(Options{Settings: settings, Path: path})
	if _, ok := w.Certificate(); ok {
		t.Fatal("a certificate before fetching anything")
	}
	if err := w.fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	cert, ok := w.Certificate()
	if !ok || cert.Leaf.Subject.CommonName != "kitchen.home.example" {
		t.Fatalf("serving %v %v, want the kitchen certificate", cert, ok)
	}

	again := New(Options{Settings: settings, Path: path})
	if _, ok := again.Certificate(); !ok {
		t.Error("a restart lost the kept certificate")
	}
	settings.v.Server = ""
	if _, ok := again.Certificate(); ok {
		t.Error("still serving it after Cert Warden was unset")
	}
}

func TestABareHostMeansHTTPS(t *testing.T) {
	for in, want := range map[string]string{
		"certs.example.com":          "https://certs.example.com",
		"certs.example.com:4055/":    "https://certs.example.com:4055",
		"http://10.0.0.5:4050":       "http://10.0.0.5:4050",
		" https://certs.example.com": "https://certs.example.com",
	} {
		if got := base(in); got != want {
			t.Errorf("base(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAWrongAPIKeyIsReported(t *testing.T) {
	keyPEM, certPEM := issue(t, "kitchen.home.example", time.Now().Add(time.Hour))
	settings := configured(server(t, keyPEM, certPEM).URL)
	settings.v.KeyAPIKey = "wrong"
	w := New(Options{Settings: settings})
	if err := w.fetch(context.Background()); !errors.Is(err, ErrRefused) {
		t.Fatalf("fetch with a wrong key returned %v, want ErrRefused", err)
	}
	if got, want := w.Says(), "Wrong API key"; got != want {
		t.Errorf("status %q, want %q", got, want)
	}
}

func TestAnExpiredCertificateIsRefused(t *testing.T) {
	keyPEM, certPEM := issue(t, "kitchen.home.example", time.Now().Add(-time.Hour))
	w := New(Options{Settings: configured(server(t, keyPEM, certPEM).URL)})
	if err := w.fetch(context.Background()); !errors.Is(err, ErrExpired) {
		t.Fatalf("fetch of an expired certificate returned %v, want ErrExpired", err)
	}
}
