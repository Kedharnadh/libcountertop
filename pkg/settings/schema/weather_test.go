package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type weatherRoot struct {
	Weather Weather `json:"weather"`
}

const savedBeforeTheSaverHadItsOwn = `{"weather": {
	"entity": "weather.home", "look": "card", "dashboard": true, "idle": true,
	"animate": false, "themed": true, "position": "bottom", "align": "left", "size": "mini"
}}`

func loadWeather(t *testing.T, saved string) *storage.Store[weatherRoot] {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(path, []byte(saved), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := storage.Load(
		path,
		weatherRoot{Weather: DefaultWeather()},
		func(r weatherRoot) weatherRoot { return r },
	)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAnOldConfigLoadsIntoTheDashboardLayout(t *testing.T) {
	got := loadWeather(t, savedBeforeTheSaverHadItsOwn).Get().Weather
	want := WeatherLayout{
		Look:      WeatherCard,
		Themed:    true,
		Placement: Placement{Position: PositionBottom, Align: AlignLeft, Size: SizeMini},
	}
	if got.WeatherLayout != want {
		t.Fatalf("dashboard %+v, want %+v", got.WeatherLayout, want)
	}
	if got.SaverLayout() != want {
		t.Fatalf("screen saver %+v, want the dashboard's %+v", got.SaverLayout(), want)
	}
}

func TestTheFirstSaverWriteCopiesTheDashboardThenChangesOneField(t *testing.T) {
	s := loadWeather(t, savedBeforeTheSaverHadItsOwn)
	w := NewWeatherWriter(storage.Field(s, func(r *weatherRoot) *Weather { return &r.Weather }))
	dashboard := s.Get().Weather.WeatherLayout

	if err := w.Saver(func(l *WeatherLayout) { l.Size = SizeLarge }); err != nil {
		t.Fatal(err)
	}

	got := s.Get().Weather
	want := dashboard
	want.Size = SizeLarge
	if got.Saver != want || got.SaverLayout() != want {
		t.Fatalf("screen saver %+v, want %+v", got.Saver, want)
	}
	if got.WeatherLayout != dashboard {
		t.Fatalf("dashboard became %+v, want %+v", got.WeatherLayout, dashboard)
	}
}

func TestTheLayoutsKeepTheirKeys(t *testing.T) {
	b, err := json.Marshal(DefaultWeather())
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(b, &keys); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{
		"entity", "look", "dashboard", "idle", "animate", "themed",
		"position", "align", "size", "saver",
	} {
		if _, ok := keys[k]; !ok {
			t.Errorf("%s is missing from %s", k, b)
		}
	}
}
