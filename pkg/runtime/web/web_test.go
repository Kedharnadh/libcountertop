package web

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/runtime/status"
)

func TestAdoptionHidesTheOffer(t *testing.T) {
	var adopted atomic.Bool
	s := New(Options{
		Port:    80,
		APIPort: 6053,
		Adopted: adopted.Load,
		Page:    func() status.Page { return status.Page{} },
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

	if _, ok := s.routes["POST /call/offer"]; !ok {
		t.Error("the route was dropped")
	}
	if s.Name() != "web" {
		t.Error(s.Name())
	}
}
