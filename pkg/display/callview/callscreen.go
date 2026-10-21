package callview

import (
	"fmt"
	"strings"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/callvideo"
	"github.com/ygelfand/libcountertop/pkg/display/geometry"
	"github.com/ygelfand/libcountertop/pkg/display/homeview"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/navigation"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/media/call"
	"github.com/ygelfand/libcountertop/pkg/network/peer/pictures"
	"github.com/ygelfand/libcountertop/pkg/say"
)

func (cv *View) callScreen(v *Talk) *navigation.Screen {
	return &navigation.Screen{Fixed: true, Clear: true, View: v, Build: cv.callBody}
}

func (cv *View) callBody(w *gogui.Window) gogui.View {
	c, ok := cv.o.Calls.Now()
	if !ok {
		return gogui.Column(gogui.ContainerCfg{Sizing: gogui.FillFill})
	}
	t := gogui.CurrentTheme().Cfg
	vw, vh := w.WindowSize()

	name := t.TextStyleDef
	name.Size *= 2
	status := widgets.Secondary()
	status.Size *= 1.3

	controls := []gogui.View{cv.callButtons(vw, c)}
	if c.State == call.Talking {
		level := cv.o.Voice.Level()
		controls = append([]gogui.View{cv.o.UI.Row("call-volume", "call",
			menu.Row{Label: say.T("call.volume"), Kind: menu.Slider, Level: level},
			func(l int) { cv.o.Voice.Set(l) }, false)}, controls...)
	}
	bottomFor := func(width float32) gogui.View {
		return gogui.Column(gogui.ContainerCfg{
			Width:   min(width-2*reach(), reach()*12),
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingMedium,
			Content: controls,
		})
	}
	bottom := bottomFor(float32(vw))
	pad := gogui.NewPadding(reach(), reach(), reach(), reach())
	ink := t.TextStyleDef.Color
	heading := []gogui.View{gogui.Label(c.Peer.Name, name)}
	if c.State == call.Talking && c.RemoteMuted && !c.Video {
		heading = append(
			heading,
			mark("call-remote-muted", "micoff", ink, t.ColorBackground, name.Size),
		)
	}
	title := gogui.Row(
		gogui.ContainerCfg{
			Sizing:  gogui.FitFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingMedium,
			VAlign:  gogui.VAlignMiddle,
			Content: heading,
		},
	)

	if c.Video && c.State == call.Talking {
		remote, own := cv.o.Calls.Sizes()
		l := callvideo.Arrange(vw, vh, remote, own)
		rect := func(r geometry.Rect) ui.Rect { return ui.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H} }
		var layers []gogui.View
		if c.RemoteCameraOff {
			layers = append(layers, widgets.Placed(rect(l.Remote), gogui.Column(gogui.ContainerCfg{
				Sizing:  gogui.FillFill,
				Padding: gogui.NoPadding,
				HAlign:  gogui.HAlignCenter,
				VAlign:  gogui.VAlignMiddle,
				Color:   t.ColorBackground,
				Content: []gogui.View{
					mark(
						"call-remote-camera",
						"cameraoff",
						t.ColorTextSecondary,
						t.ColorBackground,
						float32(min(l.Remote.W, l.Remote.H))*0.25,
					),
				},
			})))
		}
		if c.RemoteMuted {
			side := reach() * 1.1
			inset := int(reach() * 0.4)
			layers = append(
				layers,
				widgets.Placed(
					ui.Rect{
						X: l.Picture.X + inset,
						Y: l.Picture.Y + inset,
						W: int(side),
						H: int(side),
					},
					gogui.Column(gogui.ContainerCfg{
						Sizing:  gogui.FillFill,
						Padding: gogui.NoPadding,
						Radius:  gogui.RadiusMedium,
						HAlign:  gogui.HAlignCenter,
						VAlign:  gogui.VAlignMiddle,
						Color:   t.ColorBackground.WithOpacity(0.6),
						Content: []gogui.View{
							mark("call-remote-muted", "micoff", ink, t.ColorBackground, side*0.65),
						},
					}),
				),
			)
		}
		if c.CameraOff {
			layers = append(layers, widgets.Placed(rect(l.Self), gogui.Column(gogui.ContainerCfg{
				Sizing:  gogui.FillFill,
				Padding: gogui.NoPadding,
				Radius:  gogui.RadiusMedium,
				HAlign:  gogui.HAlignCenter,
				VAlign:  gogui.VAlignMiddle,
				Color:   t.ColorBackground,
				Content: []gogui.View{
					mark(
						"call-self-camera",
						"cameraoff",
						t.ColorTextSecondary,
						t.ColorBackground,
						float32(min(l.Self.W, l.Self.H))*0.4,
					),
				},
			})))
		}
		inset := int(reach() * 0.4)
		badge := gogui.Column(gogui.ContainerCfg{
			Sizing:  gogui.FitFit,
			Padding: gogui.NewPadding(reach()*0.2, reach()*0.4, reach()*0.2, reach()*0.4),
			Radius:  gogui.RadiusMedium,
			HAlign:  gogui.HAlignRight,
			Color:   t.ColorBackground.WithOpacity(0.6),
			Content: []gogui.View{
				gogui.Label(c.Peer.Name, name),
				gogui.Label(callStatus(c), status),
			},
		})
		layers = append(
			layers,
			widgets.Placed(
				ui.Rect{
					X: l.Picture.X,
					Y: l.Picture.Y + inset,
					W: l.Picture.W - inset,
					H: l.Picture.H / 2,
				},
				gogui.Row(gogui.ContainerCfg{
					Sizing:  gogui.FillFill,
					Padding: gogui.NoPadding,
					HAlign:  gogui.HAlignRight,
					VAlign:  gogui.VAlignTop,
					Content: []gogui.View{badge},
				}),
			),
		)
		layers = append(layers, widgets.Placed(rect(l.Panel), gogui.Column(gogui.ContainerCfg{
			Sizing:  gogui.FillFill,
			Padding: pad,
			Spacing: gogui.SpacingSmall,
			Content: []gogui.View{
				gogui.Column(gogui.ContainerCfg{Sizing: gogui.FillFill, Padding: gogui.NoPadding}),
				gogui.Row(
					gogui.ContainerCfg{
						Sizing:  gogui.FillFit,
						Padding: gogui.NoPadding,
						HAlign:  gogui.HAlignCenter,
						Content: []gogui.View{bottomFor(float32(l.Panel.W))},
					},
				),
			},
		})))
		return gogui.Column(
			gogui.ContainerCfg{Sizing: gogui.FillFill, Padding: gogui.NoPadding, Content: layers},
		)
	}
	tall := min(float32(vh)*0.3, float32(vw)*0.5*pictures.Aspect)
	key, svg := pictures.For(c.Peer.Board)
	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		Padding: pad,
		Spacing: gogui.SpacingLarge,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Color:   t.ColorBackground,
		Content: []gogui.View{
			gogui.Svg(
				gogui.SvgCfg{
					ID:      "call-" + key,
					SvgData: svg,
					Width:   tall / pictures.Aspect,
					Height:  tall,
				},
			),
			title,
			gogui.Label(callStatus(c), status),
			bottom,
		},
	})
}

