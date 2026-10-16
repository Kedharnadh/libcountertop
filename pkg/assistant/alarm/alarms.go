package alarm

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/runtime/component"
	"github.com/ygelfand/libcountertop/pkg/runtime/remember"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	tick      = time.Second
	ringFor   = 15 * time.Minute
	ringEvery = 2 * time.Second
	late      = time.Minute
	gentlest  = 0.1
	farthest  = 8 * 24 * time.Hour
)

const (
	EventRinging = "esphome.alarm_ringing"
	EventSnoozed = "esphome.alarm_snoozed"
	EventStopped = "esphome.alarm_stopped"

	StoppedByHand  = "stopped"
	StoppedRangOut = "rang_out"

	Rehearsal = "rehearsal"
)

type Options struct {
	Saved   remember.Slot[[]Alarm]
	Snoozes remember.Slot[map[string]time.Time]
	Chime   func(c schema.Chime, share float64)
	Duck    func() (release func())
	RingFor func() time.Duration
	Now     func() time.Time
}

type Dawn struct {
	Alarm Alarm
	From  time.Time
	Until time.Time
}

func (d Dawn) Rising() bool { return !d.Until.IsZero() }

func (d Dawn) sooner(al Alarm, due, now time.Time) Dawn {
	if al.Light <= 0 || due.Sub(now) > al.Light || (d.Rising() && !due.Before(d.Until)) {
		return d
	}
	return Dawn{Alarm: al, From: due.Add(-al.Light), Until: due}
}

func (d Dawn) Share(now time.Time) float64 {
	span := d.Until.Sub(d.From)
	if span <= 0 {
		return 1
	}
	return min(max(now.Sub(d.From).Seconds()/span.Seconds(), 0), 1)
}

type Alarms struct {
	Changed hook.Hook[struct{}]
	Dawning hook.Hook[Dawn]
	Told    hook.Hook[component.Event]

	opts Options

	mu      sync.Mutex
	list    []Alarm
	next    map[string]time.Time
	snoozed map[string]time.Time
	ringing *ringing
	dawn    Dawn
	trial   *rehearsal
}

type ringing struct {
	alarm  Alarm
	stop   context.CancelFunc
	silent bool
}

type rehearsal struct {
	alarm   Alarm
	due     time.Time
	look    schema.WakeLook
	silent  bool
	rang    bool
	snoozed bool
}

func New(opts Options) *Alarms {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	snoozed := opts.Snoozes.Get()
	if snoozed == nil {
		snoozed = map[string]time.Time{}
	}
	return &Alarms{
		opts:    opts,
		list:    opts.Saved.Get(),
		next:    map[string]time.Time{},
		snoozed: snoozed,
	}
}

func (a *Alarms) Name() string { return "alarms" }

func (a *Alarms) Run(ctx context.Context) error {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		a.check(a.opts.Now())
		select {
		case <-ctx.Done():
			a.end()
			return nil
		case <-t.C:
		}
	}
}

func (a *Alarms) List() []Alarm {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := slices.Clone(a.list)
	slices.SortStableFunc(out, func(x, y Alarm) int {
		return cmp.Or(cmp.Compare(x.Hour, y.Hour), cmp.Compare(x.Minute, y.Minute))
	})
	return out
}

func (a *Alarms) Upcoming() (Alarm, time.Time, bool) {
	now := a.opts.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	var soon Alarm
	var at time.Time
	for _, al := range a.list {
		due := a.dueLocked(al, now)
		if due.IsZero() || (!at.IsZero() && !due.Before(at)) {
			continue
		}
		soon, at = al, due
	}
	return soon, at, !at.IsZero()
}

func (a *Alarms) Set(al Alarm) (Alarm, error) {
	if err := al.Check(); err != nil {
		return Alarm{}, err
	}
	if al.ID == "" {
		al.ID = newID()
	}

	a.mu.Lock()
	if at := slices.IndexFunc(a.list, func(x Alarm) bool { return x.ID == al.ID }); at >= 0 {
		a.list[at] = al
	} else {
		a.list = append(a.list, al)
	}
	a.next[al.ID] = al.Next(a.opts.Now())
	a.unsnoozeLocked(al.ID)
	err := a.saveLocked()
	a.mu.Unlock()

	a.check(a.opts.Now())
	a.Changed.Emit(struct{}{})
	return al, err
}

