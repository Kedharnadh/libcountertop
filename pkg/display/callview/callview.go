package callview

import (
	"math"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/audio/volume"
	"github.com/ygelfand/libcountertop/pkg/display/drawer"
	"github.com/ygelfand/libcountertop/pkg/display/navigation"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/media/call"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/say"
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
	case *call.Directory:
		return cv.directoryScreen(t)
	}
	return nil
}

func (cv *View) Rail() drawer.Entry {
	return drawer.Entry{
		Name:  func() string { return say.T("call.tab") },
		Order: drawer.OrderCalls,
		Glyph: func() string { return gogui.IconPhone },
		Open:  cv.o.Calls.Browse,
	}
}

func (cv *View) directoryScreen(d *call.Directory) *navigation.Screen {
	return &navigation.Screen{
		Title: say.T("call.tab"),
		View:  d,
		Build: func(w *gogui.Window) gogui.View {
			vw, vh := w.WindowSize()
			at := ui.Rect{W: vw - 2*gogui.PadLarge, H: vh}
			return cv.callsBoard(at, cv.o.UI.Palette())
		},
	}
}

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
