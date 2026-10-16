package dawnview

import (
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/assistant/alarm"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type scene struct{}

func (scene) Covers() bool { return true }

type alarms struct{ dawn alarm.Dawn }

func (a *alarms) Dawn() alarm.Dawn                    { return a.dawn }
func (a *alarms) Current() (alarm.Alarm, bool)        { return alarm.Alarm{}, false }
func (a *alarms) Rehearsing() (schema.WakeLook, bool) { return "", false }
func (a *alarms) Brighten()                           {}

type saver struct{ showing, shown bool }

func (s *saver) Suspend() func() {
	if !s.showing {
		return nil
	}
	return func() { s.shown = true }
}

func rising() alarm.Dawn {
	now := time.Now()
	return alarm.Dawn{From: now, Until: now.Add(time.Minute)}
}

func TestTheScreensaverComesBackAfterTheLight(t *testing.T) {
	for _, tc := range []struct {
		name    string
		showing bool
	}{
		{"from the screensaver", true},
		{"from the dashboard", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, s := &alarms{dawn: rising()}, &saver{showing: tc.showing}
			sh := shell.New()
			l := NewLight(sh, a, scene{}, s)
			l.Check()
			if !sh.Holding(scene{}) {
				t.Fatal("the light did not go up")
			}
			s.showing = false
			a.dawn = alarm.Dawn{}
			l.Check()
			if sh.Holding(scene{}) {
				t.Error("the light stayed up")
			}
			if s.shown != tc.showing {
				t.Errorf("screensaver shown %v, want %v", s.shown, tc.showing)
			}
		})
	}
}
