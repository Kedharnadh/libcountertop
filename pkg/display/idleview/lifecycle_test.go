package idleview

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
)

func TestCloseJoinsAndAllowsReopen(t *testing.T) {
	v := New(Dependencies{Read: func() Settings { return Settings{} }, Shell: shell.New()})
	for range 2 {
		v.Keep(100, 100)
		v.Close()
		v.mu.Lock()
		running := v.running
		v.mu.Unlock()
		if running {
			t.Fatal("worker still running after Close")
		}
	}
}
