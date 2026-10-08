package assistant

import (
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	voice "github.com/ygelfand/libcountertop/pkg/assistant/turn"
	"github.com/ygelfand/libcountertop/pkg/display/geometry"
	"github.com/ygelfand/libcountertop/pkg/display/gpu"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/display/visualview"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	aboveUI     = 6
	belowUI     = 1
	stageEvery  = time.Second / 60
	stageLinger = time.Second
	stageGrace  = 200 * time.Millisecond
	stageHide   = 150 * time.Millisecond
	stageRetire = 3
)

type Layers interface {
	OpenAt(w, h, depth int) (*gpu.Layer, error)
}

type Oriented interface {
	Orientation() geometry.Orientation
}

type Visuals interface {
	Hold() (release func())
	Input() visual.Input
}

type WakeSettings interface {
	Read() schema.Wake
}

type StageDependencies struct {
	Screen  *Assistant
	GPU     Layers
	Display Oriented
	Visuals Visuals
	Wake    WakeSettings
}

type look struct {
	place schema.LookPlace
	stage schema.Stage
	kind  string
}

func (l look) custom() bool { return l.kind != "" }

func stageOf(p voice.Phase) (schema.Stage, bool) {
	switch p {
	case voice.Listening:
		return schema.StageListening, true
	case voice.Thinking:
		return schema.StageWaiting, true
	case voice.Replying, voice.Idle:
		return schema.StageResponding, true
	}
	return "", false
}

type slot struct {
	layer  *gpu.Layer
	vis    visual.Visual
	kind   string
	base   int
	z      int
	placed ui.Rect
	rot    geometry.Orientation
	fresh  bool
	last   time.Time
}

func (sl *slot) close() {
	if sl != nil && sl.layer != nil {
		_ = sl.layer.Close()
		sl.layer = nil
	}
}

func (sl *slot) fits(w wanted) bool {
	return sl.layer != nil && sl.layer.W == w.at.W && sl.layer.H == w.at.H && sl.base == w.z &&
		sl.layer.Alive()
}

type wanted struct {
	at     ui.Rect
	z      int
	kind   string
	source schema.Source
}

type Stage struct {
	deps StageDependencies

	shown atomic.Bool
	full  atomic.Bool

	mu      sync.Mutex
	want    ui.Rect
	z       int
	kind    string
	source  schema.Source
	seen    time.Time
	running bool

	cur      *slot
	next     *slot
	retiring *slot
	retireIn int
	broken   string
	release  func()
	hiding   time.Time
}

func NewStage(deps StageDependencies) *Stage { return &Stage{deps: deps} }

func (s *Stage) lookFor(show Showing) look {
	st, ok := stageOf(show.Phase)
	if !ok {
		return look{}
	}
	l := s.deps.Wake.Read().Slot(show.Slot).Look
	return look{place: l.Place, stage: st, kind: l.Kind(st)}
}

func (s *Stage) Look(show Showing) (custom, full bool) {
	l := s.lookFor(show)
	return l.custom(), l.place == schema.LookFull
}

func (s *Stage) Above(at ui.Rect, show Showing) { s.stage(at, aboveUI, s.lookFor(show)) }

func (s *Stage) Below(at ui.Rect, show Showing) { s.stage(at, belowUI, s.lookFor(show)) }

func (s *Stage) Staged() bool { return s.shown.Load() }

func (s *Stage) StagedFull() bool { return s.Staged() && s.full.Load() }

func (s *Stage) stage(at ui.Rect, z int, l look) {
	if !l.custom() || at.W <= 0 || at.H <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.want, s.z, s.kind, s.source, s.seen = at, z, l.kind, l.stage.Source(), time.Now()
	if !s.running {
		s.running = true
		go s.run()
	}
}

func (s *Stage) Unstage() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		s.seen = time.Time{}
	}
}

func (s *Stage) run() {
	t := time.NewTicker(stageEvery)
	defer t.Stop()
	for range t.C {
		s.mu.Lock()
		linger := stageLinger
		if s.z != belowUI {
			linger = stageGrace
		}
		if time.Since(s.seen) <= linger {
			s.hiding = time.Time{}
		} else if s.z != belowUI {
			s.close()
			s.running = false
			s.mu.Unlock()
			return
		} else if s.hiding.IsZero() {
			s.hiding = time.Now()
			s.shown.Store(false)
			s.deps.Screen.Hide()
		} else if time.Since(s.hiding) > stageHide {
			s.close()
			s.hiding = time.Time{}
			s.running = false
			s.mu.Unlock()
			return
		}
		w := wanted{at: s.want, z: s.z, kind: s.kind, source: s.source}
		hiding := !s.hiding.IsZero()
		s.mu.Unlock()
		s.frame(w, hiding)
	}
}

