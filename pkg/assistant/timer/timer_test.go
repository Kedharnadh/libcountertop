package timer

import (
	"sync/atomic"
	"testing"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/go-esphome-device/api"
)

func TestARunningTimerCountsDownWithoutBeingTold(t *testing.T) {
	c := &timer{
		total:  time.Minute,
		left:   time.Minute,
		at:     time.Now().Add(-20 * time.Second),
		active: true,
	}

	if got := c.remaining(time.Now()); got < 39*time.Second || got > 40*time.Second {
		t.Errorf("%s left, want about 40s", got)
	}
}

func TestAPausedTimerHoldsWhereItWasStopped(t *testing.T) {
	c := &timer{total: time.Minute, left: 40 * time.Second, at: time.Now().Add(-20 * time.Second)}

	if got := c.remaining(time.Now()); got != 40*time.Second {
		t.Errorf("%s left, want 40s", got)
	}
}

func TestARunningTimerNeverGoesPastZero(t *testing.T) {
	c := &timer{
		total:  time.Minute,
		left:   time.Minute,
		at:     time.Now().Add(-2 * time.Minute),
		active: true,
	}

	if got := c.remaining(time.Now()); got != 0 {
		t.Errorf("%s left, want 0", got)
	}
}

func TestTheSoonestRunningTimerIsTheOneShown(t *testing.T) {
	now := time.Now()
	ts := New(Options{})
	ts.held = map[string]*timer{
		"long":  {name: "pasta", left: 10 * time.Minute, at: now, active: true},
		"short": {name: "eggs", left: 2 * time.Minute, at: now, active: true},
	}

	if got := ts.soonest(now); got == nil || got.name != "eggs" {
		t.Errorf("showing %v, want eggs", got)
	}
}

func TestAPausedTimerIsNotShownEvenWhenItIsTheSoonest(t *testing.T) {
	now := time.Now()
	ts := New(Options{})
	ts.held = map[string]*timer{
		"running": {name: "pasta", left: 10 * time.Minute, at: now, active: true},
		"paused":  {name: "eggs", left: 2 * time.Minute, at: now},
	}

	if got := ts.soonest(now); got == nil || got.name != "pasta" {
		t.Errorf("showing %v, want pasta", got)
	}

	ts.held = map[string]*timer{"paused": {name: "eggs", left: 2 * time.Minute, at: now}}
	if got := ts.soonest(now); got != nil {
		t.Errorf("showing %v, want nothing", got)
	}
}

func TestACancelledTimerIsForgotten(t *testing.T) {
	ts := New(Options{})
	ts.Event(started("kettle", 180))

	if len(ts.held) != 1 {
		t.Fatalf("holding %d timers, want 1", len(ts.held))
	}

	ts.Event(esphome.TimerEvent{
		Type:    api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_CANCELLED,
		TimerID: "kettle",
	})
	if len(ts.held) != 0 {
		t.Errorf("holding %d timers, want none", len(ts.held))
	}
}

func TestAFinishedTimerRingsUntilItIsStopped(t *testing.T) {
	ts := New(Options{})
	t.Cleanup(func() { ts.Stop() })

	ts.Event(started("kettle", 180))
	ts.Event(esphome.TimerEvent{
		Type:    api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_FINISHED,
		TimerID: "kettle",
		Name:    "kettle",
	})

	if !ts.Ringing() {
		t.Fatal("not ringing")
	}
	if len(ts.held) != 0 {
		t.Errorf(
			"holding %d timers, want none: a finished timer is gone at both ends",
			len(ts.held),
		)
	}
	if !ts.Stop() {
		t.Error("Stop reported nothing to stop")
	}

	for range 100 {
		if !ts.Ringing() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("still ringing after being stopped")
}

func TestWhatIsCountingDownIsNamedSoonestFirst(t *testing.T) {
	ts := New(Options{})
	ts.Event(started("pasta", 600))
	ts.Event(started("eggs", 240))

	if got := ts.names.Get(); got != "eggs, pasta" {
		t.Errorf("naming %q, want %q", got, "eggs, pasta")
	}

	ts.Event(esphome.TimerEvent{
		Type:    api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_CANCELLED,
		TimerID: "eggs",
	})
	if got := ts.names.Get(); got != "pasta" {
		t.Errorf("naming %q, want %q", got, "pasta")
	}
}

func TestAnUnnamedTimerStillHasSomethingToCallIt(t *testing.T) {
	ts := New(Options{})
	e := started("kettle", 180)
	e.Name = ""
	ts.Event(e)

	if got := ts.names.Get(); got != "Timer" {
		t.Errorf("naming %q, want %q", got, "Timer")
	}
}

func TestStopSaysWhenThereWasNothingRinging(t *testing.T) {
	ts := New(Options{})
	ts.Event(started("kettle", 180))

	if ts.Stop() {
		t.Error("Stop claimed to have silenced a timer that was still running")
	}
}

func TestForgettingLeavesARingingTimerAlone(t *testing.T) {
	ts := New(Options{})
	t.Cleanup(func() { ts.Stop() })

	ts.Event(started("pasta", 600))
	ts.Event(esphome.TimerEvent{
		Type:    api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_FINISHED,
		TimerID: "kettle",
		Name:    "kettle",
	})
	ts.Forget()

	if len(ts.held) != 0 {
		t.Errorf("holding %d timers, want none", len(ts.held))
	}
	if !ts.Ringing() {
		t.Error("stopped ringing")
	}
}

func started(id string, seconds uint32) esphome.TimerEvent {
	return esphome.TimerEvent{
		Type:         api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_STARTED,
		TimerID:      id,
		Name:         id,
		TotalSeconds: seconds,
		SecondsLeft:  seconds,
		IsActive:     true,
	}
}

type shown struct {
	left, total time.Duration
	cleared     int
}

func (s *shown) Show(left, total time.Duration) { s.left, s.total = left, total }
func (s *shown) Clear()                         { s.cleared++ }

func TestTheCountdownFollowsTheSoonestTimerAndClearsWhenNoneRun(t *testing.T) {
	var s shown
	ts := New(Options{Countdown: &s})

	ts.Event(started("pasta", 600))
	ts.Event(started("eggs", 240))
	if s.total != 240*time.Second {
		t.Errorf("counting down %s, want 4m0s", s.total)
	}

	ts.Forget()
	if s.cleared == 0 {
		t.Error("the countdown was not cleared once nothing was running")
	}
}

func TestTheAlarmIsReleasedWhenTheRingingStops(t *testing.T) {
	var released atomic.Bool
	ts := New(Options{Alarm: func() func() { return func() { released.Store(true) } }})

	ts.Event(esphome.TimerEvent{
		Type:    api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_FINISHED,
		TimerID: "kettle",
	})
	ts.Stop()

	for range 100 {
		if released.Load() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("the alarm was never released")
}

func TestASilentRingShowsWithoutTheAlarm(t *testing.T) {
	var sounded atomic.Bool
	ts := New(Options{
		Alarm: func() func() { sounded.Store(true); return func() {} },
		Chime: func() { sounded.Store(true) },
	})

	ts.RingSilently()
	if !ts.Now().Ringing {
		t.Fatal("not ringing")
	}
	time.Sleep(20 * time.Millisecond)
	if !ts.Stop() {
		t.Error("Stop reported nothing to stop")
	}
	if sounded.Load() {
		t.Error("a silent ring sounded the alarm")
	}
}
