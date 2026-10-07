package settings

import (
	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
)

func PowerPage(push func(shell.View), restart, reboot func()) *shell.Page {
	open := func(v shell.View) func(int) { return func(int) { push(v) } }
	return &shell.Page{
		Title: say.T("power.title"),
		Build: func() ([]widget.Row, []func(int)) {
			return []widget.Row{
					{Label: say.T("power.restart"), Kind: widget.Chevron},
					{Label: say.T("power.reboot"), Kind: widget.Chevron},
				}, []func(int){
					open(ConfirmPage(say.T("power.restart"),
						say.T("power.restart.what"), restart)),
					open(ConfirmPage(say.T("power.reboot"),
						say.T("power.reboot.what"), reboot)),
				}
		},
	}
}
func ConfirmPage(title, what string, do func()) *shell.Page {
	return &shell.Page{
		Title: title,
		Build: func() ([]widget.Row, []func(int)) {
			return []widget.Row{
					{Label: what, Kind: widget.Plain},
					{Label: title},
				}, []func(int){
					nil,
					func(int) { do() },
				}
		},
	}
}
