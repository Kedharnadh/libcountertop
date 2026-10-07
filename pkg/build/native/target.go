// Package native resolves native helper targets from hardware metadata.
package native

import "fmt"

// Target is a board's supported native helper ABI. API zero means no helpers.
type Target struct {
	Name string
	API  int
}

// Select resolves one board or a uniform set of boards for a shared artifact.
// Mixed APIs require separate artifacts or API-specific embedded payloads.
func Select(targets []Target, board string) (int, error) {
	api := 0
	for _, t := range targets {
		if board != "" && t.Name != board {
			continue
		}
		if t.API == 0 {
			if board != "" {
				return 0, fmt.Errorf("board %q has no native helpers", board)
			}
			continue
		}
		if t.API != 25 && t.API != 27 {
			return 0, fmt.Errorf("board %q uses unsupported native API %d", t.Name, t.API)
		}
		if api != 0 && api != t.API {
			return 0, fmt.Errorf("native targets mix API %d and %d; select a board or build separate payloads", api, t.API)
		}
		api = t.API
	}
	if api == 0 {
		return 0, fmt.Errorf("no native helper target for board %q", board)
	}
	return api, nil
}
