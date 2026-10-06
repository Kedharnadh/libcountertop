package startup

import (
	"testing"
	"time"
)

func TestSkipDoesNotBypassReadiness(t *testing.T) {
	var c Controller
	p := []Progress{{Name: "network"}}
	for at := Frame; at < 20*time.Second; at += Frame {
		c.Tick(at, p, false, true)
	}
	if c.Ready != 0 {
		t.Fatalf("skipping without readiness gave finish=%v", c.Ready)
	}
	for at := 20 * time.Second; at < 24*time.Second; at += Frame {
		c.Tick(at, p, true, true)
	}
	if c.Ready != 1 {
		t.Fatalf("ready after skip gave finish=%v, want 1", c.Ready)
	}
}
func TestReadinessLossStopsHandover(t *testing.T) {
	var c Controller
	p := []Progress{{Done: true}}
	for at := Frame; at < 8*time.Second; at += Frame {
		c.Tick(at, p, true, false)
	}
	if c.finishing == 0 {
		t.Fatal("ready components never began handover")
	}
	c.Tick(8*time.Second, p, false, false)
	if c.finishing != 0 || c.done != 0 {
		t.Fatal("lost readiness retained handover")
	}
}
func TestSummaryTracksFailureAndCompletion(t *testing.T) {
	p := []Progress{{Name: "audio", Doing: "open"}}
	before := Summary(p)
	p[0].Failed = true
	if Summary(p) == before {
		t.Fatal("failure did not invalidate progress")
	}
	before = Summary(p)
	p[0].Failed = false
	p[0].Done = true
	if Summary(p) == before {
		t.Fatal("completion did not invalidate progress")
	}
}

func TestDisplayUnifiesBoardNamesWithoutChangingReadiness(t *testing.T) {
	p := []Progress{{Name: "wifi", Doing: "joining"}, {Name: "capture", Failed: true}, {Name: "display", Done: true}}
	rows := Display(p)
	if rows[0].Name != "Display" || rows[1].Name != "Microphones" || rows[2].Name != "Wi-Fi" {
		t.Fatalf("Display order = %+v", rows)
	}
	if p[1].Name != "capture" || !rows[1].Failed || rows[2].Doing != "joining" {
		t.Fatal("Display changed source or status")
	}
}
