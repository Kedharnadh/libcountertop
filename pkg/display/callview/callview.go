package callview

import (
	"math"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/audio/volume"
	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	"github.com/ygelfand/libcountertop/pkg/display/navigation"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/media/call"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
)

type Peers interface {
	Peers() []peer.Peer
	Find(id string) (peer.Peer, bool)
}

type Options struct {
	UI    *widgets.UI
	Calls *call.Calls
	Peers Peers
	Voice volume.Channel
	Video bool
}

type View struct{ o Options }

func New(o Options) *View { return &View{o: o} }

func (cv *View) Screen(v shell.View) *navigation.Screen {
	switch t := v.(type) {
	case *call.View:
		return cv.callScreen(t)
	case *call.Profile:
		return cv.profileScreen(t)
	}
	return nil
}

func (cv *View) Board(at ui.Rect, t dashboard.Tab, pal theme.Theme) gogui.View {
	return cv.callsBoard(at, t, pal)
}

func Glyph(dashboard.Tab) string { return gogui.IconPhone }

func iconStyle(c gogui.Color) gogui.TextStyle {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Color = c
	return st
}

func secondary() gogui.TextStyle {
	t := gogui.CurrentTheme().Cfg
	st := t.TextStyleDef
	st.Color = t.ColorTextSecondary
	return st
}

func reach() float32 { return style.Reach() }

func cell(width, gap float32, n int) float32 {
	return float32(math.Floor(float64((width - gap*float32(n-1)) / float32(n))))
}

func color(c theme.Color) gogui.Color { return gogui.RGB(c.R, c.G, c.B) }
