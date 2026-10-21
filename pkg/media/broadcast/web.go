package broadcast

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
)

const webFrom = "Browser"

type webTarget struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model string `json:"model"`
}

func (b *Broadcaster) WebHandler() http.Handler {
	r := chi.NewRouter()
	r.Get("/", b.webTargets)
	r.Post("/send", b.webSend)
	r.Post("/record", func(w http.ResponseWriter, _ *http.Request) {
		b.say("")
		w.WriteHeader(http.StatusNoContent)
	})
	return r
}

func (b *Broadcaster) webTargets(w http.ResponseWriter, _ *http.Request) {
	out := []webTarget{}
	for _, p := range b.Targets() {
		out = append(out, webTarget{ID: p.ID, Name: p.Name, Model: p.Model})
	}
	slices.SortFunc(out, func(x, y webTarget) int { return strings.Compare(x.Name, y.Name) })
	reply(w, map[string]any{"targets": out, "rate": Rate, "longest": int(longest.Seconds())})
}

func (b *Broadcaster) webSend(w http.ResponseWriter, r *http.Request) {
	most := int(longest.Seconds()) * Rate * 2
	raw, err := io.ReadAll(io.LimitReader(r.Body, int64(most)+1))
	if err != nil || len(raw) < 2 || len(raw) > most {
		http.Error(w, "send up to 30 s of 16 kHz mono audio", http.StatusBadRequest)
		return
	}
	samples := decode(raw)
	sent, err := b.send(samples)
	if err != nil && !errors.Is(err, ErrNobody) {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	b.play(webFrom, samples)
	reply(w, map[string]int{"sent": sent})
}

func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("writing a broadcast reply failed", "err", err)
	}
}
