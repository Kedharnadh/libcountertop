package playerview

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/audio/volume"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	videoplayer "github.com/ygelfand/libcountertop/pkg/display/videopage"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/media/nowplaying"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Player interface {
	Now() nowplaying.Now
	Named() string
	Transport() nowplaying.Controller
	Open()
}
type Dependencies struct {
	Player Player
	Volume volume.Channel
	Shell  *shell.Shell
	UI     *widgets.UI
	Dock   func() schema.Edge
}
type Renderer struct {
	options Dependencies
	kit     widgets.Toolkit
	seeking scrub
}

func New(o Dependencies) *Renderer { return &Renderer{options: o, kit: o.UI.Toolkit} }
func (p *Renderer) Close() {
	p.seeking.mu.Lock()
	if p.seeking.timer != nil {
		p.seeking.timer.Stop()
	}
	p.seeking.sk = nil
	p.seeking.mu.Unlock()
}
func reach() float32 { return style.Reach() }
func iconStyle(c gogui.Color) gogui.TextStyle {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Color = c
	return st
}
func spans(marks []videoplayer.Mark, length time.Duration) []style.Span {
	if length <= 0 {
		return nil
	}
	out := make([]style.Span, 0, len(marks))
	for _, m := range marks {
		out = append(
			out,
			style.Span{
				From:  float32(m.From) / float32(length),
				To:    float32(m.To) / float32(length),
				Color: widgets.Color(m.Color),
			},
		)
	}
	return out
}
