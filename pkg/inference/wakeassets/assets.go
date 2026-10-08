package wakeassets

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

// The stop word, from kahrendt/microWakeWord.
//
//go:embed stop.tflite
var files embed.FS

// Windowing from the stop.json of the microWakeWord release.
const (
	StopPhrase      = "Stop"
	StopWindowSize  = 5
	StopFeatureStep = 10
)

func Stop(dir string) (string, error) {
	want, err := files.ReadFile("stop.tflite")
	if err != nil {
		return "", fmt.Errorf("assets: reading the embedded stop model: %w", err)
	}

	path := filepath.Join(dir, "stop.tflite")
	if same(path, want) {
		return path, nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("assets: %w", err)
	}
	f, err := os.CreateTemp(dir, ".stop-*")
	if err != nil {
		return "", fmt.Errorf("assets: creating the stop model: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return "", err
	}
	if _, err := f.Write(want); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("assets: writing the stop model: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", fmt.Errorf("assets: replacing the stop model: %w", err)
	}
	return path, nil
}

func same(path string, want []byte) bool {
	got, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return sha256.Sum256(got) == sha256.Sum256(want)
}
