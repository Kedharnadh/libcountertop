package tessera

import "fmt"

var features = []string{
	"climate_range",
	"media_library",
	"screensaver",
	"saver_keys",
	"saver_items",
	"speaker_groups",
	"bar_status",
}

func (s *Screen) answer(status string) map[string]any {
	g := s.grid(s.size())
	sizes := []string{"single", "wide", "full"}
	if g.Rows >= 2 {
		sizes = append(sizes, "tall")
	}
	if g.Columns >= 2 && g.Rows >= 2 {
		sizes = append(sizes, "square")
	}
	for c := 1; c <= g.Columns; c++ {
		for r := 1; r <= g.Rows; r++ {
			if (c > 2 || r > 2) && (c != g.Columns || r != g.Rows) {
				sizes = append(sizes, fmt.Sprintf("%dx%d", c, r))
			}
		}
	}
	page, home, shown := 0, 0, 0
	if s.session.active {
		home = s.layout.Home + 1
		page = home
		for _, p := range s.layout.Pages {
			if !p.Excluded {
				shown++
			}
		}
	}
	return map[string]any{
		"status":             status,
		"rev":                s.layout.Revision,
		"protocol":           protocol,
		"bar_values":         1,
		"appearance_updates": 1,
		"group_lamps":        1,
		"tile_keys":          1,
		"tile_repeats":       1,
		"free_pages":         1,
		"max_tiles":          min(maxTiles, maxPages*g.Columns*g.Rows),
		"max_pages":          maxPages,
		"max_bar_items":      barItems,
		"features":           features,
		"tile_sizes":         sizes,
		"request":            keyText(s.session.hello),
		"session":            keyText(s.session.lease),
		"seq":                s.session.sequence,
		"applied":            s.session.active,
		"page":               page,
		"home":               home,
		"pagination_count":   shown,
	}
}
