package weather

import (
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (w *Weather) sel(name string) *esphome.Select {
	return w.knobs.Entity(name).(*esphome.Select)
}

type section struct{ v schema.Weather }

func (s *section) Read() schema.Weather { return s.v }

func (s *section) Update(change func(*schema.Weather)) error {
	change(&s.v)
	return nil
}

func TestASettingIsSavedAndShownToHomeAssistant(t *testing.T) {
	saved := &section{v: schema.DefaultWeather()}
	redrawn := 0
	w := New(Options{Settings: saved, Redraw: func() { redrawn++ }})

	w.sel("look").OnCommand(schema.WeatherDetail.Label())
	w.SetEntity("  weather.home ")

	if saved.v.Look != schema.WeatherDetail {
		t.Errorf("look saved as %v, want %v", saved.v.Look, schema.WeatherDetail)
	}
	if saved.v.Entity != "weather.home" {
		t.Errorf("entity saved as %q, want it trimmed", saved.v.Entity)
	}
	if got := w.sel("look").Get(); got != schema.WeatherDetail.Label() {
		t.Errorf("Home Assistant shows %q, want %q", got, schema.WeatherDetail.Label())
	}
	if redrawn != 2 {
		t.Errorf("redrawn %d times, want 2", redrawn)
	}
}

func TestAnUnknownChoiceFromHomeAssistantChangesNothing(t *testing.T) {
	saved := &section{v: schema.DefaultWeather()}
	w := New(Options{Settings: saved, Redraw: func() {}})

	w.sel("size").OnCommand("enormous")

	if saved.v != schema.DefaultWeather() {
		t.Errorf("settings became %+v", saved.v)
	}
}
