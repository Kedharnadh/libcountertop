package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAdoptionHidesTheOffer(t *testing.T) {
	var adopted atomic.Bool
	s := New(Options{
		Port:    80,
		APIPort: 6053,
		Adopted: adopted.Load,
	})

	var heard []string
	s.Offered.Listen(func(addr string) { heard = append(heard, addr) })

	s.offer()
	if got := s.Offering(); got != "" && !strings.HasSuffix(got, ":6053") {
		t.Errorf("offering %q, want the API port", got)
	}

	adopted.Store(true)
	s.offer()
	if got := s.Offering(); got != "" {
		t.Errorf("still offering %q after adoption", got)
	}
	if len(heard) > 0 && heard[len(heard)-1] != "" {
		t.Errorf("the last word was %q, want the offer withdrawn", heard[len(heard)-1])
	}
}

func TestRoutesAreKept(t *testing.T) {
	s := New(Options{})
	s.Handle("POST /call/offer", http.NotFoundHandler())

	if _, ok := s.pages.routes["POST /call/offer"]; !ok {
		t.Error("the route was dropped")
	}
	if s.Name() != "web" {
		t.Error(s.Name())
	}
}

func TestAMethodPatternOnlyAnswersThatMethod(t *testing.T) {
	r := chi.NewRouter()
	route(r, "POST /call/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(chi.URLParam(r, "id")))
	}))

	post := httptest.NewRecorder()
	r.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/call/c1", nil))
	if post.Code != http.StatusOK || post.Body.String() != "c1" {
		t.Errorf("POST answered %d %q, want 200 c1", post.Code, post.Body.String())
	}
	get := httptest.NewRecorder()
	r.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/call/c1", nil))
	if get.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET answered %d, want 405", get.Code)
	}
}

func TestThePlainPageMovesToHTTPS(t *testing.T) {
	for _, c := range []struct {
		port int
		host string
		to   string
	}{
		{443, "kitchen.home:80", "https://kitchen.home/"},
		{443, "kitchen.home", "https://kitchen.home/"},
		{8444, "10.0.0.5", "https://10.0.0.5:8444/"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = c.host
		front(c.port).ServeHTTP(w, req)
		if w.Code != http.StatusFound || w.Header().Get("Location") != c.to {
			t.Errorf(
				"%s: got %d %q, want %d %q",
				c.host, w.Code, w.Header().Get("Location"), http.StatusFound, c.to,
			)
		}
	}
}
