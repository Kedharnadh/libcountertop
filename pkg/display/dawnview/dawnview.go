package dawnview

import (
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/assistant/alarm"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Alarms interface {
	Dawn() alarm.Dawn
	Current() (alarm.Alarm, bool)
	Rehearsing() (schema.WakeLook, bool)
	Brighten()
}

type Options struct {
	Alarms  Alarms
	Look    func() schema.WakeLook
	Clock24 func() bool
}

type Source struct {
	o     Options
	began time.Time
}

func NewSource(o Options) *Source { return &Source{o: o, began: time.Now()} }

func Kind(look schema.WakeLook) visual.Kind {
	switch look {
	case schema.WakeWhite:
		return visual.WakeWhite
	case schema.WakeEmber:
		return visual.WakeEmber
	}
	return visual.WakeSunrise
}

func (s *Source) Kind() visual.Kind {
	if look, ok := s.o.Alarms.Rehearsing(); ok && look != "" {
		return Kind(look)
	}
	return Kind(s.o.Look())
}

func (s *Source) Input() visual.Input {
	now := time.Now()
	share := 1.0
	if d := s.o.Alarms.Dawn(); d.Rising() {
		share = d.Share(now)
	}
	return visual.Input{Now: now.Sub(s.began), Label: s.clock(now), Dawn: share}
}

func (s *Source) Hold() func() { return func() {} }

func (s *Source) Loudest() float64 { return 0 }

func (s *Source) Wakes() bool { return true }

func (s *Source) Tap() bool {
	s.o.Alarms.Brighten()
	return true
}

func (s *Source) clock(now time.Time) string {
	if s.o.Clock24 != nil && s.o.Clock24() {
		return now.Format("15:04")
	}
	return now.Format("3:04")
}

func Showing(a Alarms) bool {
	if a.Dawn().Rising() {
		return true
	}
	al, ok := a.Current()
	return ok && al.Light > 0
}

type Saver interface {
	Suspend() (resume func())
}

type Light struct {
	shell  *shell.Shell
	alarms Alarms
	view   shell.View
	saver  Saver

	mu     sync.Mutex
	hold   *shell.Hold
	resume func()
}

func NewLight(sh *shell.Shell, alarms Alarms, view shell.View, saver Saver) *Light {
	return &Light{shell: sh, alarms: alarms, view: view, saver: saver}
}

func (l *Light) Check() {
	want := Showing(l.alarms)
	l.mu.Lock()
	defer l.mu.Unlock()
	switch {
	case want && (l.hold == nil || !l.shell.Holding(l.view)):
		if l.hold == nil {
			l.resume = l.saver.Suspend()
		}
		l.hold = l.shell.Hold(l.view)
	case !want && l.hold != nil:
		l.hold.Release()
		l.hold = nil
		if resume := l.resume; resume != nil {
			l.resume = nil
			resume()
		}
	}
}
