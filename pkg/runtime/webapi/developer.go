package webapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/android/logd"
)

type actionArg struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type actionInfo struct {
	Name    string      `json:"name"`
	Args    []actionArg `json:"args"`
	Answers bool        `json:"answers"`
}

var argNames = map[esphome.ArgType]string{
	esphome.ArgString: "string",
	esphome.ArgInt:    "int",
	esphome.ArgFloat:  "float",
	esphome.ArgBool:   "bool",
}

func (a *API) listActions(w http.ResponseWriter, _ *http.Request) {
	out := []actionInfo{}
	if a.o.States != nil {
		for _, act := range a.o.States.Actions() {
			info := actionInfo{Name: act.Name, Args: []actionArg{}, Answers: act.Answers}
			for _, arg := range act.Args {
				info.Args = append(info.Args, actionArg{Name: arg.Name, Type: argNames[arg.Type]})
			}
			out = append(out, info)
		}
	}
	slices.SortFunc(out, func(x, y actionInfo) int { return strings.Compare(x.Name, y.Name) })
	reply(w, out)
}

func (a *API) runAction(w http.ResponseWriter, r *http.Request) {
	if a.o.States == nil {
		http.NotFound(w, r)
		return
	}
	var body struct {
		Args map[string]any `json:"args"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).Decode(&body); err != nil {
		http.Error(w, "send args", http.StatusBadRequest)
		return
	}
	call, err := esphome.NewCall(body.Args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	answer, err := a.o.States.RunAction(chi.URLParam(r, "name"), call)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	reply(w, map[string]any{"answer": answer})
}

func (a *API) press(w http.ResponseWriter, r *http.Request) {
	if a.o.States == nil {
		http.NotFound(w, r)
		return
	}
	if err := a.o.States.Press(chi.URLParam(r, "id")); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) logs(w http.ResponseWriter, r *http.Request) {
	if a.o.LogTag == "" {
		http.NotFound(w, r)
		return
	}
	lines, err := logd.Dump(a.o.LogTag)
	if err != nil && len(lines) == 0 {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		slog.Warn("reading the log back was cut short", "lines", len(lines), "err", err)
	}
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.Level.String())
		b.WriteByte(' ')
		b.WriteString(l.Text)
		b.WriteByte('\n')
	}
	a.attach(w, "log", "txt", "text/plain; charset=utf-8", []byte(b.String()))
}

type diagnostics struct {
	Device   Device                `json:"device"`
	Taken    time.Time             `json:"taken"`
	Entities []esphome.Description `json:"entities"`
	States   []esphome.State       `json:"states"`
}

func (a *API) diagnostics(w http.ResponseWriter, _ *http.Request) {
	d := diagnostics{Device: a.o.Device(), Taken: a.o.Now()}
	if a.o.States != nil {
		own := map[string]bool{}
		for _, e := range a.knobs.Entities() {
			own[e.Object()] = true
		}
		d.Entities = slices.DeleteFunc(
			a.o.States.Descriptions(),
			func(e esphome.Description) bool { return own[e.Object] },
		)
		d.States = slices.DeleteFunc(
			a.o.States.States(),
			func(s esphome.State) bool { return own[s.Object] },
		)
	}
	body, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.attach(w, "diagnostics", "json", "application/json", body)
}

func (a *API) attach(w http.ResponseWriter, what, ext, kind string, body []byte) {
	name := fmt.Sprintf(
		"%s-%s-%s.%s",
		strings.ToLower(strings.ReplaceAll(a.o.Device().Name, " ", "-")),
		what,
		a.o.Now().Format("20060102-150405"),
		ext,
	)
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write(body); err != nil {
		slog.Debug("writing a download failed", "err", err)
	}
}
