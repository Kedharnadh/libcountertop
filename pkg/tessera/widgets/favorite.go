package widgets

import (
	"strconv"
	"strings"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func favorite(
	t tessera.Tile,
) bool {
	return t.Display == "favorite" && t.Domain() == "media_player"
}

func favoritePlaying(t tessera.Tile) bool {
	n, _ := t.ExtraNumber("fp")
	return n == 1
}

func favoriteLine(t tessera.Tile) string {
	kind, source, now := t.ExtraText("fk"), t.ExtraText("fo"), t.ExtraText("so")
	if favoritePlaying(t) {
		return join(word(t, tessera.Numbers{}), now)
	}
	return join(kind, source)
}

func join(parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, " · ")
}

func (v *View) playFavorite(w *gogui.Window, t tessera.Tile) {
	if favoritePlaying(t) && t.State == "playing" {
		v.send(w, t, "media_player.media_pause", nil, nil, "")
		return
	}
	v.mu.Lock()
	s := v.screen
	v.mu.Unlock()
	if s == nil {
		return
	}
	l, _ := s.Layout()
	if i := indexOf(t, l); i >= 0 {
		s.Tell("screen_play", map[string]string{"entity": t.Entity, "tile": strconv.Itoa(i)})
	}
}

func indexOf(t tessera.Tile, l tessera.Layout) int {
	for i, c := range l.Tiles {
		if c.Entity == t.Entity && c.Slot == t.Slot && c.Parent == t.Parent {
			return i
		}
	}
	return -1
}
