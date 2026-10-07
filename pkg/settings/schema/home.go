package schema

import (
	"slices"
)

type Home struct {
	Enabled bool                `json:"enabled"`
	Picks   map[string]HomePick `json:"picks"`
	Control map[string]bool     `json:"control"`
	Combine bool                `json:"combine"`
	Group   map[string]int      `json:"group"`
}

const (
	HomeGroupDefault = 2
	HomeGroupMost    = 5
)

type HomePick struct {
	All      bool     `json:"all"`
	Labels   []string `json:"labels"`
	Areas    []string `json:"areas"`
	Entities []string `json:"entities"`
}

func (p HomePick) Empty() bool {
	return !p.All && len(p.Labels) == 0 && len(p.Areas) == 0 && len(p.Entities) == 0
}

// Clone returns a pick with detached selection slices.
func (p HomePick) Clone() HomePick {
	p.Labels = slices.Clone(p.Labels)
	p.Areas = slices.Clone(p.Areas)
	p.Entities = slices.Clone(p.Entities)
	return p
}

func DefaultHome() Home { return Home{Enabled: true} }

func (h Home) Picked(key string) HomePick { return h.Picks[key] }

func (h Home) Grouped(key string) int {
	if n, ok := h.Group[key]; ok {
		return n
	}
	return HomeGroupDefault
}
