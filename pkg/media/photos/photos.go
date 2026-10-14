package photos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/media/immich"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

const (
	tick       = 5 * time.Second
	retryAfter = time.Minute
	deckSize   = 250
	index      = "index.json"
)

type Status int

const (
	Unset Status = iota
	Connecting
	Ready
	BadKey
	TooOld
	Unreachable
	Unknown
	Empty
)

type State struct {
	Status  Status
	Version string
	Pending int
	Missing string
}

type Picture struct {
	ID     string
	Data   []byte
	Taken  time.Time
	Place  string
	People []string
	Year   int
}

type Options struct {
	Settings storage.Section[schema.Photos]
	Dir      string
	Keep     int
	Now      func() time.Time
	DeviceID uint32
}

type entry struct {
	ID     string   `json:"id"`
	Taken  string   `json:"taken,omitempty"`
	Place  string   `json:"place,omitempty"`
	People []string `json:"people,omitempty"`
	Year   int      `json:"year,omitempty"`
}

type Library struct {
	Changed hook.Hook[State]

	o      Options
	knobs  *Controls
	status *esphome.TextSensor

	mu       sync.Mutex
	state    State
	applied  schema.Photos
	sel      immich.Selection
	resolved bool
	deck     []entry
	ready    []entry
	next     int
	memories string
	failed   time.Time
}

