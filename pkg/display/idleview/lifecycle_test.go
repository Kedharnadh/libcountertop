package idleview

import (
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"testing"
)

func TestCloseJoinsAndAllowsReopen(t *testing.T) {
	v := New(Options{Read: func() Settings { return Settings{} }, Visible: func(shell.View) bool { return true }, Redraw: func() {}})
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
