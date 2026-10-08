package schema

import (
	"slices"

	"github.com/ygelfand/libcountertop/pkg/say"
)

type Home struct {
	Enabled bool                `json:"enabled"`
	Mode    HomeMode            `json:"mode"`
	Picks   map[string]HomePick `json:"picks"`
	Control map[string]bool     `json:"control"`
	Combine bool                `json:"combine"`
	Group   map[string]int      `json:"group"`
	Tessera TesseraGrid         `json:"tessera"`
	Return  bool                `json:"tessera_return"`
	Wide    bool                `json:"tessera_landscape"`
	Tall    bool                `json:"tessera_portrait"`
}

const (
	HomeGroupDefault = 2
	HomeGroupMost    = 5
)

type HomeMode string

const (
	HomeNative  HomeMode = "native"
	HomeTessera HomeMode = "tessera"
)

func (m HomeMode) Label() string {
	switch m {
	case HomeNative:
		return say.T("home.mode.native")
	case HomeTessera:
		return say.T("home.mode.tessera")
	}
	return string(m)
}

func HomeModes() []HomeMode { return []HomeMode{HomeNative, HomeTessera} }

type TesseraGrid struct {
	Columns int `json:"columns"`
	Rows    int `json:"rows"`
}

const (
	TesseraMost  = 12
	TesseraCells = 64
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

func (p HomePick) Clone() HomePick {
	p.Labels = slices.Clone(p.Labels)
	p.Areas = slices.Clone(p.Areas)
	p.Entities = slices.Clone(p.Entities)
	return p
}

func DefaultHome() Home {
	return Home{Enabled: true, Mode: HomeNative, Wide: true, Tall: true}
}

func (h Home) Picked(key string) HomePick { return h.Picks[key] }

func (h Home) Grouped(key string) int {
	if n, ok := h.Group[key]; ok {
		return n
	}
	return HomeGroupDefault
}

func (h Home) Speaks() HomeMode {
	if h.Mode == HomeTessera {
		return HomeTessera
	}
	return HomeNative
}

func (g TesseraGrid) Or(fallback TesseraGrid) TesseraGrid {
	if g.Columns <= 0 {
		g.Columns = fallback.Columns
	}
	if g.Rows <= 0 {
		g.Rows = fallback.Rows
	}
	g.Columns = min(max(g.Columns, 1), TesseraMost)
	g.Rows = min(max(g.Rows, 1), TesseraMost, TesseraCells/g.Columns)
	return g
}
