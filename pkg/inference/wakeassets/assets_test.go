package wakeassets

import (
	"bytes"
	"os"
	"testing"
)

func TestStopReplacesDamagedModel(t *testing.T) {
	dir := t.TempDir()
	path, err := Stop(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("partial model"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Stop(dir); err != nil {
		t.Fatal(err)
	}
	want, _ := files.ReadFile("stop.tflite")
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("model not restored: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary model files left behind: %v, %v", entries, err)
	}
}
