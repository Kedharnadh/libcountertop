package satellite

import (
	"slices"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/assistant/turn"
)

type fakeTimers struct{ ringing, stopped bool }

func (f *fakeTimers) Event(esphome.TimerEvent) {}
func (f *fakeTimers) Forget()                  {}
func (f *fakeTimers) Ringing() bool            { return f.ringing }
func (f *fakeTimers) Stop() bool {
	if f.ringing {
		f.ringing, f.stopped = false, true
		return true
	}
	return false
}

type fakeSound struct{ busy, silenced bool }

func (f *fakeSound) Busy() bool { return f.busy }
func (f *fakeSound) Silence()   { f.busy, f.silenced = false, true }

type fakePlayer struct{ playing, paused bool }

func (f *fakePlayer) Playing() (bool, bool) { return f.playing, f.paused }
func (f *fakePlayer) Pause()                { f.playing, f.paused = false, true }

func fixture(
	initial ...string,
) (*Satellite, *esphome.VoiceSatellite, *fakeTimers, *fakeSound, *fakePlayer, map[int]string) {
	vs := &esphome.VoiceSatellite{}
	tm, snd, pl := &fakeTimers{}, &fakeSound{}, &fakePlayer{}
	saved := map[int]string{}
	s := New(vs, turn.Dependencies{}, Options{
		Slots:   2,
		Initial: initial,
		Timers:  tm,
		Sound:   snd,
		Player:  pl,
		Save:    func(slot int, id string) error { saved[slot] = id; return nil },
	})
	return s, vs, tm, snd, pl, saved
}

func TestTheSecondaryPhraseUsesItsSlot(t *testing.T) {
	s, vs, _, _, _, _ := fixture("", "B")

	if !slices.Equal(vs.ActiveWakeWords, []string{"B"}) {
		t.Fatalf("advertised %v", vs.ActiveWakeWords)
	}
	if phrase, ok := s.Turn().PhraseFor(1); !ok || phrase != "B" {
		t.Fatal(phrase, ok)
	}
	if _, ok := s.Turn().PhraseFor(0); ok {
		t.Fatal("empty primary has a phrase")
	}
}

func TestStopTakesTheTimerFirst(t *testing.T) {
	s, _, tm, snd, pl, _ := fixture()
	tm.ringing, snd.busy, pl.playing = true, true, true

	if !s.Stop() || !tm.stopped || snd.silenced || pl.paused {
		t.Errorf("timer %v sound %v player %v", tm.stopped, snd.silenced, pl.paused)
	}
	if !s.Stop() || !snd.silenced || pl.paused {
		t.Errorf("sound %v player %v", snd.silenced, pl.paused)
	}
	if !s.Stop() || !pl.paused {
		t.Error("the player kept playing")
	}
	if s.Stop() {
		t.Error("stopped with nothing going")
	}
}

type fakeAlarms struct {
	ringing, snoozable bool
	stopped, snoozed   bool
}

func (f *fakeAlarms) Ringing() bool { return f.ringing }

func (f *fakeAlarms) Stop() bool {
	if !f.ringing {
		return false
	}
	f.ringing, f.stopped = false, true
	return true
}

func (f *fakeAlarms) Snooze() bool {
	if !f.ringing || !f.snoozable {
		return false
	}
	f.ringing, f.snoozed = false, true
	return true
}

func TestAnAlarmStopsBeforeATimer(t *testing.T) {
	s, _, tm, _, _, _ := fixture()
	al := &fakeAlarms{ringing: true}
	s.options.Alarms = al
	tm.ringing = true

	if !s.Stop() || !al.stopped || tm.stopped {
		t.Errorf("alarm stopped %v, timer stopped %v", al.stopped, tm.stopped)
	}
}

func TestTappingTheActionButtonStopsAnAlarm(t *testing.T) {
	s, _, _, _, _, _ := fixture()
	al := &fakeAlarms{ringing: true, snoozable: true}
	s.options.Alarms = al
	s.Action()
	if !al.stopped || al.snoozed {
		t.Errorf("tap: stopped %v snoozed %v", al.stopped, al.snoozed)
	}
}

func TestHoldingTheActionButtonSnoozesWhenItCan(t *testing.T) {
	s, _, _, _, _, _ := fixture()
	al := &fakeAlarms{ringing: true, snoozable: true}
	s.options.Alarms = al
	s.ActionHold(1)
	if !al.snoozed || al.stopped {
		t.Errorf("snoozable alarm: snoozed %v stopped %v", al.snoozed, al.stopped)
	}

	fixed := &fakeAlarms{ringing: true}
	s.options.Alarms = fixed
	s.ActionHold(1)
	if !fixed.stopped {
		t.Error("an alarm with snooze off kept ringing after a hold")
	}
}

func TestTheStopWordStopsARingingAlarm(t *testing.T) {
	s, _, _, _, _, _ := fixture()
	al := &fakeAlarms{ringing: true, snoozable: true}
	s.options.Alarms = al
	s.Interrupt()
	if !al.stopped {
		t.Error("the stop word left the alarm ringing")
	}
}

func TestTheStopWordNeedsSomethingToStop(t *testing.T) {
	s, _, tm, snd, pl, _ := fixture()

	s.Interrupt()
	if tm.stopped || snd.silenced || pl.paused {
		t.Error("the stop word acted with nothing going")
	}

	pl.playing = true
	s.Interrupt()
	if !pl.paused {
		t.Error("the stop word left the music playing")
	}
}

func TestASelectionIsSavedSlotBySlot(t *testing.T) {
	s, vs, _, _, _, saved := fixture("A", "")
	var told bool
	s.OnWakeWord(func(ids []string) []string { return []string{"", "B"} }, func() { told = true })

	vs.OnSetActiveWakeWords([]string{"B"})

	if saved[0] != "" || saved[1] != "B" || len(saved) != 2 {
		t.Errorf("saved %v", saved)
	}
	if !slices.Equal(s.Slots(), []string{"", "B"}) ||
		!slices.Equal(vs.ActiveWakeWords, []string{"B"}) {
		t.Errorf("slots %v advertised %v", s.Slots(), vs.ActiveWakeWords)
	}
	if !told {
		t.Error("the selection callback did not run")
	}
}