func (cv *View) callButtons(vw int, c call.Call) gogui.View {
	t := gogui.CurrentTheme().Cfg
	glyph := iconStyle(t.TextStyleDef.Color)
	glyph.Size = reach() * 0.6
	words := t.TextStyleDef
	height := reach() * 1.4
	calls := cv.o.Calls
	var buttons []gogui.View
	switch c.State {
	case call.Ringing:
		buttons = []gogui.View{
			cv.o.UI.Toolkit.WideKey(
				"call-decline",
				gogui.IconClose,
				glyph,
				height,
				style.Partial,
				func(gogui.EventCtx) { calls.Hangup() },
			),
		}
		if c.Video {
			buttons = append(
				buttons,
				cv.o.UI.Toolkit.WideKey(
					"call-answer-audio",
					gogui.IconPhone,
					glyph,
					height,
					style.Partial,
					func(gogui.EventCtx) { calls.AnswerAudio() },
				),
				cv.o.UI.Toolkit.WideKey(
					"call-answer-video",
					gogui.IconVideo,
					glyph,
					height,
					style.Chosen,
					func(gogui.EventCtx) { calls.Answer() },
				),
			)
		} else {
			buttons = append(
				buttons,
				cv.o.UI.Toolkit.WideKey(
					"call-answer",
					gogui.IconPhone,
					glyph,
					height,
					style.Chosen,
					func(gogui.EventCtx) { calls.Answer() },
				),
			)
		}
	case call.Calling:
		buttons = []gogui.View{
			cv.o.UI.Toolkit.WideKey(
				"call-cancel",
				gogui.IconClose,
				glyph,
				height,
				style.Partial,
				func(gogui.EventCtx) { calls.Hangup() },
			),
		}
	default:
		mute, state := say.T("call.mute"), style.Partial
		if c.Muted {
			mute, state = say.T("call.unmute"), style.Chosen
		}
		buttons = []gogui.View{
			cv.o.UI.Toolkit.WideKey(
				"call-mute",
				mute,
				words,
				height,
				state,
				func(gogui.EventCtx) { calls.Mute(!c.Muted) },
			),
		}
		if c.Video {
			camera, state := say.T("call.camera.off"), style.Partial
			if c.CameraOff {
				camera, state = say.T("call.camera.on"), style.Chosen
			}
			buttons = append(
				buttons,
				cv.o.UI.Toolkit.WideKey(
					"call-camera",
					camera,
					words,
					height,
					state,
					func(gogui.EventCtx) { calls.Camera(c.CameraOff) },
				),
			)
		}
		buttons = append(
			buttons,
			cv.o.UI.Toolkit.WideKey(
				"call-hangup",
				gogui.IconClose,
				glyph,
				height,
				style.Partial,
				func(gogui.EventCtx) { calls.Hangup() },
			),
		)
	}
	return gogui.Row(
		gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingLarge,
			Content: buttons,
		},
	)
}

