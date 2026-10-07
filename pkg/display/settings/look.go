package settings

import (
	"image"
	"log/slog"

	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (p *LookPages) Page(slot int) *shell.Page {
	return &shell.Page{
		Title: say.T("look.title"),
		Build: func() ([]widget.Row, []func(int)) {
			look := p.options.Read(slot)
			rows := []widget.Row{{Label: say.T("look.place"), Kind: widget.Chevron, Value: look.Place.Label()}}
			taps := []func(int){p.open(func() shell.View { return p.Place(slot) })}
			for _, st := range config.Stages() {
				rows = append(rows, widget.Row{Label: st.Label(), Kind: widget.Chevron, Value: LookName(look.Kind(st))})
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
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Read(slot).Place
			var rows []widget.Row
			var taps []func(int)
			for _, value := range config.LookPlaces() {
				rows = append(rows, widget.Row{Label: value.Label(), Chosen: value == now})
				taps = append(taps, func(int) {
					if err := p.options.Place(slot, value); err != nil {
						slog.Error("saving the assistant visual placement failed", "slot", slot+1, "err", err)
					}
				})
			}
			return rows, taps
		},
	}
}

func (p *LookPages) Stage(slot int, st config.Stage) *shell.Page {
	use := func(kind string) {
		if err := p.options.Visual(slot, st, kind); err != nil {
			slog.Error("saving the assistant visual failed", "slot", slot+1, "stage", st, "err", err)
		}
		p.options.Pop()
	}
	return VisualPicker(st.Label(), say.T("look.default"),
		func() { use("") },
		func() string { return p.options.Read(slot).Kind(st) },
		func(k visual.Kind) { use(string(k)) }, p.options.Thumbnail)
}

type LookOptions struct {
	Read      func(int) config.Look
	Place     func(int, config.LookPlace) error
	Visual    func(int, config.Stage, string) error
	Push      func(shell.View)
	Pop       func()
	Thumbnail func(visual.Kind, int, int) *image.RGBA
}
type LookPages struct{ options LookOptions }

func NewLookPages(o LookOptions) *LookPages { return &LookPages{options: o} }
func (p *LookPages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Push(makePage()) }
}
