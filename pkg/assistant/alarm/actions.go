package alarm

import (
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (a *Alarms) Actions() []*esphome.Action {
	return []*esphome.Action{
		{
			Name:    "alarm_list",
			Answers: true,
			Run: func(esphome.Call) (any, error) {
				return map[string]any{"alarms": a.List()}, nil
			},
		},
		{
			Name:    "alarm_set",
			Answers: true,
			Args: []esphome.Arg{
				{Name: "id", Type: esphome.ArgString},
				{Name: "name", Type: esphome.ArgString},
				{Name: "time", Type: esphome.ArgString},
				{Name: "days", Type: esphome.ArgString},
				{Name: "enabled", Type: esphome.ArgBool},
				{Name: "sound", Type: esphome.ArgString},
				{Name: "snooze_minutes", Type: esphome.ArgInt},
				{Name: "volume", Type: esphome.ArgString},
				{Name: "ramp_seconds", Type: esphome.ArgInt},
				{Name: "light_minutes", Type: esphome.ArgInt},
			},
			Run: func(c esphome.Call) (any, error) {
				al, err := fromCall(c)
				if err != nil {
					return nil, err
				}
				saved, err := a.Set(al)
				if err != nil {
					return nil, err
				}
				return map[string]any{"alarm": saved}, nil
			},
		},
		{
			Name: "alarm_enable",
			Args: []esphome.Arg{
				{Name: "id", Type: esphome.ArgString},
				{Name: "enabled", Type: esphome.ArgBool},
			},
			Run: func(c esphome.Call) (any, error) {
				return nil, a.Enable(c.String("id"), c.Bool("enabled"))
			},
		},
		{
			Name: "alarm_delete",
			Args: []esphome.Arg{{Name: "id", Type: esphome.ArgString}},
			Run: func(c esphome.Call) (any, error) {
				return nil, a.Delete(c.String("id"))
			},
		},
		{
			Name: "alarm_stop",
			Run:  func(esphome.Call) (any, error) { a.Stop(); return nil, nil },
		},
		{
			Name: "alarm_snooze",
			Run:  func(esphome.Call) (any, error) { a.Snooze(); return nil, nil },
		},
	}
}

func fromCall(c esphome.Call) (Alarm, error) {
	hour, minute, err := ParseClock(c.String("time"))
	if err != nil {
		return Alarm{}, err
	}
	days, err := ParseWeekdays(c.String("days"))
	if err != nil {
		return Alarm{}, err
	}
	volume := Volume(c.String("volume"))
	if volume == "" {
		volume = VolumeStatic
	}
	return Alarm{
		ID:     c.String("id"),
		Name:   c.String("name"),
		Hour:   hour,
		Minute: minute,
		Days:   days,
		On:     c.Bool("enabled"),
		Sound:  schema.Chime(c.String("sound")),
		Snooze: time.Duration(c.Int("snooze_minutes")) * time.Minute,
		Volume: volume,
		Ramp:   time.Duration(c.Int("ramp_seconds")) * time.Second,
		Light:  time.Duration(c.Int("light_minutes")) * time.Minute,
	}, nil
}
