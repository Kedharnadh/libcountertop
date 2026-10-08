package weather

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/states"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

const (
	attrTemperature = "temperature"
	attrUnit        = "temperature_unit"
	attrHumidity    = "humidity"
	attrFeels       = "apparent_temperature"
	attrWind        = "wind_speed"
	attrWindUnit    = "wind_speed_unit"
	attrUV          = "uv_index"
)

var attributes = []string{
	"",
	attrTemperature,
	attrUnit,
	attrHumidity,
	attrFeels,
	attrWind,
	attrWindUnit,
	attrUV,
}

type Reading struct {
	Condition   string
	Temperature *float64
	Unit        string
	Humidity    *float64
	Feels       *float64
	Wind        *float64
	WindUnit    string
	UV          *float64
	Phase       float64
}

type Options struct {
	Controller *homeassistant.Controller
	States     *states.States
	Settings   storage.Section[schema.Weather]
	Home       storage.Section[schema.Home]
	Redraw     func()
	DeviceID   uint32
}

type Weather struct {
	opts Options

	mu     sync.Mutex
	entity string
	values map[string]*states.Value

	knobs *setting.Controls[schema.Weather]

	offered  []homeassistant.Entity
	failed   error
	fetching bool
	fetched  bool
}

const fetchWait = 10 * time.Second

func New(opts Options) *Weather {
	w := &Weather{opts: opts}
	w.knobs = &setting.Controls[schema.Weather]{
		Table:  NewTable(),
		Device: opts.DeviceID,
		Read:   opts.Settings.Read,
		Save:   w.store,
	}
	return w
}

func (w *Weather) Offered() ([]homeassistant.Entity, error, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.offered, w.failed, w.fetched
}

func (w *Weather) Fetch() {
	w.mu.Lock()
	if w.fetching {
		w.mu.Unlock()
		return
	}
	w.fetching = true
	w.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), fetchWait)
		defer cancel()
		list, err := w.opts.Controller.Entities(
			ctx,
			homeassistant.Filter{Domains: []string{"weather"}},
		)
		w.mu.Lock()
		w.offered, w.failed, w.fetching, w.fetched = list, err, false, true
		w.mu.Unlock()
		if err != nil {
			slog.Info("weather entities", "err", err)
		}
		if err == nil && len(list) > 0 && strings.TrimSpace(w.opts.Settings.Read().Entity) == "" {
			w.SetEntity(list[0].ID)
		}
		w.opts.Redraw()
	}()
}

func (w *Weather) Title(entity string) string {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, e := range w.offered {
		if e.ID == entity && e.Name != "" {
			return e.Name
		}
	}
	return entity
}

func (w *Weather) Now() (Reading, bool) {
	entity := strings.TrimSpace(w.opts.Settings.Read().Entity)
	if !w.opts.Home.Read().Enabled {
		return Reading{}, false
	}
	if entity == "" {
		if _, _, fetched := w.Offered(); !fetched &&
			w.opts.Controller.Access() == homeassistant.Allowed {
			w.Fetch()
		}
		return Reading{}, false
	}
	values := w.follow(entity)
	condition, ok := values[""].Get()
	if !ok {
		return Reading{}, false
	}
	text := func(a string) string {
		s, _ := values[a].Get()
		return strings.TrimSpace(s)
	}
	number := func(a string) *float64 {
		f, ok := values[a].Float()
		if !ok || math.IsNaN(f) {
			return nil
		}
		return &f
	}
	return Reading{
		Condition:   condition,
		Temperature: number(attrTemperature),
		Unit:        text(attrUnit),
		Humidity:    number(attrHumidity),
		Feels:       number(attrFeels),
		Wind:        number(attrWind),
		WindUnit:    text(attrWindUnit),
		UV:          number(attrUV),
	}, true
}

func (w *Weather) follow(entity string) map[string]*states.Value {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.entity == entity {
		return w.values
	}
	values := make(map[string]*states.Value, len(attributes))
	for _, a := range attributes {
		v := w.opts.States.Follow(entity, a)
		v.Changed.Listen(func(string) { w.opts.Redraw() })
		values[a] = v
	}
	w.entity, w.values = entity, values
	slog.Info("weather following", "entity", entity)
	return values
}

func (w *Weather) save(what string, err error) {
	if err != nil {
		slog.Error("saving a weather setting failed", "setting", what, "err", err)
	}
	w.knobs.Publish()
	w.opts.Redraw()
}

func (w *Weather) set() schema.WeatherWriter { return schema.NewWeatherWriter(w.opts.Settings) }

func (w *Weather) SetEntity(v string) { w.save("entity", w.set().Entity(strings.TrimSpace(v))) }

func (w *Weather) SetLook(v schema.WeatherLook) { w.save("look", w.set().Look(v)) }

func (w *Weather) SetDashboard(on bool) { w.save("dashboard", w.set().Dashboard(on)) }

func (w *Weather) SetIdle(on bool) { w.save("idle", w.set().Idle(on)) }

func (w *Weather) SetAnimate(on bool) { w.save("animate", w.set().Animate(on)) }

func (w *Weather) SetThemed(on bool) { w.save("themed", w.set().Themed(on)) }

func (w *Weather) SetPosition(v schema.Position) { w.save("position", w.set().Position(v)) }

func (w *Weather) SetAlign(v schema.Align) { w.save("align", w.set().Align(v)) }

func (w *Weather) SetSize(v schema.Size) { w.save("size", w.set().Size(v)) }