func New(o Options) *Library {
	if o.Keep <= 0 {
		o.Keep = 24
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	l := &Library{o: o}
	l.knobs = &Controls{
		Table:  NewTable(),
		Device: o.DeviceID,
		Read:   o.Settings.Read,
		Save:   func(s Setting, v string) error { return setting.Store(o.Settings, s, v) },
	}
	l.status = &esphome.TextSensor{Base: esphome.Base{
		ObjectID: "photos_status", Name: "Photos status", Icon: "mdi:image-multiple",
		Category: esphome.CategoryDiagnostic, DeviceID: o.DeviceID,
	}}
	l.load()
	return l
}

func (l *Library) Name() string { return "photos" }

func (l *Library) Entities() []esphome.Entity { return append(l.knobs.Entities(), l.status) }

func (l *Library) Settings() *Controls { return l.knobs }

func (l *Library) Restore(c schema.Photos) {
	l.knobs.PublishFrom(c)
	l.status.Set(l.State().Says())
}

func (l *Library) Run(ctx context.Context) error {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		l.step(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func (l *Library) State() State {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.state
}

func (l *Library) Next() (Picture, bool) {
	l.mu.Lock()
	if len(l.ready) == 0 {
		l.mu.Unlock()
		return Picture{}, false
	}
	e := l.ready[l.next%len(l.ready)]
	l.next++
	l.mu.Unlock()

	data, err := os.ReadFile(l.file(e.ID))
	if err != nil {
		slog.Warn("reading a cached photo failed", "asset", e.ID, "err", err)
		l.forget(e.ID)
		return Picture{}, false
	}
	taken, _ := time.Parse(time.RFC3339, e.Taken)
	return Picture{
		ID:     e.ID,
		Data:   data,
		Taken:  taken,
		Place:  e.Place,
		People: e.People,
		Year:   e.Year,
	}, true
}

func (l *Library) step(ctx context.Context) {
	cfg := l.o.Settings.Read()
	if cfg.Server == "" {
		l.reset(cfg)
		l.set(State{Status: Unset})
		return
	}
	if !same(cfg, l.applied) {
		l.reset(cfg)
		l.set(State{Status: Connecting})
	}
	l.mu.Lock()
	failed := l.failed
	l.mu.Unlock()
	if !failed.IsZero() && l.o.Now().Sub(failed) < retryAfter {
		return
	}
	c := immich.New(cfg.Server, cfg.Key)
	if err := l.prepare(ctx, c, cfg); err != nil {
		l.fail(err)
		return
	}
	if err := l.refill(ctx, c, cfg); err != nil {
		l.fail(err)
		return
	}
	if err := l.prefetch(ctx, c); err != nil {
		l.fail(err)
		return
	}
	l.mu.Lock()
	st := State{Status: Ready, Version: l.state.Version, Pending: len(l.deck)}
	if len(l.ready) == 0 && len(l.deck) == 0 {
		st.Status = Empty
	}
	l.failed = time.Time{}
	l.mu.Unlock()
	l.set(st)
}

func (l *Library) prepare(ctx context.Context, c *immich.Client, cfg schema.Photos) error {
	l.mu.Lock()
	done := l.resolved
	l.mu.Unlock()
	if done {
		return nil
	}
	about, err := c.About(ctx)
	if err != nil {
		return err
	}
	if !immich.Supported(about.Version) {
		l.set(State{Status: TooOld, Version: about.Version})
		return errTooOld
	}
	sel, err := resolve(ctx, c, cfg)
	if err != nil {
		return err
	}
	l.mu.Lock()
	l.sel, l.resolved = sel, true
	l.state.Version = about.Version
	l.mu.Unlock()
	return nil
}

func (l *Library) refill(ctx context.Context, c *immich.Client, cfg schema.Photos) error {
	today := l.o.Now().Format(time.DateOnly)
	l.mu.Lock()
	sel, low, memories := l.sel, len(l.deck) < l.o.Keep, cfg.Memories && l.memories != today
	l.mu.Unlock()

	var fresh []entry
	if memories {
		found, err := c.Memories(ctx, l.o.Now())
		if err != nil {
			return err
		}
		for _, m := range found {
			for _, a := range m.Assets {
				if a.Type == "" || a.Type == "IMAGE" {
					fresh = append(fresh, describe(a, m.Data.Year))
				}
			}
		}
	}
	var drawn []entry
	if low {
		found, err := c.Random(ctx, sel, deckSize)
		if err != nil {
			return err
		}
		for _, a := range found {
			drawn = append(drawn, describe(a, 0))
		}
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if memories {
		l.memories = today
	}
	seen := map[string]bool{}
	for _, e := range l.ready {
		seen[e.ID] = true
	}
	for _, e := range l.deck {
		seen[e.ID] = true
	}
	keep := func(in []entry) []entry {
		var out []entry
		for _, e := range in {
			if !seen[e.ID] {
				seen[e.ID] = true
				out = append(out, e)
			}
		}
		return out
	}
	l.deck = append(append(keep(fresh), l.deck...), keep(drawn)...)
	return nil
}

func (l *Library) prefetch(ctx context.Context, c *immich.Client) error {
	for {
		l.mu.Lock()
		if len(l.ready)-l.next >= l.o.Keep || len(l.deck) == 0 {
			l.mu.Unlock()
			return nil
		}
		e := l.deck[0]
		l.mu.Unlock()

		data, err := c.Preview(ctx, e.ID)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(l.o.Dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(l.file(e.ID), data, 0o644); err != nil {
			return err
		}

		l.mu.Lock()
		l.deck = l.deck[1:]
		l.ready = append(l.ready, e)
		first := len(l.ready) == 1
		l.trim()
		l.mu.Unlock()
		l.save()
		if first {
			l.Changed.Emit(l.State())
		}
	}
}

func (l *Library) trim() {
	most := 2 * l.o.Keep
	if l.next > len(l.ready) {
		l.next = len(l.ready)
	}
	for len(l.ready) > most && l.next > 0 {
		_ = os.Remove(l.file(l.ready[0].ID))
		l.ready = l.ready[1:]
		l.next--
	}
}

func (l *Library) reset(cfg schema.Photos) {
	l.mu.Lock()
	had := len(l.ready) > 0 && !same(cfg, l.applied) && l.applied.Server != ""
	l.applied = cfg
	l.sel, l.resolved = immich.Selection{}, false
	l.deck, l.memories, l.failed = nil, "", time.Time{}
	if had {
		for _, e := range l.ready {
			_ = os.Remove(l.file(e.ID))
		}
		l.ready, l.next = nil, 0
	}
	l.mu.Unlock()
	if had {
		l.save()
	}
}

func (l *Library) fail(err error) {
	l.mu.Lock()
	l.failed = l.o.Now()
	version := l.state.Version
	l.mu.Unlock()
	var missing *missingError
	switch {
	case errors.Is(err, errTooOld):
		return
	case errors.Is(err, immich.ErrKey):
		l.set(State{Status: BadKey, Version: version})
	case errors.As(err, &missing):
		l.set(State{Status: Unknown, Version: version, Missing: missing.name})
	default:
		l.set(State{Status: Unreachable, Version: version})
	}
	slog.Warn("photos", "err", err)
}

func (l *Library) set(st State) {
	l.mu.Lock()
	changed := l.state != st
	l.state = st
	l.mu.Unlock()
	if changed {
		l.status.Set(st.Says())
		l.Changed.Emit(st)
	}
}

func (st State) Says() string {
	switch st.Status {
	case Connecting:
		return say.T("photos.status.connecting")
	case Ready:
		return say.F("photos.status.ready", map[string]any{"Version": st.Version})
	case BadKey:
		return say.T("photos.status.key")
	case TooOld:
		return say.F("photos.status.old", map[string]any{"Version": st.Version})
	case Unreachable:
		return say.T("photos.status.unreachable")
	case Unknown:
		return say.F("photos.status.unknown", map[string]any{"Name": st.Missing})
	case Empty:
		return say.T("photos.status.empty")
	}
	return say.T("photos.status.unset")
}

func (l *Library) forget(id string) {
	l.mu.Lock()
	if i := slices.IndexFunc(l.ready, func(e entry) bool { return e.ID == id }); i >= 0 {
		l.ready = slices.Delete(l.ready, i, i+1)
		if i < l.next {
			l.next--
		}
	}
	l.mu.Unlock()
	l.save()
}

func (l *Library) file(id string) string {
	return filepath.Join(l.o.Dir, filepath.Base(id)+".jpg")
}

func (l *Library) load() {
	data, err := os.ReadFile(filepath.Join(l.o.Dir, index))
	if err != nil {
		return
	}
	var saved []entry
	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}
	for _, e := range saved {
		if _, err := os.Stat(l.file(e.ID)); err == nil {
			l.ready = append(l.ready, e)
		}
	}
}

func (l *Library) save() {
	l.mu.Lock()
	data, err := json.Marshal(l.ready)
	l.mu.Unlock()
	if err != nil {
		return
	}
	if err := os.MkdirAll(l.o.Dir, 0o755); err != nil {
		return
	}
	tmp := filepath.Join(l.o.Dir, index+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, filepath.Join(l.o.Dir, index))
}

var errTooOld = errors.New("photos: immich is older than 3.2")

type missingError struct{ name string }

func (e *missingError) Error() string { return fmt.Sprintf("photos: nothing named %q", e.name) }

func resolve(ctx context.Context, c *immich.Client, cfg schema.Photos) (immich.Selection, error) {
	sel := immich.Selection{Favorites: cfg.Favorites}
	if names := schema.Names(cfg.Albums); len(names) > 0 {
		all, err := c.Albums(ctx)
		if err != nil {
			return sel, err
		}
		if sel.Albums, err = match(names, all, func(a immich.Album) (string, []string) {
			return a.ID, []string{a.Name}
		}); err != nil {
			return sel, err
		}
	}
	if names := schema.Names(cfg.Tags); len(names) > 0 {
		all, err := c.Tags(ctx)
		if err != nil {
			return sel, err
		}
		if sel.Tags, err = match(names, all, func(t immich.Tag) (string, []string) {
			return t.ID, []string{t.Value, t.Name}
		}); err != nil {
			return sel, err
		}
	}
	if names := schema.Names(cfg.People); len(names) > 0 {
		all, err := c.People(ctx)
		if err != nil {
			return sel, err
		}
		if sel.People, err = match(names, all, func(p immich.Person) (string, []string) {
			return p.ID, []string{p.Name}
		}); err != nil {
			return sel, err
		}
	}
	return sel, nil
}

func match[T any](wants []string, all []T, of func(T) (string, []string)) ([]string, error) {
	var ids []string
	for _, want := range wants {
		found := ""
		for _, item := range all {
			id, names := of(item)
			if id == want || slices.ContainsFunc(names, func(n string) bool {
				return strings.EqualFold(n, want)
			}) {
				found = id
				break
			}
		}
		if found == "" {
			return nil, &missingError{name: want}
		}
		ids = append(ids, found)
	}
	return ids, nil
}

func describe(a immich.Asset, year int) entry {
	e := entry{ID: a.ID, Year: year, Taken: a.Local}
	if a.Exif != nil {
		if a.Exif.Taken != "" {
			e.Taken = a.Exif.Taken
		}
		e.Place = place(a.Exif)
	}
	for _, p := range a.People {
		if p.Name != "" && !p.Hidden {
			e.People = append(e.People, p.Name)
		}
	}
	return e
}

func place(x *immich.Exif) string {
	var parts []string
	for _, s := range []string{x.City, x.Country} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", ")
}

func same(a, b schema.Photos) bool {
	return a.Server == b.Server && a.Key == b.Key && a.Albums == b.Albums &&
		a.Tags == b.Tags && a.People == b.People && a.Favorites == b.Favorites &&
		a.Memories == b.Memories
}
