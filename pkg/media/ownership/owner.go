package ownership

import "sync"

type Owner[T comparable] struct {
	mu                 sync.Mutex
	kind               func(T) uint8
	held, live         T
	heldKind, liveKind uint8
	hasHeld, hasLive   bool
}

func New[T comparable](kind func(T) uint8) *Owner[T] { return &Owner[T]{kind: kind} }

func (o *Owner[T]) Get() (T, bool) { o.mu.Lock(); defer o.mu.Unlock(); return o.held, o.hasHeld }
func (o *Owner[T]) Set(source T) {
	kind := o.kind(source)
	o.mu.Lock()
	o.held, o.hasHeld, o.heldKind = source, true, kind
	o.mu.Unlock()
}

func (o *Owner[T]) Clear() { o.mu.Lock(); var zero T; o.held, o.hasHeld = zero, false; o.mu.Unlock() }

func (o *Owner[T]) Begin(source T) (replaced T, fresh bool, displaced bool) {
	kind := o.kind(source)
	o.mu.Lock()
	defer o.mu.Unlock()
	fresh = !o.hasLive || o.liveKind != kind
	if fresh && o.hasHeld && o.heldKind != kind {
		replaced, displaced = o.held, true
	}
	o.live, o.hasLive = source, true
	o.held, o.hasHeld = source, true
	o.liveKind, o.heldKind = kind, kind
	return
}

func (o *Owner[T]) End(source T) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.hasLive && o.live == source {
		var zero T
		o.live, o.hasLive = zero, false
	}
}

func (o *Owner[T]) ResetSession() {
	o.mu.Lock()
	var zero T
	o.live, o.hasLive = zero, false
	o.mu.Unlock()
}

func (o *Owner[T]) Release(source T) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.hasHeld || o.held != source {
		return false
	}
	var zero T
	o.held, o.hasHeld = zero, false
	return true
}
