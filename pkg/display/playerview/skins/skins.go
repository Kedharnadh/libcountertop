package skins

import (
	"embed"
	"slices"
)

//go:embed *.webp
var art embed.FS

type Action int

const (
	Previous Action = iota + 1
	Next
	Play
	Source
	Favorite
	Quieter
	Louder
)

type Rect struct{ X0, Y0, X1, Y1 float32 }

type Key struct {
	Do Action
	At Rect
}

type Skin struct {
	Name   string
	Label  string
	Dial   bool
	Screen Rect
	Keys   []Key
}

func (s Skin) Art() []byte {
	b, _ := art.ReadFile(s.Name + ".webp")
	return b
}

func All() []Skin { return slices.Clone(all) }

func Find(name string) (Skin, bool) {
	i := slices.IndexFunc(all, func(s Skin) bool { return s.Name == name })
	if i < 0 {
		return Skin{}, false
	}
	return all[i], true
}

func key(do Action, x0, y0, x1, y1 float32) Key { return Key{Do: do, At: Rect{x0, y0, x1, y1}} }

var all = []Skin{
	{Name: "1940s", Label: "1940s", Dial: true, Screen: Rect{.28, .60, .72, .77}, Keys: []Key{
		key(Previous, .22, .78, .34, .89), key(Play, .44, .78, .57, .89), key(Next, .66, .78, .79, .89),
	}},
	{Name: "1950s", Label: "1950s", Dial: true, Screen: Rect{.40, .23, .90, .48}, Keys: []Key{
		key(Previous, .42, .58, .53, .75), key(Play, .60, .58, .71, .75), key(Next, .77, .58, .88, .75),
	}},
	{Name: "1960s", Label: "1960s", Dial: true, Screen: Rect{.08, .13, .71, .33}, Keys: []Key{
		key(Previous, .77, .18, .84, .36), key(Next, .84, .18, .91, .36), key(Play, .80, .72, .90, .83),
	}},
	{Name: "1970s", Label: "1970s", Dial: true, Screen: Rect{.07, .19, .72, .43}, Keys: []Key{
		key(Previous, .75, .27, .845, .55), key(Next, .845, .27, .94, .55), key(Play, .74, .66, .83, .80),
	}},
	{Name: "1980s", Label: "1980s", Dial: true, Screen: Rect{.16, .14, .85, .29}, Keys: []Key{
		key(Play, .40, .80, .47, .89), key(Previous, .47, .80, .55, .89), key(Next, .55, .80, .62, .89),
	}},
	{Name: "1990s", Label: "1990s", Dial: true, Screen: Rect{.35, .37, .64, .50}, Keys: []Key{
		key(Previous, .34, .74, .415, .85), key(Play, .42, .74, .50, .85), key(Next, .50, .74, .58, .85),
	}},
	{Name: "2000s", Label: "2000s", Dial: true, Screen: Rect{.36, .34, .65, .56}, Keys: []Key{
		key(Previous, .34, .64, .40, .71), key(Play, .45, .64, .56, .71), key(Next, .60, .64, .66, .71),
	}},
	{Name: "2010s", Label: "2010s", Screen: Rect{.345, .20, .655, .57}, Keys: []Key{
		key(Source, .30, .69, .35, .81), key(Previous, .375, .69, .425, .81),
		key(Quieter, .455, .63, .50, .86), key(Louder, .50, .63, .545, .86),
		key(Play, .575, .69, .63, .81), key(Favorite, .65, .69, .70, .81),
	}},
	{Name: "2020s", Label: "2020s", Screen: Rect{.32, .17, .68, .63}, Keys: []Key{
		key(Source, .38, .73, .43, .84),
		key(Quieter, .455, .67, .50, .89), key(Louder, .50, .67, .545, .89),
		key(Previous, .565, .73, .62, .84), key(Next, .635, .73, .69, .84),
	}},
	{Name: "2030s", Label: "2030s", Screen: Rect{.24, .20, .76, .92}},
	{Name: "2040s", Label: "2040s", Screen: Rect{.11, .17, .89, .78}},
	{Name: "aura", Label: "Aura", Screen: Rect{.28, .13, .72, .75}, Keys: []Key{
		key(Source, .345, .79, .415, .90),
		key(Quieter, .445, .76, .482, .93), key(Louder, .482, .76, .52, .93),
		key(Previous, .55, .79, .595, .90), key(Play, .61, .79, .66, .90), key(Next, .665, .79, .72, .90),
	}},
	{Name: "pulse", Label: "Pulse", Screen: Rect{.18, .16, .82, .85}},
}
