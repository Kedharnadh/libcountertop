// Package session provides idle camera-session retention without stale timer callbacks.
package session

import (
	"sync"
	"time"
)

const Linger = 30 * time.Second

// Idle is guarded by the owning session hub's mutex, including Cancel and Schedule calls.
type Idle struct {
	timer      *time.Timer
	generation uint64
}

func (i *Idle) Cancel() {
	i.generation++
	if i.timer != nil {
		i.timer.Stop()
		i.timer = nil
	}
}
func (i *Idle) Schedule(delay time.Duration, owner sync.Locker, expire func()) {
	i.Cancel()
	generation := i.generation
	i.timer = time.AfterFunc(delay, func() {
		owner.Lock()
		defer owner.Unlock()
		if generation != i.generation {
			return
		}
		i.timer = nil
		expire()
	})
}
