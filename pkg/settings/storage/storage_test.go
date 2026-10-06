package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
)

type config struct {
	Enabled bool     `json:"enabled"`
	Words   []string `json:"words"`
	Count   int      `json:"count"`
}

func copyConfig(c config) config { c.Words = slices.Clone(c.Words); return c }
func TestSnapshotsAndCallerValuesAreDetached(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "state"), config{}, copyConfig)
	if err != nil {
		t.Fatal(err)
	}
	words := []string{"one"}
	if err = s.Update(func(c *config) { c.Words = words }); err != nil {
		t.Fatal(err)
	}
	words[0] = "changed"
	snapshot := s.Get()
	snapshot.Words[0] = "changed again"
	if s.Get().Words[0] != "one" {
		t.Fatal("store shares mutable caller state")
	}
}
func TestTypeErrorsReturnCompleteDefaultsAndPreserveOriginalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	raw := []byte(`{"enabled":true,"count":"invalid"}`)
	os.WriteFile(path, raw, 0600)
	s, err := Load(path, config{Count: 7}, copyConfig)
	if err == nil {
		t.Fatal("want load error")
	}
	if s.Get().Enabled || s.Get().Count != 7 {
		t.Fatal(s.Get())
	}
	if s.Update(func(c *config) { c.Count = 2 }) == nil {
		t.Fatal("overwrote unreadable state")
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(raw) {
		t.Fatal(string(got))
	}
}
func TestDirectoryDurabilityFailuresAreReported(t *testing.T) {
	s, _ := Load(filepath.Join(t.TempDir(), "state"), config{}, copyConfig)
	failure := errors.New("sync failed")
	s.syncDir = func(string) error { return failure }
	if err := s.Update(func(c *config) { c.Count = 1 }); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}
func TestConcurrentUpdatesProduceTheLatestReadableState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	s, _ := Load(path, config{}, copyConfig)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Update(func(c *config) { c.Count++ }); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var disk config
	raw, _ := os.ReadFile(path)
	if err := json.Unmarshal(raw, &disk); err != nil {
		t.Fatal(err)
	}
	if disk.Count != 20 || s.Get().Count != 20 {
		t.Fatal(disk, s.Get())
	}
}

func TestFailedWriteDoesNotPublishTheChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state")
	s, _ := Load(path, config{Count: 7}, copyConfig)
	// A directory at the destination prevents replacing it with a settings file.
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(c *config) { c.Count = 9 }); err == nil {
		t.Fatal("want write failure")
	}
	if s.Get().Count != 7 {
		t.Fatal("published a failed update", s.Get())
	}
}
