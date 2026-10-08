package wakeslots

import (
	"slices"
	"sync"
)

func Initial(saved, installed []string, defaultID string, count int) []string {
	out := make([]string, max(count, 0))
	known := make(map[string]bool, len(installed))
	for _, id := range installed {
		known[id] = true
	}
	if saved != nil {
		for i := range out {
			if i < len(saved) && known[saved[i]] {
				out[i] = saved[i]
			}
		}
		return out
	}
	if len(out) > 0 {
		if known[defaultID] {
			out[0] = defaultID
		} else if len(installed) > 0 {
			out[0] = installed[0]
		}
	}
	return out
}
func Advertised(slots []string) []string {
	var out []string
	for _, id := range slots {
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}
func ID(slots []string, slot int) (string, bool) {
	if slot < 0 || slot >= len(slots) || slots[slot] == "" {
		return "", false
	}
	return slots[slot], true
}

type Selection struct {
	mu    sync.RWMutex
	slots []string
}

func (s *Selection) Set(slots []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.slots = slices.Clone(slots)
}

func (s *Selection) Get() []string { s.mu.RLock(); defer s.mu.RUnlock(); return slices.Clone(s.slots) }
func (s *Selection) ID(slot int) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ID(s.slots, slot)
}
