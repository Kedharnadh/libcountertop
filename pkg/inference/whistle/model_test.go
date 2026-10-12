package whistle

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func served(t *testing.T, body []byte) (Model, *int) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	sum := sha256.Sum256(body)
	return Model{URL: srv.URL, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body))}, &hits
}

func TestFetchVerifiesAndKeepsTheModel(t *testing.T) {
	m, hits := served(t, []byte("whistle weights"))
	dir := t.TempDir()

	path, err := m.Fetch(t.Context(), http.DefaultClient, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "whistle weights" {
		t.Errorf("model file holds %q", got)
	}
	if _, err := m.Fetch(t.Context(), http.DefaultClient, dir); err != nil || *hits != 1 {
		t.Errorf("second fetch: err %v, %d downloads, want none", err, *hits)
	}
}

func TestFetchRejectsTheWrongBytes(t *testing.T) {
	m, _ := served(t, []byte("tampered"))
	m.SHA256 = strings.Repeat("0", 64)
	dir := t.TempDir()

	if _, err := m.Fetch(t.Context(), http.DefaultClient, dir); err == nil {
		t.Fatal("a model with the wrong hash was accepted")
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "*")); len(left) != 0 {
		t.Errorf("files left behind: %v", left)
	}
}

func TestFetchDropsOlderModels(t *testing.T) {
	m, _ := served(t, []byte("new weights"))
	dir := t.TempDir()
	old := filepath.Join(dir, "whistle-0000000000000000.cact")
	if err := os.WriteFile(old, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := m.Fetch(t.Context(), http.DefaultClient, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("old model still there: %v", err)
	}
}