func callStatus(c call.Call) string {
	switch c.State {
	case call.Calling:
		return say.T("call.calling")
	case call.Ringing:
		if c.Video {
			return say.T("call.incoming.video")
		}
		return say.T("call.incoming")
	}
	if c.Reconnecting {
		return say.T("call.reconnecting")
	}
	d := time.Since(c.Since).Truncate(time.Second)
	return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

func (cv *View) profileScreen(p *Profile) *navigation.Screen {
	title := p.ID
	if peer, ok := cv.o.Peers.Find(p.ID); ok {
		title = peer.Name
	}
	return &navigation.Screen{
		Title: title,
		View:  p,
		Build: func(w *gogui.Window) gogui.View { return cv.profileBody(w, p) },
	}
}

func (cv *View) profileBody(w *gogui.Window, prof *Profile) gogui.View {
	p, ok := cv.o.Peers.Find(prof.ID)
	if !ok {
		return homeview.Column([]gogui.View{gogui.Label(say.T("call.gone"), widgets.Secondary())})
	}
	t := gogui.CurrentTheme().Cfg
	vw, vh := w.WindowSize()
	tall := min(float32(vh)*0.18, float32(vw)*0.6*pictures.Aspect)
	key, svg := pictures.For(p.Board)

	addrs := make([]string, 0, len(p.Addrs))
	for _, a := range p.Addrs {
		addrs = append(addrs, a.String())
	}
	can := make([]string, 0, len(p.Caps))
	for _, c := range p.Caps {
		if say.Missing("peer.cap." + c) {
			can = append(can, c)
			continue
		}
		can = append(can, say.T("peer.cap."+c))
	}
	capable := say.T("call.no")
	if len(can) > 0 {
		capable = strings.Join(can, ", ")
	}
	details := []menu.Row{
		{Label: say.T("call.model"), Kind: menu.Plain, Value: p.Model},
		{
			Label: say.T("call.standing"),
			Kind:  menu.Plain,
			Value: say.T("peer." + p.Standing.String()),
		},
		{Label: say.T("call.address"), Kind: menu.Plain, Value: strings.Join(addrs, ", ")},
		{Label: say.T("call.version"), Kind: menu.Plain, Value: p.Version},
		{Label: say.T("call.caps"), Kind: menu.Plain, Value: capable},
		{
			Label: say.T("call.seen"),
			Kind:  menu.Plain,
			Value: time.Since(p.Seen).Round(time.Second).String(),
		},
	}
	rows := []gogui.View{
		gogui.Row(
			gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				Padding: gogui.NoPadding,
				HAlign:  gogui.HAlignCenter,
				Content: []gogui.View{
					gogui.Svg(
						gogui.SvgCfg{
							ID:      "profile-" + key,
							SvgData: svg,
							Width:   tall / pictures.Aspect,
							Height:  tall,
						},
					),
				},
			},
		),
	}
	for i, r := range details {
		rows = append(
			rows,
			cv.o.UI.Row(fmt.Sprintf("profile-%d", i), "profile", r, nil, false),
		)
	}

	if !p.Standing.Trusted() {
		return homeview.Column(rows)
	}
	glyph := iconStyle(t.TextStyleDef.Color)
	glyph.Size = reach() * 0.7
	height := reach() * 1.8
	keys := []gogui.View{
		cv.o.UI.Toolkit.WideKey(
			"profile-audio",
			gogui.IconPhone,
			glyph,
			height,
			style.Partial,
			func(gogui.EventCtx) { cv.dial(p, false) },
		),
	}
	if cv.video(p) {
		keys = append(keys, cv.o.UI.Toolkit.WideKey(
			"profile-video",
			gogui.IconVideo,
			glyph,
			height,
			style.Partial,
			func(gogui.EventCtx) { cv.dial(p, true) },
		))
	}
	rows = append(rows, gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		Padding: gogui.NewPadding(reach()*0.5, 0, 0, 0),
		Spacing: gogui.SpacingLarge,
		Content: keys,
	}))
	return homeview.Column(rows)
}
