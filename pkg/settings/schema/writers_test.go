package schema

import (
	"path/filepath"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type root struct {
	Wake   Wake   `json:"wake"`
	Camera Camera `json:"camera"`
	Home   Home   `json:"home"`
	Cast   Cast   `json:"cast"`
	Idle   Idle   `json:"idle"`
	Clock  Clock  `json:"clock"`
	Media  Media  `json:"media"`
}

func cloneRoot(r root) root {
	r.Wake = r.Wake.Clone()
	r.Camera = r.Camera.Clone()
	r.Home = r.Home.Clone()
	r.Cast = r.Cast.Clone()
	return r
}

func load(t *testing.T) (*storage.Store[root], string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	s, err := storage.Load(path, root{Cast: DefaultCast(), Media: DefaultMedia()}, cloneRoot)
	if err != nil {
		t.Fatal(err)
	}
	return s, path
}

func TestReachingAWakeSlotInventsTheOnesBeforeIt(t *testing.T) {
	s, _ := load(t)
	w := NewWakeWriter(storage.Field(s, func(r *root) *Wake { return &r.Wake }), 1)
	if err := w.ID("hey_jarvis"); err != nil {
		t.Fatal(err)
	}
	got := s.Get().Wake
	if len(got.Words) != 2 || got.Words[0] != DefaultWakeWord() || got.Words[1].ID != "hey_jarvis" {
		t.Fatalf("words %+v", got.Words)
	}
	if err := NewWakeWriter(
		storage.Field(s, func(r *root) *Wake { return &r.Wake }),
		-1,
	).ID("x"); err == nil {
		t.Fatal("a negative slot was written")
	}
}

func TestAWakeWriteLeavesWhatWasReadAlone(t *testing.T) {
	s, _ := load(t)
	w := NewWakeWriter(storage.Field(s, func(r *root) *Wake { return &r.Wake }), 0)
	if err := w.MaxThink(3); err != nil {
		t.Fatal(err)
	}
	read := s.Get()
	if err := w.MaxThink(9); err != nil {
		t.Fatal(err)
	}
	if got := read.Wake.Words[0].MaxThink; got != 3 {
		t.Fatalf("a write reached into what was read: %d", got)
	}
}

func TestCameraEmptyValueRemovesTheKey(t *testing.T) {
	s, _ := load(t)
	w := NewCameraWriter(storage.Field(s, func(r *root) *Camera { return &r.Camera }))
	if err := w.Put(map[string]string{"gamma": "3", "scene": "auto"}); err != nil {
		t.Fatal(err)
	}
	read := s.Get()
	if err := w.Set("gamma", ""); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().Camera.Settings; len(got) != 1 || got["scene"] != "auto" {
		t.Fatalf("settings %v", got)
	}
	if read.Camera.Settings["gamma"] != "3" {
		t.Fatal("a write reached into what was read")
	}
}

func TestHomeGroupIsClampedAndEmptyPicksAreDropped(t *testing.T) {
	s, _ := load(t)
	w := NewHomeWriter(storage.Field(s, func(r *root) *Home { return &r.Home }))
	if err := w.Group("a", HomeGroupMost+3); err != nil {
		t.Fatal(err)
	}
	if err := w.Group("b", -1); err != nil {
		t.Fatal(err)
	}
	if err := w.Pick("x", HomePick{Labels: []string{"l"}}); err != nil {
		t.Fatal(err)
	}
	if err := w.Pick("x", HomePick{}); err != nil {
		t.Fatal(err)
	}
	got := s.Get().Home
	if got.Group["a"] != HomeGroupMost || got.Group["b"] != 0 {
		t.Fatalf("group %v", got.Group)
	}
	if _, ok := got.Picks["x"]; ok {
		t.Fatal("an empty pick was kept")
	}
}

func TestIdleVisualSlotsAreSeparate(t *testing.T) {
	s, _ := load(t)
	w := NewIdleWriter(storage.Field(s, func(r *root) *Idle { return &r.Idle }))
	v := IdleVisual{Kind: "orb", Source: SourceMic}
	if err := w.Visual(1, v); err != nil {
		t.Fatal(err)
	}
	if got := s.Get().Idle; got.First.On() || got.Second != v {
		t.Fatalf("first %+v second %+v", got.First, got.Second)
	}
}

func TestDuckingOutsideTheRangeIsRefused(t *testing.T) {
	s, _ := load(t)
	w := NewMediaWriter(storage.Field(s, func(r *root) *Media { return &r.Media }))
	if err := w.DuckDB(1000); err == nil {
		t.Fatal("an impossible ducking level was saved")
	}
	if got := s.Get().Media; got != DefaultMedia() {
		t.Fatalf("media %+v", got)
	}
}
