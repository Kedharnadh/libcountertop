package payload

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
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
	return safe.WriteFile(path, binary, 0o755)
}
