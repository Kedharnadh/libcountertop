package nws

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/runtime/remember"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type weatherSection struct{ v schema.Weather }

func (s *weatherSection) Read() schema.Weather { return s.v }

func (s *weatherSection) Update(change func(*schema.Weather)) error {
	change(&s.v)
	return nil
}

func TestDismissStaysThroughUpdates(t *testing.T) {
	store, err := remember.Open(filepath.Join(t.TempDir(), "runtime.json"))
	if err != nil {
		t.Fatal(err)
	}
	n := New(Options{
		Settings:  &weatherSection{v: schema.Weather{Alerts: true}},
		Dismissed: remember.NewSlot[[]string](store, "weather.dismissed"),
	})
	first := Alert{ID: "w1", Event: "Tornado Warning", Severity: Extreme}
	n.alerts = []Alert{first}
	if got, ok := n.Current(); !ok || got.ID != "w1" {
		t.Fatalf("Current = %+v %v, want w1", got, ok)
	}
	n.Dismiss(first)
	update := Alert{ID: "w2", Refers: []string{"w1"}, Event: "Tornado Warning", Severity: Extreme}
	other := Alert{ID: "s1", Event: "Severe Thunderstorm Warning", Severity: Severe}
	n.alerts = []Alert{update, other}
	if got, ok := n.Current(); !ok || got.ID != "s1" {
		t.Fatalf("after dismissing w1, Current = %+v %v, want s1", got, ok)
	}
	n.forget(n.alerts)
	if got := n.o.Dismissed.Get(); len(got) != 1 || got[0] != "w1" {
		t.Errorf("dismissed = %v, want w1 kept while its update is live", got)
	}
	n.forget([]Alert{other})
	if got := n.o.Dismissed.Get(); len(got) != 0 {
		t.Errorf("dismissed = %v, want empty once w1 and its update are gone", got)
	}
	n.o.Settings = &weatherSection{}
	if _, ok := n.Current(); ok {
		t.Error("Current gave an alert with alerts switched off")
	}
}

func TestParse(t *testing.T) {
	body := `{"features": [
		{"properties": {"id": "a", "event": "Wind Advisory", "headline": "Wind Advisory until 6 PM",
			"areaDesc": "Kings", "severity": "Moderate", "status": "Actual",
			"expires": "2026-10-12T18:00:00-04:00"}},
		{"geometry": {"type": "Polygon", "coordinates": [[[-73.1, 41.2], [-73.0, 41.3], [-73.1, 41.2]]]},
		 "properties": {"id": "b", "event": "Tornado Warning", "severity": "Extreme", "status": "Actual",
			"expires": "2026-10-12T15:00:00-04:00", "ends": "2026-10-12T16:00:00-04:00"}},
		{"properties": {"id": "c", "event": "Test Message", "severity": "Minor", "status": "Test"}},
		{"properties": {"id": "", "event": "No id", "severity": "Severe", "status": "Actual"}}
	]}`
	got, err := parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parse gave %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("parse kept %d alerts, want 2: %+v", len(got), got)
	}
	if got[0].ID != "b" || got[0].Severity != Extreme {
		t.Errorf("first alert = %+v, want the extreme one first", got[0])
	}
	if want := time.Date(2026, 10, 12, 20, 0, 0, 0, time.UTC); !got[0].Expires.Equal(want) {
		t.Errorf("expires = %v, want ends %v", got[0].Expires, want)
	}
	if o := got[0].Outline; len(o) != 3 || o[1] != [2]float64{41.3, -73.0} {
		t.Errorf("outline = %v, want lat/lon pairs from the polygon", o)
	}
	if got[1].Outline != nil {
		t.Errorf("an alert without geometry has outline %v", got[1].Outline)
	}
	if got[1].Area != "Kings" || got[1].Severity != Moderate {
		t.Errorf("second alert = %+v", got[1])
	}
	if _, err := parse(strings.NewReader("nope")); err == nil {
		t.Error("parse of garbage gave no error")
	}
}
