package payload

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallWritesAnExecutableOnlyWhenChanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bin", "voice")

	if err := Install(path, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Errorf("mode %v, want 0755", info.Mode().Perm())
	}
	first := info.ModTime()

	if err := Install(path, []byte("v1")); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.Stat(path); !again.ModTime().Equal(first) {
		t.Error("an unchanged helper was rewritten")
	}

	if err := Install(path, []byte("v2")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "v2" {
		t.Errorf("helper holds %q, want v2", got)
	}
}

func TestInstallWithoutABinaryLeavesThePathAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "voice")
	if err := os.WriteFile(path, []byte("pushed by hand"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Install(path, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "pushed by hand" {
		t.Errorf("helper holds %q", got)
	}
}
