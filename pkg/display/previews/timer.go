package previews

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Timer schema.TimerStyle

func (t Timer) View(_ *gogui.Window, width, height float32) gogui.View {
	s := newStage(width, height)
	wide := width - 2*s.pad()
	switch schema.TimerStyle(t) {
	case schema.TimerBanner:
		return s.screen(gogui.ContainerCfg{
			HAlign:  gogui.HAlignCenter,
			Content: []gogui.View{s.strip(wide, height/5)},
		})
	case schema.TimerDial:
		return s.screen(gogui.ContainerCfg{
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{ring(height*0.6, s.accent)},
		})
	case schema.TimerFull:
		return s.screen(gogui.ContainerCfg{
			Color:   s.accent,
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{ring(height*0.6, s.back)},
		})
	case schema.TimerFace:
		h := height * 0.42
		digits := make([]gogui.View, 4)
		for i := range digits {
			digits[i] = block(width/14, h*0.55, h/24, s.accent)
		}
		return s.screen(gogui.ContainerCfg{
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{block(width*0.7, h, h/8, s.panel, digits...)},
		})
	case schema.TimerCorner:
		h := height / 7
		return s.screen(gogui.ContainerCfg{
			HAlign:  gogui.HAlignEnd,
			Content: []gogui.View{block(width/3, h, h/2, s.panel, disc(h/2, s.accent))},
		})
	}
	return s.screen(gogui.ContainerCfg{
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignBottom,
		Content: []gogui.View{s.strip(wide, height/5)},
	})
}

func (s stage) strip(width, height float32) gogui.View {
	return gogui.Column(gogui.ContainerCfg{
		Width:   width,
		Height:  height,
		Sizing:  gogui.FixedFixed,
		Color:   s.panel,
		Radius:  gogui.RadiusPx(height / 5),
		Padding: gogui.PadAll(height / 5),
		VAlign:  gogui.VAlignBottom,
		Content: []gogui.View{block(width*0.55, height/7, height/14, s.accent)},
	})
}
