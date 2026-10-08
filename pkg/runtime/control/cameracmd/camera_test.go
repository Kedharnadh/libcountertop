package cameracmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAStillIsWrittenWhereAsked(t *testing.T) {
	at := filepath.Join(t.TempDir(), "still.jpg")
	out, err := Still(func() ([]byte, error) { return []byte("jpeg"), nil }, []string{at})
	if err != nil || !strings.Contains(out, at) {
		t.Fatalf("%q, %v", out, err)
	}
	if got, _ := os.ReadFile(at); string(got) != "jpeg" {
		t.Errorf("wrote %q", got)
	}
	if _, err := Still(
		func() ([]byte, error) { return nil, errors.New("no") },
		[]string{at},
	); err == nil {
		t.Error("a failed picture was reported as taken")
	}
}
