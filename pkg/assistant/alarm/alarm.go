package alarm

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Volume string

const (
	VolumeStatic Volume = "static"
	VolumeGentle Volume = "gentle"
	VolumeSilent Volume = "silent"
)

func Volumes() []Volume { return []Volume{VolumeStatic, VolumeGentle, VolumeSilent} }

const (
	MaxSnooze = 60 * time.Minute
	MinRamp   = 15 * time.Second
	MaxRamp   = 300 * time.Second
	MaxLight  = 30 * time.Minute
)

type Alarm struct {
	ID     string
	Name   string
	Hour   int
	Minute int
	Days   Weekdays
	On     bool
	Sound  schema.Chime
	Snooze time.Duration
	Volume Volume
	Ramp   time.Duration
	Light  time.Duration
}

func (a Alarm) Clock() string { return fmt.Sprintf("%02d:%02d", a.Hour, a.Minute) }

func (a Alarm) Once() bool { return len(a.Days) == 0 }

func (a Alarm) Next(after time.Time) time.Time {
	day := time.Date(
		after.Year(),
		after.Month(),
		after.Day(),
		a.Hour,
		a.Minute,
		0,
		0,
		after.Location(),
	)
	for range 8 {
		if day.After(after) && (a.Once() || slices.Contains(a.Days, day.Weekday())) {
			return day
		}
		day = day.AddDate(0, 0, 1)
	}
	return time.Time{}
}

func (a Alarm) Check() error {
	switch {
	case a.Hour < 0 || a.Hour > 23 || a.Minute < 0 || a.Minute > 59:
		return fmt.Errorf("alarm: time %s is not a time of day", a.Clock())
	case a.Snooze < 0 || a.Snooze > MaxSnooze:
		return fmt.Errorf("alarm: snooze %v is outside 0 to %v", a.Snooze, MaxSnooze)
	case !slices.Contains(Volumes(), a.Volume):
		return fmt.Errorf("alarm: volume %q is not one of %v", a.Volume, Volumes())
	case a.Volume == VolumeGentle && (a.Ramp < MinRamp || a.Ramp > MaxRamp):
		return fmt.Errorf("alarm: ramp %v is outside %v to %v", a.Ramp, MinRamp, MaxRamp)
	case a.Light < 0 || a.Light > MaxLight:
		return fmt.Errorf("alarm: wake light %v is outside 0 to %v", a.Light, MaxLight)
	}
	return nil
}

func ParseClock(s string) (hour, minute int, err error) {
	t, err := time.Parse("15:04", strings.TrimSpace(s))
	if err != nil {
		return 0, 0, fmt.Errorf("alarm: time %q is not HH:MM", s)
	}
	return t.Hour(), t.Minute(), nil
}

type Weekdays []time.Weekday

var dayNames = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

func ParseWeekdays(s string) (Weekdays, error) {
	var out Weekdays
	for _, part := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == ',' || r == ' '
	}) {
		at := slices.IndexFunc(dayNames, func(n string) bool { return strings.HasPrefix(part, n) })
		if at < 0 {
			return nil, fmt.Errorf("alarm: %q is not a day", part)
		}
		if d := time.Weekday(at); !slices.Contains(out, d) {
			out = append(out, d)
		}
	}
	slices.Sort(out)
	return out, nil
}

func (w Weekdays) String() string {
	names := make([]string, 0, len(w))
	for _, d := range slices.Sorted(slices.Values(w)) {
		names = append(names, dayNames[d])
	}
	return strings.Join(names, ",")
}

type wire struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Time   string       `json:"time"`
	Days   []string     `json:"days"`
	On     bool         `json:"enabled"`
	Sound  schema.Chime `json:"sound"`
	Snooze int          `json:"snooze_minutes"`
	Volume Volume       `json:"volume"`
	Ramp   int          `json:"ramp_seconds"`
	Light  int          `json:"light_minutes"`
}

func (a Alarm) MarshalJSON() ([]byte, error) {
	days := []string{}
	if len(a.Days) > 0 {
		days = strings.Split(a.Days.String(), ",")
	}
	return json.Marshal(wire{
		ID:     a.ID,
		Name:   a.Name,
		Time:   a.Clock(),
		Days:   days,
		On:     a.On,
		Sound:  a.Sound,
		Snooze: int(a.Snooze / time.Minute),
		Volume: a.Volume,
		Ramp:   int(a.Ramp / time.Second),
		Light:  int(a.Light / time.Minute),
	})
}

func (a *Alarm) UnmarshalJSON(raw []byte) error {
	var w wire
	if err := json.Unmarshal(raw, &w); err != nil {
		return err
	}
	hour, minute, err := ParseClock(w.Time)
	if err != nil {
		return err
	}
	days, err := ParseWeekdays(strings.Join(w.Days, ","))
	if err != nil {
		return err
	}
	*a = Alarm{
		ID:     w.ID,
		Name:   w.Name,
		Hour:   hour,
		Minute: minute,
		Days:   days,
		On:     w.On,
		Sound:  w.Sound,
		Snooze: time.Duration(w.Snooze) * time.Minute,
		Volume: w.Volume,
		Ramp:   time.Duration(w.Ramp) * time.Second,
		Light:  time.Duration(w.Light) * time.Minute,
	}
	return nil
}
