package storage

import (
	"maps"
	"path/filepath"
	"sync"
	"testing"
)

type sectionRoot struct {
	First, Second int
	Labels        map[string]string
}

func TestSectionsKeepConcurrentUpdatesAndDetachedReads(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "state.json"), sectionRoot{Labels: map[string]string{"name": "saved"}}, func(v sectionRoot) sectionRoot { v.Labels = maps.Clone(v.Labels); return v })
	if err != nil {
		t.Fatal(err)
	}
	first := Bind(func() *Store[sectionRoot] { return s }, func(c *sectionRoot) *int { return &c.First })
	second := Bind(func() *Store[sectionRoot] { return s }, func(c *sectionRoot) *int { return &c.Second })
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() {
			if err := first.Update(func(v *int) { *v++ }); err != nil {
				t.Error(err)
			}
		})
		workers.Go(func() {
			if err := second.Update(func(v *int) { *v++ }); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	if first.Read() != 8 || second.Read() != 8 {
		t.Fatal("section writes lost another update", s.Get())
	}
	labels := Bind(func() *Store[sectionRoot] { return s }, func(c *sectionRoot) *map[string]string { return &c.Labels })
	labels.Read()["name"] = "changed"
	if labels.Read()["name"] != "saved" {
		t.Fatal("section read exposed saved mutable state")
	}
}
