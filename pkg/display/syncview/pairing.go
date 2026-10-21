package syncview

import (
	"fmt"
	"strings"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/network/peer/pictures"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/configsync"
)

const codeLength = 6

func (v *View) pairing(w *gogui.Window) gogui.View {
	x, ok := v.o.Sync.Now()
	if !ok {
		return gogui.Column(gogui.ContainerCfg{Sizing: gogui.FillFill})
	}
	t := gogui.CurrentTheme().Cfg
	vw, vh := w.WindowSize()
	reach := style.Reach()
	wide := min(float32(vw)-2*reach, reach*12)

	status := t.TextStyleDef
	status.Size *= 1.4
	content := []gogui.View{
		v.logo(float32(vh) * 0.2),
		v.who(x, float32(vh)*0.12),
		gogui.Label(message(x), status),
	}

	switch {
	case x.State == configsync.Waiting && x.Role == configsync.Sending:
		content = append(content, code(x.Code))
	case x.State == configsync.Waiting && x.Role == configsync.Receiving:
		content = append(content, v.entry(x, wide, float32(vh)))
	}
	content = append(content, v.keys(x, wide))

	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		Padding: gogui.NewPadding(reach*0.5, reach, reach*0.5, reach),
		Spacing: gogui.SpacingLarge,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Color:   t.ColorBackground,
		Content: content,
	})
}

func (v *View) logo(tall float32) gogui.View {
	mark := v.o.Logo()
	mw, mh := mark.Bounds().Dx(), mark.Bounds().Dy()
	return gogui.Image(gogui.ImageCfg{
		Src:    widgets.ImageSource(fmt.Sprintf("sync-logo-%p", mark), mark),
		Width:  tall * float32(mw) / float32(max(mh, 1)),
		Height: tall,
	})
}

func (v *View) who(x configsync.Exchange, tall float32) gogui.View {
	key, svg := pictures.For(x.Peer.Board)
	name := gogui.CurrentTheme().Cfg.TextStyleDef
	name.Size *= 1.6
	return gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingMedium,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{
			gogui.Svg(
				gogui.SvgCfg{
					ID:      "sync-" + key,
					SvgData: svg,
					Width:   tall / pictures.Aspect,
					Height:  tall,
				},
			),
			gogui.Label(x.Peer.Name, name),
		},
	})
}

func code(digits string) gogui.View {
	st := gogui.CurrentTheme().Cfg.TextStyleDef
	st.Size = style.Reach() * 1.8
	st.Color = gogui.CurrentTheme().Cfg.ColorAccent
	if len(digits) == codeLength {
		digits = digits[:3] + " " + digits[3:]
	}
	return gogui.Label(digits, st)
}

func (v *View) entry(x configsync.Exchange, wide, vh float32) gogui.View {
	v.mu.Lock()
	if v.round != x.ID {
		v.round = x.ID
		v.pad.Reset()
	}
	v.mu.Unlock()
	typed := v.pad.Typed()
	shown := typed + strings.Repeat("·", codeLength-len(typed))
	pad := v.pad.View(&v.o.UI.Toolkit, wide, vh, codeLength, v.o.Sync.Enter)
	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingMedium,
		HAlign:  gogui.HAlignCenter,
		Content: []gogui.View{code(shown), pad},
	})
}

func (v *View) keys(x configsync.Exchange, wide float32) gogui.View {
	icon := gogui.CurrentTheme().TextStyleIconMedium
	icon.Size = style.Reach() * 0.6
	glyph, id := gogui.IconClose, "sync-cancel"
	if x.State == configsync.Done || x.State == configsync.Failed {
		glyph, id = gogui.IconCheck, "sync-done"
	}
	return gogui.Row(gogui.ContainerCfg{
		Width:   min(wide, style.Reach()*6),
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		Content: []gogui.View{
			v.o.UI.Toolkit.WideKey(
				id,
				glyph,
				icon,
				style.Reach()*1.4,
				style.Partial,
				func(gogui.EventCtx) { v.o.Sync.Close() },
			),
		},
	})
}

func message(x configsync.Exchange) string {
	with := map[string]any{"Peer": x.Peer.Name, "N": x.Applied}
	switch x.State {
	case configsync.Waiting:
		if x.Role == configsync.Sending {
			return say.F("sync.show", with)
		}
		if x.Reason == configsync.ReasonWrong {
			return say.F("sync.wrong", map[string]any{"N": x.Left})
		}
		return say.F("sync.enter", with)
	case configsync.Working:
		if x.Role == configsync.Sending {
			return say.T("sync.sending")
		}
		return say.T("sync.receiving")
	case configsync.Done:
		if x.Role == configsync.Sending {
			return say.F("sync.sent", with)
		}
		if x.Applied == 0 {
			return say.F("sync.matched", with)
		}
		return say.F("sync.received", with)
	}
	switch x.Reason {
	case configsync.ReasonExpired:
		return say.T("sync.expired")
	case configsync.ReasonBusy:
		return say.F("sync.busy", with)
	case configsync.ReasonCancelled:
		return say.T("sync.cancelled")
	case configsync.ReasonWrong:
		return say.T("sync.tries")
	}
	return say.T("sync.failed")
}
