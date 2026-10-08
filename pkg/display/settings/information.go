package settings

import (
	"log/slog"
	"strconv"

	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
)

type About struct{ Name, Version, Built, Commit string }

func AboutPage(read func() About) *shell.Page {
	return &shell.Page{Title: say.T("about.title"), Build: func() ([]widget.Row, []func(int)) {
		c := read()
		return []widget.Row{
			{Label: say.T("about.name"), Kind: widget.Plain, Value: c.Name},
			{Label: say.T("about.version"), Kind: widget.Plain, Value: c.Version},
			{Label: say.T("about.built"), Kind: widget.Plain, Value: c.Built},
			{Label: say.T("about.commit"), Kind: widget.Plain, Value: c.Commit},
		}, nil
	}}
}

func PresencePage[K any](
	table func() *setting.Table[K],
	read func() K,
	save func(string, string) error,
) *shell.Page {
	return &shell.Page{Title: say.T("display.presence"), Build: func() ([]widget.Row, []func(int)) {
		c := read()
		wake, reach := table().Row("wake"), table().Row("range")
		set := func(name, value string) {
			if err := save(name, value); err != nil {
				slog.Error("the presence setting could not be saved", "setting", name, "err", err)
			}
		}
		return []widget.Row{
				{Label: wake.Title(), Kind: widget.Toggle, On: wake.On(&c)},
				{
					Label: reach.Title(),
					Kind:  widget.Slider,
					Level: reach.Percent(reach.Level(&c)),
					Value: reach.Read(&c),
					Snap:  func(v int) int { return reach.Percent(reach.Raw(v)) },
				},
			}, []func(int){
				func(int) { set("wake", setting.OnOff(!wake.On(&c))) },
				func(v int) { set("range", strconv.Itoa(reach.Raw(v))) },
			}
	}}
}
