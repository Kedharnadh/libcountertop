package settings

import (
	"fmt"
	"log/slog"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Preferences struct {
	Screen  storage.Section[config.Screen]
	Clock   storage.Section[config.Clock]
	Idle    storage.Section[config.Idle]
	Visual  storage.Section[config.Visual]
	Poster  storage.Section[config.Poster]
	Weather storage.Section[config.Weather]
	Wake    storage.Section[config.Wake]
	Call    storage.Section[config.Call]
}

func (p Preferences) Basic(size string) BasicState {
	return BasicState{
		Screen:      p.Screen.Read(),
		Clock:       p.Clock.Read(),
		Idle:        p.Idle.Read(),
		Visual:      p.Visual.Read(),
		PosterLabel: PosterSays(p.Poster.Read()),
		UISize:      size,
	}
}
func (p Preferences) ClockState() ClockState {
	return ClockState{Screen: p.Screen.Read(), Clock: p.Clock.Read()}
}
func (p Preferences) IdleState() IdleState {
	return IdleState{
		Screen:  p.Screen.Read(),
		Clock:   p.Clock.Read(),
		Idle:    p.Idle.Read(),
		Visual:  p.Visual.Read(),
		Weather: p.Weather.Read(),
	}
}

func saveScreen(
	section storage.Section[config.Screen],
	stack *shell.Shell,
	name string,
	change func(*config.Screen),
) {
	if err := section.Update(change); err != nil {
		slog.Error("saving a setting failed", "setting", name, "err", err)
	}
	stack.Redraw()
}
func (p Preferences) Word(slot int, change func(*config.WakeWord)) error {
	if slot < 0 {
		return fmt.Errorf("config: wake slot %d", slot)
	}
	return p.Wake.Update(func(w *config.Wake) {
		for len(w.Words) <= slot {
			w.Words = append(w.Words, config.DefaultWakeWord())
		}
		change(&w.Words[slot])
	})
}
