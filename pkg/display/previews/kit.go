package previews

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/weatherview"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
)

type Kit struct {
	ui      *widgets.UI
	hours   func() bool
	ink     func() theme.Theme
	weather weatherview.View
}

func NewKit(ui *widgets.UI, hours func() bool, ink func() theme.Theme) Kit {
	return Kit{
		ui:    ui,
		hours: hours,
		ink:   ink,
		weather: weatherview.View{
			UI:    ui,
			Again: func(*gogui.Window, time.Duration) {},
			Start: time.Now(),
		},
	}
}
