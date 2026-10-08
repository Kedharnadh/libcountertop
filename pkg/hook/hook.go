package hook

import "sync"

type Hook[T any] struct {
	mu   sync.Mutex
	next int
	fns  map[int]func(T)
}

func (h *Hook[T]) Listen(fn func(T)) (cancel func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.fns == nil {
		h.fns = map[int]func(T){}
	}
	id := h.next
	h.next++
	h.fns[id] = fn

	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.fns, id)
	}
}

func (h *Hook[T]) Emit(v T) {
	h.mu.Lock()
	fns := make([]func(T), 0, len(h.fns))
	for _, fn := range h.fns {
		fns = append(fns, fn)
	}
	h.mu.Unlock()

	for _, fn := range fns {
		fn(v)
	}
}

func (h *Hook[T]) Listeners() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.fns)
}
