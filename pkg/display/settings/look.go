package settings

import (
	"image"
	"log/slog"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (p *LookPages) Page(slot int) *shell.Page {
	return &shell.Page{
		Title: say.T("look.title"),
		Build: func() ([]menu.Row, []func(int)) {
			look := p.options.Preferences.Wake.Read().Slot(slot).Look
			rows := []menu.Row{
				{Label: say.T("look.place"), Kind: menu.Chevron, Value: look.Place.Label()},
			}
			taps := []func(int){p.open(func() shell.View { return p.Place(slot) })}
			for _, st := range config.Stages() {
				rows = append(
					rows,
					menu.Row{
						Label: st.Label(),
						Kind:  menu.Chevron,
						Value: LookName(look.Kind(st)),
					},
				)
				taps = append(taps, p.open(func() shell.View { return p.Stage(slot, st) }))
			}
			return rows, taps
		},
	}
}

func LookName(kind string) string {
	if kind == "" {
		return say.T("look.default")
	}
	return visual.Kind(kind).Label()
}

func (p *LookPages) Place(slot int) *shell.Page {
	return &shell.Page{
		Title: say.T("look.place"),
		Build: func() ([]menu.Row, []func(int)) {
			now := p.options.Preferences.Wake.Read().Slot(slot).Look.Place
			var rows []menu.Row
			var taps []func(int)
			for _, value := range config.LookPlaces() {
				rows = append(rows, menu.Row{Label: value.Label(), Chosen: value == now})
				taps = append(taps, func(int) {
					if err := p.options.Preferences.Word(
						slot,
						func(w *config.WakeWord) { w.Look.Place = value },
					); err != nil {
						slog.Error(
							"saving the assistant visual placement failed",
							"slot",
							slot+1,
							"err",
							err,
						)
					}
				})
			}
			return rows, taps
		},
	}
}

func (p *LookPages) Stage(slot int, st config.Stage) *shell.Page {
	use := func(kind string) {
		if err := p.options.Preferences.Word(slot, func(w *config.WakeWord) {
			switch st {
			case config.StageListening:
				w.Look.Listening = kind
			case config.StageWaiting:
				w.Look.Waiting = kind
			case config.StageResponding:
				w.Look.Responding = kind
			}
		}); err != nil {
			slog.Error(
				"saving the assistant visual failed",
				"slot",
				slot+1,
				"stage",
				st,
				"err",
				err,
			)
		}
		p.options.Shell.Pop()
	}
	return VisualPicker(st.Label(), say.T("look.default"),
		func() { use("") },
		func() string { return p.options.Preferences.Wake.Read().Slot(slot).Look.Kind(st) },
		func(k visual.Kind) { use(string(k)) }, p.options.Thumbnail)
}

type LookDependencies struct {
	Preferences Preferences
	Shell       *shell.Shell
	Thumbnail   func(visual.Kind, int, int) *image.RGBA
}
type LookPages struct{ options LookDependencies }

func NewLookPages(o LookDependencies) *LookPages { return &LookPages{options: o} }
func (p *LookPages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Shell.Push(makePage()) }
}
