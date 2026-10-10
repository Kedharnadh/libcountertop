package castreceiver

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/media/cast"
)

func bundleBody(t *testing.T, k *keys, starts ...time.Time) []byte {
	t.Helper()
	pemOf := func(kind string, der []byte) string {
		return string(pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: der}))
	}
	b64 := base64.StdEncoding.EncodeToString

	var b cast.Bundle
	for _, at := range starts {
		c, err := k.authority.Issue("Kitchen", at.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		key, err := x509.MarshalPKCS8PrivateKey(c.Peer.PrivateKey)
		if err != nil {
			t.Fatal(err)
		}
		b.Device = pemOf("CERTIFICATE", c.Device)
		b.Intermediate = pemOf("CERTIFICATE", c.Intermediates[0])
		b.Peers = append(b.Peers, cast.BundlePeer{
			Public:  pemOf("CERTIFICATE", c.Peer.Certificate[0]),
			Private: pemOf("PRIVATE KEY", key),
			SHA1:    b64(c.SHA1),
			SHA256:  b64(c.SHA256),
		})
	}
	body, _ := json.Marshal(b)
	return body
}

func counted(t *testing.T, status int, body []byte) (string, *atomic.Int32) {
	t.Helper()
	var asked atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(s.Close)
	return s.URL, &asked
}

func TestABundleIsFetchedOnceAndStepsThroughItsCertificates(t *testing.T) {
	k := testKeys(t)
	now := time.Now()
	body := bundleBody(t, k, now.Add(-47*time.Hour), now.Add(-time.Hour), now.Add(47*time.Hour))
	url, asked := counted(t, http.StatusOK, body)

	wait := k.step(t.Context(), url, "")
	c := k.Current()
	if !c.Remote || !c.NotBefore.Equal(now.Add(-time.Hour).Truncate(time.Second)) {
		t.Fatalf("the certificate used runs from %v", c.NotBefore)
	}
	if wait < 46*time.Hour || wait > 47*time.Hour {
		t.Errorf("the next look is in %v, want at notAfter", wait)
	}

	if again := k.step(t.Context(), url, ""); again < 46*time.Hour {
		t.Errorf("a current certificate was looked up again, next look in %v", again)
	}
	expired := *c
	expired.NotAfter = now
	k.held.Store(&held{creds: &expired, from: url})
	k.step(t.Context(), url, "")
	if n := asked.Load(); n != 1 {
		t.Errorf("the bundle was fetched %d times", n)
	}

	raw, err := os.ReadFile(k.bundlePath)
	if err != nil || !strings.Contains(string(raw), `"notAfter"`) {
		t.Errorf("the kept bundle has no times: %v", err)
	}
}

func TestABundleThatFailsToDownloadIsTriedOnceAndTheOracleTakesOver(t *testing.T) {
	k := testKeys(t)
	now := time.Now()
	k.step(t.Context(), serve(t, bundleBody(t, k, now.Add(-time.Hour))), "")

	broken, asked := counted(t, http.StatusNotFound, nil)
	oracle := serve(t, oracleAnswer(t, k, now.Add(-time.Hour), now.Add(47*time.Hour), now))

	k.step(t.Context(), broken, oracle)
	if h := k.held.Load(); h.from != oracle {
		t.Errorf("the credentials came from %q, want the oracle", h.from)
	}
	if _, err := os.Stat(k.bundlePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the old bundle was kept: %v", err)
	}

	k.held.Store(&held{creds: k.Current(), from: "elsewhere"})
	k.step(t.Context(), broken, oracle)
	if n := asked.Load(); n != 1 {
		t.Errorf("a bundle that failed was fetched %d times", n)
	}

	restarted, err := newKeys(
		"Kitchen",
		filepath.Join(t.TempDir(), "cast-credentials.json"),
		k.bundlePath,
		filepath.Join(t.TempDir(), "cast-authority.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	restarted.step(t.Context(), broken, "")
	if n := asked.Load(); n != 2 {
		t.Errorf("after a restart the bundle was fetched %d times in all, want 2", n)
	}
}

func TestABundleWithNothingCurrentFallsBackUntilItsNextCertificate(t *testing.T) {
	k := testKeys(t)
	now := time.Now()
	url, asked := counted(t, http.StatusOK, bundleBody(t, k, now.Add(2*time.Hour)))

	wait := k.step(t.Context(), url, "")
	if k.Current().Remote {
		t.Error("a certificate not valid yet was used")
	}
	if wait > 2*time.Hour {
		t.Errorf("the next look is in %v, want by the next certificate", wait)
	}

	k.step(t.Context(), url, "")
	if n := asked.Load(); n != 1 {
		t.Errorf("the bundle was fetched %d times", n)
	}
	if k.dry.url != url {
		t.Error("a bundle with nothing current was not set aside")
	}
}