func (s *Stage) frame(w wanted, hiding bool) {
	if s.release == nil {
		s.release = s.deps.Visuals.Hold()
	}
	if s.cur != nil && !s.cur.fits(w) {
		s.drop()
	}
	if s.cur == nil {
		if w.kind == s.broken {
			return
		}
		if s.cur = s.open(w, w.z, w.at); s.cur == nil {
			return
		}
		s.full.Store(w.z == belowUI)
	}

	in := visualview.From(s.deps.Visuals.Input(), w.source)
	if !s.shade(s.cur, w.at, w.at, in) {
		s.drop()
		return
	}
	if !hiding && !s.shown.Load() {
		s.shown.Store(true)
		s.deps.Screen.Paint()
	}
	if s.retiring != nil {
		if s.retireIn--; s.retireIn <= 0 {
			s.retiring.close()
			s.retiring = nil
		}
	}

	if s.cur.kind == w.kind || w.kind == s.broken {
		s.next.close()
		s.next = nil
		return
	}
	if s.next != nil && (s.next.kind != w.kind || !s.next.fits(w)) {
		s.next.close()
		s.next = nil
	}
	off := w.at
	off.X += off.W * 2
	if s.next == nil {
		z := w.z
		if s.cur.z == w.z {
			z = w.z + 1
		}
		if s.next = s.open(w, z, off); s.next == nil {
			return
		}
	}
	if !s.shade(s.next, off, w.at, in) {
		s.next.close()
		s.next = nil
		return
	}
	if !s.place(s.next, w.at) {
		s.next.close()
		s.next = nil
		return
	}
	s.retiring.close()
	s.retiring, s.retireIn = s.cur, stageRetire
	s.cur, s.next = s.next, nil
}

func (s *Stage) open(w wanted, z int, at ui.Rect) *slot {
	l, err := s.deps.GPU.OpenAt(w.at.W, w.at.H, z)
	if errors.Is(err, gpu.ErrNoHelper) {
		return nil
	}
	if err != nil {
		slog.Error("assistant: opening the GPU layer", "err", err)
		s.broken = w.kind
		return nil
	}
	sl := &slot{
		layer: l,
		vis:   visual.New(visual.Kind(w.kind)),
		kind:  w.kind,
		base:  w.z,
		z:     z,
		fresh: true,
	}
	if !s.place(sl, at) {
		sl.close()
		return nil
	}
	return sl
}

func (s *Stage) place(sl *slot, at ui.Rect) bool {
	rot := s.deps.Display.Orientation()
	if sl.placed == at && sl.rot == rot {
		return true
	}
	if err := sl.layer.Place(at); err != nil {
		slog.Error("assistant: placing the GPU layer", "err", err)
		return false
	}
	sl.placed, sl.rot = at, rot
	return true
}

func (s *Stage) shade(sl *slot, placed, area ui.Rect, in visual.Input) bool {
	if !s.place(sl, placed) {
		return false
	}
	now := time.Now()
	if !sl.last.IsZero() {
		in.Dt = now.Sub(sl.last)
	}
	sl.last = now
	if err := sl.vis.Shade(sl.layer, sl.fresh, area, in); err != nil {
		slog.Error(
			"assistant: GPU visual failed",
			"kind",
			sl.kind,
			"alive",
			sl.layer.Alive(),
			"fresh",
			sl.fresh,
			"err",
			err,
		)
		if sl.layer.Alive() {
			s.broken = sl.kind
		}
		return false
	}
	sl.fresh = false
	return true
}

func (s *Stage) drop() {
	s.shown.Store(false)
	s.cur.close()
	s.next.close()
	s.retiring.close()
	s.cur, s.next, s.retiring = nil, nil, nil
}

func (s *Stage) close() {
	s.drop()
	if s.release != nil {
		s.release()
		s.release = nil
	}
}
