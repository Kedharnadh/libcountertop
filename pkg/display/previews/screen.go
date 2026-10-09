package previews

import gogui "github.com/go-gui-org/go-gui/gui"

type stage struct {
	width, height       float32
	back, panel, accent gogui.Color
}

func newStage(width, height float32) stage {
	t := gogui.CurrentTheme().Cfg
	return stage{
		width:  width,
		height: height,
		back:   t.ColorBackground,
		panel:  t.ColorPanel,
		accent: t.ColorAccent,
	}
}

func (s stage) pad() float32 { return s.width / 20 }

func (s stage) screen(cfg gogui.ContainerCfg) gogui.View {
	if !cfg.Color.IsSet() {
		cfg.Color = s.back
	}
	cfg.Width, cfg.Height, cfg.Sizing = s.width, s.height, gogui.FixedFixed
	cfg.Radius, cfg.Clip = gogui.RadiusMedium, true
	cfg.Padding = gogui.PadAll(s.pad())
	return gogui.Column(cfg)
}

func block(width, height, radius float32, fill gogui.Color, content ...gogui.View) gogui.View {
	return gogui.Row(gogui.ContainerCfg{
		Width:   width,
		Height:  height,
		Sizing:  gogui.FixedFixed,
		Color:   fill,
		Radius:  gogui.RadiusPx(radius),
		Padding: gogui.PadVH(0, height/3),
		Spacing: gogui.SpacingPx(height / 3),
		VAlign:  gogui.VAlignMiddle,
		Content: content,
	})
}

func disc(side float32, fill gogui.Color) gogui.View {
	return block(side, side, side/2, fill)
}

func ring(side float32, ink gogui.Color) gogui.View {
	return gogui.Column(gogui.ContainerCfg{
		Width:       side,
		Height:      side,
		Sizing:      gogui.FixedFixed,
		Radius:      gogui.RadiusPx(side / 2),
		ColorBorder: ink,
		SizeBorder:  gogui.BorderPx(side / 7),
		Padding:     gogui.NoPadding,
	})
}
