package alarm

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/runtime/component"
	"github.com/ygelfand/libcountertop/pkg/runtime/remember"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

var monday = time.Date(2026, 10, 12, 6, 0, 0, 0, time.UTC)

type rig struct {
	t      *testing.T
	store  *remember.Store
	alarms *Alarms
	now    time.Time

	mu     sync.Mutex
	chimes []float64
	events []string
	dawns  []Dawn
}

func newRig(t *testing.T) *rig {
	t.Helper()
	store, err := remember.Open(filepath.Join(t.TempDir(), "runtime.json"))
	if err != nil {
		t.Fatal(err)
	}
	r := &rig{t: t, store: store, now: monday}
	r.alarms = r.build()
	r.alarms.Told.Listen(func(e component.Event) {
		r.mu.Lock()
		r.events = append(r.events, e.Name+":"+e.Data["why"])
		r.mu.Unlock()
	})
	r.alarms.Dawning.Listen(func(d Dawn) {
		r.mu.Lock()
		r.dawns = append(r.dawns, d)
		r.mu.Unlock()
	})
	t.Cleanup(func() { r.alarms.end() })
	return r
}

func (r *rig) build() *Alarms {
	return New(Options{
		Saved:   remember.NewSlot[[]Alarm](r.store, "alarms"),
		Snoozes: remember.NewSlot[map[string]time.Time](r.store, "alarm_snoozes"),
		Chime: func(_ schema.Chime, share float64) {
			r.mu.Lock()
			r.chimes = append(r.chimes, share)
			r.mu.Unlock()
		},
		Now: func() time.Time { return r.now },
	})
}

func (r *rig) at(d time.Duration) {
	r.now = monday.Add(d)
	r.alarms.check(r.now)
}

func (r *rig) set(al Alarm) Alarm {
	r.t.Helper()
	if al.Volume == "" {
		al.Volume = VolumeStatic
	}
	saved, err := r.alarms.Set(al)
	if err != nil {
		r.t.Fatalf("Set(%+v) gave %v", al, err)
	}
	return saved
}

func (r *rig) heard() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.events)
}

