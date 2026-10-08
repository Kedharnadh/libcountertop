package background

import (
	"fmt"
	"log/slog"
	"sync"
)

type Producer interface {
	Stand(down bool)

	Duck(on bool)

	Requeue()
}

type Arbiter struct {
	mu    sync.Mutex
	stack []Producer
	held  bool
	duck  bool
}

func (a *Arbiter) Took(p Producer) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.drop(p)
	if stood := a.top(); stood != nil && stood != p {
		slog.Debug("background handover", "from", kind(stood), "to", kind(p))
	}
	a.stack = append(a.stack, p)

	p.Duck(a.duck)
	a.tell()
}

func (a *Arbiter) Gave(p Producer) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.drop(p)
	if now := a.top(); now != nil && !a.held {
		slog.Debug("background resumed", "who", kind(now), "after", kind(p))
	}

	p.Stand(false)
	a.tell()
}

func (a *Arbiter) Stand(down bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.held = down
	a.tell()
}

func (a *Arbiter) Owns(p Producer) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.held {
		return false
	}
	top := a.top()
	return top == nil || top == p
}

func (a *Arbiter) Duck(on bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.duck == on {
		return
	}
	a.duck = on

	for _, p := range a.stack {
		p.Duck(on)
	}

	if heard := a.top(); on && heard != nil {
		heard.Requeue()
	}
}

func (a *Arbiter) Playing() Producer {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.top()
}

func (a *Arbiter) tell() {
	top := a.top()
	for _, p := range a.stack {
		p.Stand(a.held || p != top)
	}
}

func (a *Arbiter) top() Producer {
	if len(a.stack) == 0 {
		return nil
	}
	return a.stack[len(a.stack)-1]
}

func (a *Arbiter) drop(p Producer) {
	for i, have := range a.stack {
		if have == p {
			a.stack = append(a.stack[:i], a.stack[i+1:]...)
			return
		}
	}
}

func kind(p Producer) string {
	if p == nil {
		return "nothing"
	}
	return fmt.Sprintf("%T", p)
}
