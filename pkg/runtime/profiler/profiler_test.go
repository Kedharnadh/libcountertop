package profiler

import (
	"net/http"
	"testing"
)

func TestTheProfilerOpensAndCloses(t *testing.T) {
	s := &Server{Addr: "127.0.0.1:0"}
	if s.Listening() != "" {
		t.Fatal("listening before it was opened")
	}

	at, err := s.Open()
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := s.Open(); again != at {
		t.Errorf("a second open moved it from %s to %s", at, again)
	}

	resp, err := http.Get("http://" + at + "/debug/pprof/cmdline")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("the profiler answered %s", resp.Status)
	}

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if s.Listening() != "" {
		t.Error("still listening after it was closed")
	}
	if _, err := http.Get("http://" + at + "/debug/pprof/cmdline"); err == nil {
		t.Error("the profiler still answers after it was closed")
	}
}
