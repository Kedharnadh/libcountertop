package session

import (
	"sync"
	"testing"
	"time"
)

func TestAQueuedOldExpiryCannotCloseARefreshedSession(t *testing.T) {
	var owner sync.Mutex
	var idle Idle
	old := make(chan struct{}, 1)
	fresh := make(chan struct{}, 1)
	owner.Lock()
	idle.Schedule(time.Millisecond, &owner, func() { old <- struct{}{} })
	// Keep the expiry queued behind the hub lock, then renew the lease.
	time.Sleep(15 * time.Millisecond)
	idle.Schedule(30*time.Millisecond, &owner, func() { fresh <- struct{}{} })
	owner.Unlock()
	select {
	case <-old:
		t.Fatal("old timer closed refreshed session")
	case <-fresh:
	case <-time.After(time.Second):
		t.Fatal("new lease never expired")
	}
}
