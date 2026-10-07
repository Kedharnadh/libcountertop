package playerview

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/audio/volume"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	videoplayer "github.com/ygelfand/libcountertop/pkg/display/videopage"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	media "github.com/ygelfand/libcountertop/pkg/media/source"
)

// Player supplies the active playback and its transport.
type Player interface {
	Now() media.Now
	Named() string
	Transport() media.Controller
	Open()
}
type Dependencies struct {
	Player Player
	Volume volume.Channel
	Shell  *shell.Shell
	UI     *widgets.UI
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
func Color(c theme.Color) gogui.Color { return gogui.Color{R: c.R, G: c.G, B: c.B, A: 0xff} }
func reach() float32                  { return style.Reach() }
func secondary() gogui.TextStyle {
	t := gogui.CurrentTheme().Cfg
	st := t.TextStyleDef
	st.Color = t.ColorTextSecondary
	return st
}
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
		out = append(out, style.Span{From: float32(m.From) / float32(length), To: float32(m.To) / float32(length), Color: Color(m.Color)})
	}
	return out
}
