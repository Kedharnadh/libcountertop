package pair

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type File struct {
	path string

	mu   sync.RWMutex
	keys map[Addr]Key

	readable bool
}

func Open(path string) (*File, error) {
	f := &File{path: path, keys: map[Addr]Key{}, readable: true}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		f.readable = false
		return f, fmt.Errorf("pair: %s: %w", path, err)
	}

	keys, err := parseBonds(data)
	if err != nil {
		f.readable = false
		return f, fmt.Errorf("pair: %s: %w", path, err)
	}

	f.keys = keys
	return f, nil
}

func parseBonds(data []byte) (map[Addr]Key, error) {
	out := map[Addr]Key{}

	s := bufio.NewScanner(bytes.NewReader(data))
	for line := 1; s.Scan(); line++ {
		text := strings.TrimSpace(s.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}

		addr, key, ok := strings.Cut(text, " ")
		if !ok {
			return nil, fmt.Errorf("line %d is not an address and a key", line)
		}

		a, err := decodeAddr(addr)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}

		raw, err := hex.DecodeString(strings.TrimSpace(key))
		if err != nil {
			return nil, fmt.Errorf("line %d: the key: %w", line, err)
		}
		if len(raw) != len(Key{}) {
			return nil, fmt.Errorf("line %d: a key of %d bytes, want %d",
				line, len(raw), len(Key{}))
		}

		out[a] = Key(raw)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeAddr(s string) (Addr, error) {
	var a Addr

	parts := strings.Split(s, ":")
	if len(parts) != len(a) {
		return a, fmt.Errorf("%q is not an address", s)
	}

	for i, p := range parts {
		raw, err := hex.DecodeString(p)
		if err != nil || len(raw) != 1 {
			return a, fmt.Errorf("%q is not an address", s)
		}
		a[len(a)-1-i] = raw[0]
	}
	return a, nil
}

func (f *File) Key(a Addr) (Key, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	k, ok := f.keys[a]
	return k, ok
}

func (f *File) Save(a Addr, k Key) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if cur, ok := f.keys[a]; ok && cur == k {
		return nil
	}

	f.keys[a] = k
	return f.write()
}

func (f *File) Forget(a Addr) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.keys[a]; !ok {
		return nil
	}

	delete(f.keys, a)
	return f.write()
}

func (f *File) Bonded() []Addr {
	f.mu.RLock()
	defer f.mu.RUnlock()

	out := slices.Collect(maps.Keys(f.keys))
	slices.SortFunc(out, func(a, b Addr) int { return slices.Compare(a[:], b[:]) })
	return out
}

func (f *File) write() error {
	if !f.readable {
		return fmt.Errorf("pair: not writing over %s, which could not be read", f.path)
	}

	var b bytes.Buffer
	for _, a := range slices.SortedFunc(maps.Keys(f.keys), func(x, y Addr) int {
		return slices.Compare(x[:], y[:])
	}) {
		key := f.keys[a]
		fmt.Fprintf(&b, "%s %s\n", a, hex.EncodeToString(key[:]))
	}

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("pair: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(f.path)+".*")
	if err != nil {
		return fmt.Errorf("pair: %s: %w", f.path, err)
	}
	name := tmp.Name()

	if err := writeThrough(tmp, b.Bytes()); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("pair: %s: %w", name, err)
	}

	if err := os.Rename(name, f.path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("pair: %s: %w", f.path, err)
	}

	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}

func writeThrough(f *os.File, data []byte) error {
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
