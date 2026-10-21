package safe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileReplacesTheWholeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peer.pem")
	if err := os.WriteFile(path, []byte("an older, longer file"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(path, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil || string(got) != "new" {
		t.Fatalf("read back %q, %v; want %q", got, err, "new")
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, %v; want 0600", info.Mode().Perm(), err)
	}
}

func TestWriteFileLeavesNoTemporaryFileBehind(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFile(filepath.Join(dir, "state.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the directory holds %d entries, want only the file", len(entries))
	}
}

func TestWriteFileIntoAMissingDirectoryFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "peer.pem")
	if err := WriteFile(path, []byte("x"), 0o600); err == nil {
		t.Error("WriteFile into a missing directory succeeded")
	}
}
