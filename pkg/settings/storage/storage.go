package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type Store[T any] struct {
	path     string
	clone    func(T) T
	writes   sync.Mutex
	mu       sync.RWMutex
	cfg      T
	readable bool
	syncDir  func(string) error
}

func Load[T any](
	path string,
	defaults T,
	clone func(T) T,
	migrations ...func([]byte, *T) error,
) (*Store[T], error) {
	if clone == nil {
		clone = func(v T) T { return v }
	}
	s := &Store[T]{
		path:     path,
		clone:    clone,
		cfg:      clone(defaults),
		readable: true,
		syncDir:  syncDirectory,
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		s.readable = false
		return s, fmt.Errorf("config: %s: %w", path, err)
	}
	if err = json.Unmarshal(data, &s.cfg); err != nil {
		s.readable = false
		s.cfg = clone(defaults)
		return s, fmt.Errorf("config: %s: %w", path, err)
	}
	for _, migrate := range migrations {
		if err := migrate(data, &s.cfg); err != nil {
			s.readable = false
			s.cfg = clone(defaults)
			return s, fmt.Errorf("config: %s: %w", path, err)
		}
	}
	return s, nil
}

func (s *Store[T]) Get() T { s.mu.RLock(); defer s.mu.RUnlock(); return s.clone(s.cfg) }

func (s *Store[T]) Runtime(change func(*T)) {
	s.writes.Lock()
	defer s.writes.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	change(&s.cfg)
	s.cfg = s.clone(s.cfg)
}

func (s *Store[T]) Update(change func(*T)) error {
	s.writes.Lock()
	defer s.writes.Unlock()
	s.mu.RLock()
	cfg, readable := s.clone(s.cfg), s.readable
	s.mu.RUnlock()
	if !readable {
		return fmt.Errorf("config: not writing over %s, which could not be read", s.path)
	}
	change(&cfg)
	cfg = s.clone(cfg)
	return s.write(cfg, func() {
		s.mu.Lock()
		s.cfg = s.clone(cfg)
		s.mu.Unlock()
	})
}

func (s *Store[T]) write(cfg T, publish func()) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(s.path)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	f, err := os.CreateTemp(dir, filepath.Base(s.path)+".*")
	if err != nil {
		return fmt.Errorf("config: %s: %w", s.path, err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, s.path); err != nil {
		return err
	}
	publish()
	if err = s.syncDir(dir); err != nil {
		return fmt.Errorf("config: syncing %s: %w", dir, err)
	}
	return nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
