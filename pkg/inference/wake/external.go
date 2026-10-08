package wake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/fetch"
)

// Home Assistant serves custom_wake_words statically; an offer's hash and size are the model's.

const fetchTimeout = 30 * time.Second

func Have(dir string, o esphome.ExternalWakeWord) bool {
	data, err := os.ReadFile(modelPath(dir, o.ID))
	if err != nil {
		return false
	}
	return matches(data, o) == nil
}

func Adopt(ctx context.Context, dir string, o esphome.ExternalWakeWord) (Model, error) {
	if o.ID == "" || filepath.Base(o.ID) != o.ID || o.ID == "." || o.ID == ".." {
		return Model{}, fmt.Errorf("wake: invalid model id %q", o.ID)
	}
	if Have(dir, o) {
		return load(dir, o.ID), nil
	}

	config, err := get(ctx, o.URL)
	if err != nil {
		return Model{}, fmt.Errorf("wake: config for %s: %w", o.ID, err)
	}

	var m manifest
	if err := json.Unmarshal(config, &m); err != nil {
		return Model{}, fmt.Errorf("wake: config for %s: %w", o.ID, err)
	}
	if m.Model == "" {
		return Model{}, fmt.Errorf("wake: config for %s names no model", o.ID)
	}

	model, err := get(ctx, beside(o.URL, m.Model))
	if err != nil {
		return Model{}, fmt.Errorf("wake: model for %s: %w", o.ID, err)
	}
	if err := matches(model, o); err != nil {
		return Model{}, fmt.Errorf("wake: model for %s: %w", o.ID, err)
	}

	m.WakeWord = o.Phrase
	m.TrainedLanguages = o.TrainedLanguages
	m.Model = filepath.Base(modelPath(dir, o.ID))

	if err := write(dir, o.ID, model, m); err != nil {
		return Model{}, err
	}
	return load(dir, o.ID), nil
}

func matches(model []byte, o esphome.ExternalWakeWord) error {
	if uint32(len(model)) != o.Size {
		return fmt.Errorf("%d bytes, offered as %d", len(model), o.Size)
	}

	sum := sha256.Sum256(model)
	if got := hex.EncodeToString(sum[:]); got != o.Hash {
		return fmt.Errorf("hash %s, offered as %s", got, o.Hash)
	}
	return nil
}

func write(dir, id string, model []byte, m manifest) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	config, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	for _, f := range []struct {
		path string
		data []byte
	}{
		{modelPath(dir, id), model},
		{filepath.Join(dir, id+".json"), config},
	} {
		tmp, err := os.CreateTemp(dir, ".wake-*")
		if err != nil {
			return err
		}
		name := tmp.Name()
		defer os.Remove(name)
		if err := tmp.Chmod(0o644); err != nil {
			_ = tmp.Close()
			return err
		}
		if _, err := tmp.Write(f.data); err != nil {
			_ = tmp.Close()
			return err
		}
		if err := tmp.Close(); err != nil {
			return err
		}
		if err := os.Rename(name, f.path); err != nil {
			return err
		}
	}
	return nil
}

func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := fetch.Client(fetchTimeout).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func beside(url, name string) string {
	cut := len(url) - len(path.Base(url))
	return url[:cut] + name
}

func modelPath(dir, id string) string { return filepath.Join(dir, id+".tflite") }
