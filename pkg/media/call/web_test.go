package call

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestTheWebLineHandsSignalsToTheWaitingBrowserAndForgetsOnEnd(t *testing.T) {
	h := &webCalls{lines: map[string]webLine{}}
	l := webLine{signals: make(chan webSignal, 4), ended: func() { h.forget("s1") }}
	h.lines["s1"] = l

	wait := func() webSignal {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/signal/s1", nil)
		rc := chi.NewRouteContext()
		rc.URLParams.Add("session", "s1")
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
		w := httptest.NewRecorder()
		h.signal(w, r)
		var s webSignal
		if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	if status, _ := l.send(
		context.Background(),
		"answer",
		envelope{SDP: "v=0"},
	); status != http.StatusOK {
		t.Fatalf("answer went out as %d", status)
	}
	if s := wait(); s.Type != "answer" || s.SDP != "v=0" {
		t.Errorf("browser heard %+v, want the answer", s)
	}

	if status, _ := l.send(
		context.Background(),
		"end",
		envelope{Reason: ReasonDeclined},
	); status != http.StatusOK {
		t.Fatalf("end went out as %d", status)
	}
	if _, ok := h.lines["s1"]; ok {
		t.Error("the session outlived its end")
	}
	if s := (<-l.signals); s.Type != "end" || s.Reason != ReasonDeclined {
		t.Errorf("end carried %+v", s)
	}
	if s := wait(); s.Type != "end" {
		t.Errorf("a forgotten session told the browser %+v, want end", s)
	}
}
