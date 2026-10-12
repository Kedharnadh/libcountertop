package remember

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Store struct {
	s *storage.Store[map[string]json.RawMessage]
}

func Open(path string) (*Store, error) {
	s, err := storage.Load(
		path,
		map[string]json.RawMessage{},
		maps.Clone[map[string]json.RawMessage],
	)
	return &Store{s: s}, err
}

type Slot[T any] struct {
	store *Store
	name  string
}

func NewSlot[T any](s *Store, name string) Slot[T] { return Slot[T]{store: s, name: name} }

func (s Slot[T]) Get() T {
	var v T
	if raw, ok := s.store.s.Get()[s.name]; ok {
		_ = json.Unmarshal(raw, &v)
	}
	return v
}

func (s Slot[T]) Set(v T) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("remember: %s: %w", s.name, err)
	}
	return s.store.s.Update(func(m *map[string]json.RawMessage) {
		if *m == nil {
			*m = map[string]json.RawMessage{}
		}
		(*m)[s.name] = raw
	})
}
