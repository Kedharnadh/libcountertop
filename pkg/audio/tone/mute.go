package tone

import (
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

// MuteTable keeps the paired mute/unmute option alongside uploaded sounds.
func MuteTable[T any](
	field func(*T) *schema.Chime,
	more func() []setting.Option,
) *setting.Table[T] {
	const group setting.Group = "Microphone"
	row := setting.Pick(
		setting.Setting[T]{
			Name:  "mute_sound",
			ID:    "mute_sound",
			Group: group,
			Icon:  "mdi:music-note",
			More:  more,
		},
		append(schema.Chimes(), schema.ChimeMuteSwitch),
		field,
	)
	return setting.NewTable(
		"microphone",
		[]setting.Group{group},
		[]setting.Setting[T]{row},
		setting.Messages{Text: say.T, Missing: say.Missing},
	)
}

// MuteSound resolves an uploaded or built-in sound, preserving the paired
// acknowledgement when an uploaded sound is missing.
func MuteSound(sounds Sounds, c schema.Chime, muted bool) Sound {
	fallback := schema.ChimeMuted
	if !muted {
		fallback = schema.ChimeUnmuted
	}
	if c == schema.ChimeMuteSwitch {
		c = fallback
	}
	return sounds.Sound(c, fallback)
}