func (a *Alarms) Enable(id string, on bool) error {
	a.mu.Lock()
	at := slices.IndexFunc(a.list, func(x Alarm) bool { return x.ID == id })
	if at < 0 {
		a.mu.Unlock()
		return fmt.Errorf("alarm: no alarm %q", id)
	}
	al := a.list[at]
	a.mu.Unlock()

	al.On = on
	_, err := a.Set(al)
	return err
}

func (a *Alarms) Delete(id string) error {
	a.mu.Lock()
	before := len(a.list)
	a.list = slices.DeleteFunc(a.list, func(x Alarm) bool { return x.ID == id })
	if len(a.list) == before {
		a.mu.Unlock()
		return fmt.Errorf("alarm: no alarm %q", id)
	}
	delete(a.next, id)
	a.unsnoozeLocked(id)
	err := a.saveLocked()
	ours := a.ringing != nil && a.ringing.alarm.ID == id
	a.mu.Unlock()

	if ours {
		a.Stop()
	}
	a.check(a.opts.Now())
	a.Changed.Emit(struct{}{})
	return err
}

func (a *Alarms) Ringing() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ringing != nil
}

func (a *Alarms) Current() (Alarm, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ringing == nil {
		return Alarm{}, false
	}
	return a.ringing.alarm, true
}

func (a *Alarms) Stop() bool { return a.finish(StoppedByHand) }

func (a *Alarms) finish(why string) bool {
	al, ok := a.end()
	if !ok {
		return false
	}
	slog.Info("alarm stopped", "name", al.Name, "why", why)
	a.retire(al)
	a.tell(EventStopped, al, map[string]string{"why": why})
	if al.ID == Rehearsal {
		a.mu.Lock()
		a.trial = nil
		a.mu.Unlock()
		a.check(a.opts.Now())
	}
	return true
}

func (a *Alarms) Rehearse(light time.Duration, look schema.WakeLook, silent bool) {
	now := a.opts.Now()
	a.end()
	a.mu.Lock()
	a.trial = &rehearsal{
		alarm: Alarm{
			ID: Rehearsal, Name: "Rehearsal", On: true, Sound: schema.ChimeTimer,
			Snooze: time.Minute, Volume: VolumeStatic, Light: light,
		},
		due:    now.Add(light),
		look:   look,
		silent: silent,
	}
	a.mu.Unlock()
	a.check(now)
}

func (a *Alarms) Rehearsing() (schema.WakeLook, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.trial == nil {
		return "", false
	}
	return a.trial.look, true
}

func (a *Alarms) retire(al Alarm) {
	if !al.Once() {
		return
	}
	a.mu.Lock()
	at := slices.IndexFunc(a.list, func(x Alarm) bool { return x.ID == al.ID })
	if at < 0 || !a.list[at].On {
		a.mu.Unlock()
		return
	}
	a.list[at].On = false
	err := a.saveLocked()
	a.mu.Unlock()
	if err != nil {
		slog.Warn("saving alarms", "err", err)
	}
	a.Changed.Emit(struct{}{})
}

func (a *Alarms) Dawn() Dawn {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.dawn
}

func (a *Alarms) Snoozed(id string) (time.Time, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	at, ok := a.snoozed[id]
	return at, ok
}

func (a *Alarms) Snooze() bool {
	a.mu.Lock()
	r := a.ringing
	a.mu.Unlock()
	if r == nil || r.alarm.Snooze <= 0 {
		return false
	}

	al, ok := a.end()
	if !ok {
		return false
	}
	until := a.opts.Now().Add(al.Snooze)
	a.mu.Lock()
	if t := a.trial; al.ID == Rehearsal && t != nil {
		t.due, t.rang, t.snoozed = until, false, true
	} else {
		a.snoozed[al.ID] = until
		a.saveSnoozesLocked()
	}
	a.mu.Unlock()

	slog.Info("alarm snoozed", "name", al.Name, "until", until.Format(time.TimeOnly))
	a.tell(EventSnoozed, al, map[string]string{"until": until.Format(time.RFC3339)})
	a.Changed.Emit(struct{}{})
	return true
}

