package payload

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
)

func Install(path string, binary []byte) error {
	if len(binary) == 0 {
		return nil
	}
	if have, err := os.ReadFile(path); err == nil && bytes.Equal(have, binary) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".voice-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(binary)
	err = errors.Join(err, tmp.Chmod(0o755), tmp.Close())
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
