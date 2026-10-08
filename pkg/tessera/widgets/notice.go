package widgets

import (
	_ "embed"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

//go:embed mark.svg
var Mark string

func (v *View) Notice(at ui.Rect) gogui.View {
	v.pick(at)
	h := float32(at.H)
	title, body := v.Hint()
	if title == "" {
		return nil
	}
	t := gogui.CurrentTheme().Cfg
	big := t.TextStyleDef
	big.Size *= 1.3
	quiet := t.TextStyleDef
	quiet.Color = t.ColorTextSecondary
	side := min(t.TextStyleDef.Size*8, h*0.25)
	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.PaddingLarge,
		Spacing: gogui.SpacingLarge,
		Content: []gogui.View{
			gogui.Svg(gogui.SvgCfg{ID: "tessera-mark", SvgData: Mark, Width: side, Height: side}),
			gogui.Label(title, big),
			gogui.Text(gogui.TextCfg{Text: body, TextStyle: quiet, Mode: gogui.TextModeWrap}),
		},
	})
}
