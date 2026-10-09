package tessera

import (
	"encoding/json"
	"strings"
)

const (
	lampsMost     = 24
	kelvinLeast   = 2000
	kelvinMost    = 6500
	lampColor     = 1
	lampWarmth    = 2
	kelvinFloor   = 1000
	kelvinCeiling = 15000
)

func (t Tile) Lamps() []Tile {
	if t.Domain() != "light" {
		return nil
	}
	var out []Tile
	for _, l := range t.ExtraList("lamps") {
		if len(out) == lampsMost {
			break
		}
		entity, _ := l["e"].(string)
		if !strings.HasPrefix(entity, "light.") {
			continue
		}
		out = append(out, lampTile(entity, l))
	}
	return out
}

func lampTile(entity string, l map[string]any) Tile {
	whole := func(key string) int {
		v, _ := l[key].(float64)
		return int(v)
	}
	kelvin := func(key string) int {
		if k := whole(key); k >= kelvinFloor && k <= kelvinCeiling {
			return k
		}
		return 0
	}
	name, _ := l["n"].(string)
	state := "off"
	switch {
	case whole("u") == 1:
		state = "unavailable"
	case whole("s") == 1:
		state = "on"
	}
	caps := whole("c")
	var modes []string
	if caps&lampColor != 0 {
		modes = append(modes, "hs")
	}
	if caps&lampWarmth != 0 {
		modes = append(modes, "color_temp")
	}
	switch {
	case len(modes) > 0:
	case whole("d") == 1:
		modes = []string{"brightness"}
	default:
		modes = []string{"onoff"}
	}
	attrs := map[string]any{
		"supported_color_modes": modes,
		"brightness":            min(100, whole("b")) * 255 / 100,
		"hs_color":              []int{min(360, whole("h")), min(100, whole("sa"))},
	}
	lo, hi := kelvin("lo"), kelvin("hi")
	if hi <= lo {
		lo, hi = kelvinLeast, kelvinMost
	}
	attrs["min_color_temp_kelvin"], attrs["max_color_temp_kelvin"] = lo, hi
	if k := kelvin("k"); k != 0 {
		attrs["color_temp_kelvin"] = k
	}
	raw, _ := json.Marshal(attrs)
	return Tile{Entity: entity, Name: name, State: state, Attributes: raw, Received: true}
}
