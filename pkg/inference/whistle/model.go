package whistle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Model struct {
	URL    string
	SHA256 string
	Size   int64
}

var Whistle = Model{
	URL:    "https://huggingface.co/Cactus-Compute/whistle/resolve/ca5287601bef25af26dcf2e1b2bdc0843a7c19e5/whistle.cact",
	SHA256: "b6e02f048568ac5d01a2042556c658061e699acbc0aa2a1439f52f3d461dffeb",
	Size:   16919407,
}

func (m Model) Path(dir string) string {
	return filepath.Join(dir, "whistle-"+m.SHA256[:16]+".cact")
}

func (m Model) Have(dir string) bool {
	info, err := os.Stat(m.Path(dir))
	return err == nil && info.Size() == m.Size
}

func (m Model) Fetch(ctx context.Context, client *http.Client, dir string) (string, error) {
	path := m.Path(dir)
	if m.Have(dir) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("whistle: model dir: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("whistle: model: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whistle: model: %s", resp.Status)
	}
	tmp, err := os.CreateTemp(dir, ".whistle-*")
	if err != nil {
		return "", fmt.Errorf("whistle: model: %w", err)
	}
	defer os.Remove(tmp.Name())
	sum := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, sum), io.LimitReader(resp.Body, m.Size+1))
	err = errors.Join(err, tmp.Close())
	if err != nil {
		return "", fmt.Errorf("whistle: model: %w", err)
	}
	if got := hex.EncodeToString(sum.Sum(nil)); n != m.Size || got != m.SHA256 {
		return "", fmt.Errorf(
			"whistle: model is %d bytes sha256 %s, want %d bytes %s",
			n,
			got,
			m.Size,
			m.SHA256,
		)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", fmt.Errorf("whistle: model: %w", err)
	}
	m.prune(dir, path)
	return path, nil
}

func (m Model) prune(dir, keep string) {
	old, _ := filepath.Glob(filepath.Join(dir, "whistle-*.cact"))
	for _, p := range old {
		if p != keep {
			_ = os.Remove(p)
		}
	}
}
