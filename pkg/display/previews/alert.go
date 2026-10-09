package previews

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Alert schema.AlertStyle

func (a Alert) View(_ *gogui.Window, width, height float32) gogui.View {
	s := newStage(width, height)
	wide := width - 2*s.pad()
	switch schema.AlertStyle(a) {
	case schema.AlertCard:
		h := height * 0.42
		return s.screen(gogui.ContainerCfg{
			Color:  gogui.Black.WithOpacity(0.5).Over(s.back),
			HAlign: gogui.HAlignCenter,
			VAlign: gogui.VAlignMiddle,
			Content: []gogui.View{
				block(width*0.7, h, h/8, s.panel, block(h/14, h*0.6, h/28, s.accent)),
			},
		})
	case schema.AlertFull:
		return s.screen(gogui.ContainerCfg{
			Color:   s.accent,
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{disc(height/4, s.back)},
		})
	case schema.AlertToast:
		h := height / 7
		return s.screen(gogui.ContainerCfg{
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignBottom,
			Content: []gogui.View{block(width/2, h, h/2, s.panel, disc(h/2, s.accent))},
		})
	}
	h := height / 5
	return s.screen(gogui.ContainerCfg{
		HAlign:  gogui.HAlignCenter,
		Content: []gogui.View{block(wide, h, h/5, s.panel, block(h/8, h*0.6, h/16, s.accent))},
	})
}
