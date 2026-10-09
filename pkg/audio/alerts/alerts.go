package alerts

import (
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

const (
	Level       = 0.6
	repeatEvery = 2 * time.Second
)

type (
	Setting = setting.Setting[schema.Alerts]
	Table   = setting.Table[schema.Alerts]
)

const Group setting.Group = "Alerts"

type Alerts struct {
	*setting.Controls[schema.Alerts]
	play func([]tone.Note)

	mu         sync.Mutex
	generation int
	timer      *time.Timer
}

func New(at storage.Section[schema.Alerts], device uint32, play func([]tone.Note)) *Alerts {
	return &Alerts{
		Controls: &setting.Controls[schema.Alerts]{
			Table:  NewTable(),
			Device: device,
			Read:   at.Read,
			Save: func(s Setting, v string) error {
				return setting.Store(at, s, v)
			},
		},
		play: play,
	}
}

func (*Alerts) Name() string { return "alerts" }

func (a *Alerts) Current() schema.Alerts { return a.Read() }

func (a *Alerts) Start(t schema.Tone) {
	current := a.Read()
	notes := tone.Wake(current.Chime(t))
	a.mu.Lock()
	a.stopLocked()
	generation := a.generation
	a.mu.Unlock()
	if len(notes) == 0 {
		return
	}
	a.play(notes)
	if t == schema.ToneAlert {
		a.again(generation, notes, time.Now().Add(current.RingFor.After()))
	}
}

func (a *Alerts) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stopLocked()
}

func (a *Alerts) Preview(c schema.Chime) {
	if notes := tone.Wake(c); len(notes) > 0 {
		a.play(notes)
	}
}

func (a *Alerts) RingTimer() {
	if notes := tone.Wake(a.Read().Timer); len(notes) > 0 {
		a.play(notes)
	}
}

func (a *Alerts) RingFor() time.Duration { return a.Read().RingFor.After() }

func (a *Alerts) again(generation int, notes []tone.Note, until time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.generation != generation || !time.Now().Add(repeatEvery).Before(until) {
		return
	}
	a.timer = time.AfterFunc(repeatEvery, func() {
		a.mu.Lock()
		live := a.generation == generation
		a.mu.Unlock()
		if !live {
			return
		}
		a.play(notes)
		a.again(generation, notes, until)
	})
}

func (a *Alerts) stopLocked() {
	a.generation++
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}
}

func NewTable() *Table {
	chime := func(name, icon string, field func(*schema.Alerts) *schema.Chime) Setting {
		return setting.Pick(
			Setting{Name: name, Group: Group, Icon: icon},
			schema.AlertChimes(),
			field,
		)
	}
	return setting.NewTable("alerts", []setting.Group{Group}, []Setting{
		chime(
			"info",
			"mdi:information-outline",
			func(a *schema.Alerts) *schema.Chime { return &a.Info },
		),
		chime(
			"success",
			"mdi:check-circle-outline",
			func(a *schema.Alerts) *schema.Chime { return &a.Success },
		),
		chime(
			"warning",
			"mdi:alert-outline",
			func(a *schema.Alerts) *schema.Chime { return &a.Warning },
		),
		chime(
			"alert",
			"mdi:alert-octagon-outline",
			func(a *schema.Alerts) *schema.Chime { return &a.Alert },
		),
		chime(
			"timer",
			"mdi:timer-music-outline",
			func(a *schema.Alerts) *schema.Chime { return &a.Timer },
		),
		setting.Pick(Setting{Name: "ring_for", Group: Group, Icon: "mdi:timer-cancel-outline"},
			schema.RingDelays(), func(a *schema.Alerts) *schema.Delay { return &a.RingFor }),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
}