func (r *rig) chimed() bool {
	for range 100 {
		r.mu.Lock()
		n := len(r.chimes)
		r.mu.Unlock()
		if n > 0 {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func TestNextSkipsDaysNotChosen(t *testing.T) {
	al := Alarm{Hour: 7, Minute: 30, Days: Weekdays{time.Wednesday, time.Friday}}
	if got, want := al.Next(monday), monday.AddDate(0, 0, 2).Add(90*time.Minute); !got.Equal(want) {
		t.Errorf("Next(Monday 06:00) = %v, want Wednesday 07:30 (%v)", got, want)
	}
	once := Alarm{Hour: 5, Minute: 0}
	if got, want := once.Next(monday), monday.AddDate(0, 0, 1).Add(-time.Hour); !got.Equal(want) {
		t.Errorf("a one-off already past today: Next = %v, want tomorrow 05:00 (%v)", got, want)
	}
}

func TestRingsAtItsTimeAndAOneOffTurnsOff(t *testing.T) {
	r := newRig(t)
	al := r.set(Alarm{Name: "wake", Hour: 7, On: true})

	r.at(59 * time.Minute)
	if r.alarms.Ringing() {
		t.Fatal("ringing a minute early")
	}
	r.at(time.Hour)
	if !r.alarms.Ringing() {
		t.Fatal("not ringing at 07:00")
	}
	if !r.chimed() {
		t.Error("no chime while ringing")
	}
	if got := r.alarms.List()[0]; got.ID != al.ID || !got.On {
		t.Errorf("one-off while ringing = %+v, want it still on", got)
	}
	if !r.alarms.Stop() || r.alarms.Ringing() {
		t.Error("Stop did not stop the ringing")
	}
	if got := r.alarms.List()[0]; got.On {
		t.Errorf("one-off after Stop = %+v, want it turned off", got)
	}
	if got := r.heard(); !slices.Equal(
		got,
		[]string{EventRinging + ":", EventStopped + ":" + StoppedByHand},
	) {
		t.Errorf("events = %v", got)
	}
}

func TestARepeatingAlarmComesBackTheNextDay(t *testing.T) {
	r := newRig(t)
	r.set(Alarm{Hour: 7, On: true, Days: Weekdays{time.Monday, time.Tuesday}})
	r.at(time.Hour)
	r.alarms.Stop()

	if _, at, ok := r.alarms.Upcoming(); !ok || !at.Equal(monday.AddDate(0, 0, 1).Add(time.Hour)) {
		t.Errorf("after Monday's ring the next is %v (%v), want Tuesday 07:00", at, ok)
	}
	if !r.alarms.List()[0].On {
		t.Error("a repeating alarm turned itself off")
	}
}

func TestSnoozeSurvivesARestart(t *testing.T) {
	r := newRig(t)
	r.set(Alarm{Hour: 7, On: true, Snooze: 9 * time.Minute})
	r.at(time.Hour)
	if !r.alarms.Snooze() {
		t.Fatal("Snooze did not quiet the alarm")
	}

	r.alarms = r.build()
	id := r.alarms.List()[0].ID
	if until, ok := r.alarms.Snoozed(id); !ok || !until.Equal(monday.Add(69*time.Minute)) {
		t.Fatalf("after a restart Snoozed = %v %v, want 07:09", until, ok)
	}
	r.at(time.Hour + 9*time.Minute)
	if !r.alarms.Ringing() {
		t.Fatal("the snooze was lost across a restart")
	}
	if _, ok := r.alarms.Snoozed(id); ok {
		t.Error("the snooze outlived its ring")
	}
	if r.build().snoozed[id] != (time.Time{}) {
		t.Error("the finished snooze is still saved")
	}
}

func TestSettingATimeJustPastDoesNotRing(t *testing.T) {
	r := newRig(t)
	r.at(time.Hour + 20*time.Second)
	r.set(Alarm{Hour: 7, On: true})
	r.at(time.Hour + 21*time.Second)
	if r.alarms.Ringing() {
		t.Error("an alarm set for 07:00 at 07:00:20 rang straight away")
	}
}

func TestSnoozeRingsAgainLater(t *testing.T) {
	r := newRig(t)
	r.set(Alarm{Hour: 7, On: true, Snooze: 9 * time.Minute})
	r.at(time.Hour)
	if !r.alarms.Snooze() || r.alarms.Ringing() {
		t.Fatal("Snooze did not quiet the alarm")
	}
	r.at(time.Hour + 8*time.Minute)
	if r.alarms.Ringing() {
		t.Fatal("rang again before the snooze was up")
	}
	if until, ok := r.alarms.Snoozed(
		r.alarms.List()[0].ID,
	); !ok ||
		!until.Equal(monday.Add(69*time.Minute)) {
		t.Errorf("Snoozed = %v %v, want 07:09", until, ok)
	}
	if !r.alarms.List()[0].On {
		t.Error("a snoozed one-off shows as off")
	}
	r.at(time.Hour + 9*time.Minute)
	if !r.alarms.Ringing() {
		t.Error("did not ring again when the snooze was up")
	}
	r.alarms.Stop()
	if r.alarms.List()[0].On {
		t.Error("a one-off stayed on after its snoozed ring was stopped")
	}
}

func TestAMissedOneOffTurnsOff(t *testing.T) {
	r := newRig(t)
	r.set(Alarm{Hour: 7, On: true})
	r.at(time.Hour + 10*time.Minute)
	if r.alarms.Ringing() {
		t.Error("rang ten minutes late")
	}
	if r.alarms.List()[0].On {
		t.Error("a one-off missed while the device was busy stayed on")
	}
}

func TestASilentOneOffTurnsOffAtOnce(t *testing.T) {
	r := newRig(t)
	r.set(Alarm{Hour: 7, On: true, Volume: VolumeSilent})
	r.at(time.Hour)
	if r.alarms.List()[0].On {
		t.Error("a silent one-off stayed on after telling Home Assistant")
	}
}

func TestSnoozeOffCannotSnooze(t *testing.T) {
	r := newRig(t)
	r.set(Alarm{Hour: 7, On: true})
	r.at(time.Hour)
	if r.alarms.Snooze() {
		t.Error("an alarm with snooze off snoozed")
	}
	if !r.alarms.Ringing() {
		t.Error("a refused snooze stopped the ringing")
	}
}

func TestSilentOnlyTellsHomeAssistant(t *testing.T) {
	r := newRig(t)
	r.set(Alarm{Hour: 7, On: true, Volume: VolumeSilent})
	r.at(time.Hour)
	if r.alarms.Ringing() {
		t.Error("a silent alarm is ringing")
	}
	if got := r.heard(); !slices.Equal(got, []string{EventRinging + ":"}) {
		t.Errorf("events = %v, want only the ringing event", got)
	}
}

func TestWakeLightRisesBeforeTheAlarm(t *testing.T) {
	r := newRig(t)
	r.set(Alarm{Hour: 7, On: true, Light: 20 * time.Minute})

	r.at(39 * time.Minute)
	if len(r.dawns) != 0 {
		t.Fatalf("dawn began 21 minutes early: %+v", r.dawns)
	}
	r.at(45 * time.Minute)
	if len(r.dawns) != 1 || !r.dawns[0].Rising() {
		t.Fatalf("dawns = %+v, want one rising", r.dawns)
	}
	if got := r.dawns[0].Share(r.now); got < 0.24 || got > 0.26 {
		t.Errorf("five minutes into a twenty minute dawn the share is %.2f, want 0.25", got)
	}
	r.at(time.Hour)
	if last := r.dawns[len(r.dawns)-1]; last.Rising() {
		t.Error("the dawn did not end when the alarm rang")
	}
}

func TestGentleVolumeRises(t *testing.T) {
	al := Alarm{Volume: VolumeGentle, Ramp: time.Minute}
	if got := al.share(0); got != gentlest {
		t.Errorf("share at the start = %v, want %v", got, gentlest)
	}
	if got := al.share(30 * time.Second); got < 0.54 || got > 0.56 {
		t.Errorf("share halfway = %v, want 0.55", got)
	}
	if got := al.share(2 * time.Minute); got != 1 {
		t.Errorf("share after the ramp = %v, want 1", got)
	}
	if got := (Alarm{Volume: VolumeStatic}).share(0); got != 1 {
		t.Errorf("static share = %v, want 1", got)
	}
}

func TestAlarmsSurviveARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime.json")
	store, err := remember.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	first := New(Options{
		Saved:   remember.NewSlot[[]Alarm](store, "alarms"),
		Snoozes: remember.NewSlot[map[string]time.Time](store, "alarm_snoozes"),
	})
	want, err := first.Set(Alarm{
		Name: "gym", Hour: 5, Minute: 45, Days: Weekdays{time.Monday, time.Thursday},
		On: true, Sound: schema.ChimeRise, Snooze: 5 * time.Minute,
		Volume: VolumeGentle, Ramp: 90 * time.Second, Light: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}

	again, err := remember.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := New(Options{
		Saved:   remember.NewSlot[[]Alarm](again, "alarms"),
		Snoozes: remember.NewSlot[map[string]time.Time](again, "alarm_snoozes"),
	}).List()
	if len(got) != 1 || !equal(got[0], want) {
		t.Errorf("after reopening: %+v, want %+v", got, want)
	}
}

func TestHomeAssistantSeesPlainFields(t *testing.T) {
	raw, err := json.Marshal(Alarm{
		ID: "a1", Hour: 6, Minute: 5, Days: Weekdays{time.Saturday, time.Sunday},
		Volume: VolumeStatic, Snooze: 9 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["time"] != "06:05" || fields["snooze_minutes"] != 9.0 {
		t.Errorf("fields = %v", fields)
	}
	if days, _ := fields["days"].([]any); len(days) != 2 || days[0] != "sun" || days[1] != "sat" {
		t.Errorf("days = %v, want [sun sat]", fields["days"])
	}
}

func TestCheckRefusesWhatCannotBe(t *testing.T) {
	for name, al := range map[string]Alarm{
		"hour 24":     {Hour: 24, Volume: VolumeStatic},
		"long snooze": {Volume: VolumeStatic, Snooze: 2 * time.Hour},
		"no volume":   {},
		"short ramp":  {Volume: VolumeGentle, Ramp: 5 * time.Second},
		"long dawn":   {Volume: VolumeStatic, Light: time.Hour},
	} {
		if err := al.Check(); err == nil {
			t.Errorf("%s: Check passed", name)
		}
	}
	if err := (Alarm{Volume: VolumeGentle, Ramp: MinRamp}).Check(); err != nil {
		t.Errorf("a valid alarm: %v", err)
	}
}

func equal(a, b Alarm) bool {
	return a.ID == b.ID && a.Name == b.Name && a.Hour == b.Hour && a.Minute == b.Minute &&
		slices.Equal(a.Days, b.Days) && a.On == b.On && a.Sound == b.Sound &&
		a.Snooze == b.Snooze && a.Volume == b.Volume && a.Ramp == b.Ramp && a.Light == b.Light
}