func (a *Alarms) check(now time.Time) {
	var ring []Alarm
	dawn := Dawn{}

	a.mu.Lock()
	changed := false
	for i, al := range a.list {
		due := a.dueLocked(al, now)
		if due.IsZero() {
			continue
		}
		if now.Before(due) {
			if _, snoozed := a.snoozed[al.ID]; !snoozed {
				dawn = dawn.sooner(al, due, now)
			}
			continue
		}

		a.unsnoozeLocked(al.ID)
		a.next[al.ID] = al.Next(now)
		switch {
		case now.Sub(due) < late:
			ring = append(ring, al)
		case al.Once() && al.On:
			a.list[i].On = false
			changed = true
		}
	}
	if changed {
		if err := a.saveLocked(); err != nil {
			slog.Warn("saving alarms", "err", err)
		}
	}
	if t := a.trial; t != nil {
		switch {
		case now.Before(t.due) && !t.snoozed:
			dawn = dawn.sooner(t.alarm, t.due, now)
		case !now.Before(t.due) && !t.rang:
			t.rang = true
			ring = append(ring, t.alarm)
		}
	}
	moved := dawn.Alarm.ID != a.dawn.Alarm.ID || !dawn.Until.Equal(a.dawn.Until)
	a.dawn = dawn
	a.mu.Unlock()

	for _, al := range ring {
		a.ring(al)
	}
	if moved {
		a.Dawning.Emit(dawn)
	}
	if changed && len(ring) == 0 {
		a.Changed.Emit(struct{}{})
	}
}

func (a *Alarms) dueLocked(al Alarm, now time.Time) time.Time {
	if at := a.snoozed[al.ID]; !at.IsZero() {
		return at
	}
	if !al.On {
		return time.Time{}
	}
	next := a.next[al.ID]
	if next.IsZero() || next.Sub(now) > farthest {
		next = al.Next(now.Add(-late))
		a.next[al.ID] = next
	}
	return next
}

func (a *Alarms) ring(al Alarm) {
	a.end()
	slog.Info("alarm ringing", "name", al.Name, "volume", al.Volume)
	a.tell(EventRinging, al, nil)
	if al.Volume == VolumeSilent {
		a.retire(al)
		a.Changed.Emit(struct{}{})
		return
	}

	ctx, stop := context.WithCancel(context.Background())
	a.mu.Lock()
	r := &ringing{
		alarm:  al,
		stop:   stop,
		silent: al.ID == Rehearsal && a.trial != nil && a.trial.silent,
	}
	a.ringing = r
	a.mu.Unlock()
	a.Changed.Emit(struct{}{})

	safe.Go("alarm ringing", func() { a.sound(ctx, r) })
}

func (a *Alarms) sound(ctx context.Context, r *ringing) {
	if a.opts.Duck != nil {
		defer a.opts.Duck()()
	}

	length := ringFor
	if a.opts.RingFor != nil && a.opts.RingFor() > 0 {
		length = a.opts.RingFor()
	}
	start := time.Now()
	over := time.After(length)
	for {
		if a.opts.Chime != nil && !r.silent {
			a.opts.Chime(r.alarm.Sound, r.alarm.share(time.Since(start)))
		}
		select {
		case <-ctx.Done():
			return
		case <-over:
			a.mu.Lock()
			ours := a.ringing == r
			a.mu.Unlock()
			if ours {
				a.finish(StoppedRangOut)
			}
			return
		case <-time.After(ringEvery):
		}
	}
}

func (al Alarm) share(rang time.Duration) float64 {
	if al.Volume != VolumeGentle || al.Ramp <= 0 {
		return 1
	}
	return gentlest + (1-gentlest)*min(rang.Seconds()/al.Ramp.Seconds(), 1)
}

func (a *Alarms) end() (Alarm, bool) {
	a.mu.Lock()
	r := a.ringing
	a.ringing = nil
	a.mu.Unlock()
	if r == nil {
		return Alarm{}, false
	}
	r.stop()
	a.Changed.Emit(struct{}{})
	return r.alarm, true
}

func (a *Alarms) tell(name string, al Alarm, extra map[string]string) {
	if al.ID == Rehearsal {
		return
	}
	data := map[string]string{"id": al.ID, "name": al.Name, "time": al.Clock()}
	maps.Copy(data, extra)
	a.Told.Emit(component.Event{Name: name, Data: data})
}

func (a *Alarms) saveLocked() error { return a.opts.Saved.Set(slices.Clone(a.list)) }

func (a *Alarms) unsnoozeLocked(id string) {
	if _, ok := a.snoozed[id]; !ok {
		return
	}
	delete(a.snoozed, id)
	a.saveSnoozesLocked()
}

func (a *Alarms) saveSnoozesLocked() {
	if err := a.opts.Snoozes.Set(maps.Clone(a.snoozed)); err != nil {
		slog.Warn("saving snoozes", "err", err)
	}
}

func newID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
