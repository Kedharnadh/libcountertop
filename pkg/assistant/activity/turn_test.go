package activity

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentCompletionEmitsOnceOutsideLock(t *testing.T) {
	var count atomic.Int32
	var turn *Turn
	log := New(Options{Emit: func(fields map[string]string) {
		count.Add(1)
		turn.Heard("late transcript")
		if fields["heard"] != "original" || fields["outcome"] != string(Completed) {
			t.Errorf("completion snapshot: %v", fields)
		}
	}})
	turn = log.Begin(1, "alexa")
	turn.Listening()
	turn.Heard("original")
	var callers sync.WaitGroup
	for range 32 {
		callers.Go(func() { turn.Ends(Completed) })
	}
	callers.Wait()
	if count.Load() != 1 {
		t.Fatalf("completion emitted %d times", count.Load())
	}
	turn.mu.Lock()
	defer turn.mu.Unlock()
	if turn.heard != "original" {
		t.Fatal("closed turn changed after completion")
	}
}
