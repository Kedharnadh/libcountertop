package alarm

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const bodyMost = 16 << 10

type sound struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type view struct {
	Alarms    []Alarm          `json:"alarms"`
	Ringing   string           `json:"ringing,omitempty"`
	Next      string           `json:"next,omitempty"`
	NextAt    *time.Time       `json:"next_at,omitempty"`
	Snoozed   map[string]int64 `json:"snoozed,omitempty"`
	Sounds    []sound          `json:"sounds"`
	Volumes   []Volume         `json:"volumes"`
	Snooze    int              `json:"snooze_most"`
	RampLeast int              `json:"ramp_least"`
	RampMost  int              `json:"ramp_most"`
	Light     int              `json:"light_most"`
}

func (a *Alarms) Handler(tones func() []setting.Option) http.Handler {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { reply(w, a.view(tones)) })
	r.Post("/", a.save(""))
	r.Put(
		"/{id}",
		func(w http.ResponseWriter, r *http.Request) { a.save(chi.URLParam(r, "id"))(w, r) },
	)
	r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
		answer(w, a.Delete(chi.URLParam(r, "id")))
	})
	r.Post("/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			On *bool `json:"enabled"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).Decode(&body); err != nil ||
			body.On == nil {
			http.Error(w, "send enabled", http.StatusBadRequest)
			return
		}
		answer(w, a.Enable(chi.URLParam(r, "id"), *body.On))
	})
	r.Post("/stop", func(w http.ResponseWriter, _ *http.Request) { a.Stop(); answer(w, nil) })
	r.Post("/snooze", func(w http.ResponseWriter, _ *http.Request) { a.Snooze(); answer(w, nil) })
	return r
}

func (a *Alarms) view(tones func() []setting.Option) view {
	v := view{
		Alarms:    a.List(),
		Volumes:   Volumes(),
		Snooze:    int(MaxSnooze / time.Minute),
		RampLeast: int(MinRamp / time.Second),
		RampMost:  int(MaxRamp / time.Second),
		Light:     int(MaxLight / time.Minute),
	}
	if cur, ok := a.Current(); ok {
		v.Ringing = cur.ID
	}
	if al, at, ok := a.Upcoming(); ok {
		v.Next, v.NextAt = al.ID, &at
	}
	for _, al := range v.Alarms {
		if until, ok := a.Snoozed(al.ID); ok {
			if v.Snoozed == nil {
				v.Snoozed = map[string]int64{}
			}
			v.Snoozed[al.ID] = until.Unix()
		}
	}
	for _, c := range schema.AlertChimes() {
		if !c.Silent() {
			v.Sounds = append(v.Sounds, sound{Value: string(c), Label: c.Label()})
		}
	}
	if tones != nil {
		for _, o := range tones() {
			v.Sounds = append(v.Sounds, sound{Value: o.Value, Label: o.Label})
		}
	}
	return v
}

func (a *Alarms) save(id string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var al Alarm
		if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).Decode(&al); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		al.ID = id
		saved, err := a.Set(al)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		reply(w, saved)
	}
}

func answer(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(v); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		slog.Debug("writing an alarm reply failed", "err", err)
	}
}
