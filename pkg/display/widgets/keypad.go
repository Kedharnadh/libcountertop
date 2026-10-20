package widgets

import (
	"sync"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
)

type Keypad struct {
	ID     string
	Redraw func()

	mu    sync.Mutex
	typed string
}

func (k *Keypad) Typed() string {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.typed
}

func (k *Keypad) Reset() {
	k.mu.Lock()
	k.typed = ""
	k.mu.Unlock()
}

func (k *Keypad) Press(d string, length int, full func(string)) {
	k.mu.Lock()
	k.typed += d
	code := ""
	if len(k.typed) >= length {
		code, k.typed = k.typed, ""
	}
	k.mu.Unlock()
	if code != "" {
		full(code)
	}
	k.redraw()
}

func (k *Keypad) Back() {
	k.mu.Lock()
	if k.typed != "" {
		k.typed = k.typed[:len(k.typed)-1]
	}
	k.mu.Unlock()
	k.redraw()
}

func (k *Keypad) Clear() {
	k.Reset()
	k.redraw()
}

func (k *Keypad) redraw() {
	if k.Redraw != nil {
		k.Redraw()
	}
}

func (k *Keypad) View(
	kit *Toolkit,
	wide, vh float32,
	length int,
	full func(string),
) gogui.View {
	key := min(style.Reach()*1.2, vh*0.07)
	glyph := gogui.CurrentTheme().Cfg.TextStyleDef
	glyph.Size = key * 0.5
	icon := gogui.CurrentTheme().TextStyleIconMedium
	icon.Size = key * 0.45
	digit := func(d string) gogui.View {
		return kit.WideKey(k.ID+"-key-"+d, d, glyph, key, style.Partial, func(gogui.EventCtx) {
			k.Press(d, length, full)
		})
	}
	row := func(keys ...gogui.View) gogui.View {
		return gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingSmall,
			Content: keys,
		})
	}
	return gogui.Column(gogui.ContainerCfg{
		Width:   min(wide, key*6),
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingSmall,
		Content: []gogui.View{
			row(digit("1"), digit("2"), digit("3")),
			row(digit("4"), digit("5"), digit("6")),
			row(digit("7"), digit("8"), digit("9")),
			row(
				kit.WideKey(k.ID+"-key-clear", gogui.IconClose, icon, key, style.Partial,
					func(gogui.EventCtx) { k.Clear() }),
				digit("0"),
				kit.WideKey(k.ID+"-key-back", gogui.IconArrowLeft, icon, key, style.Partial,
					func(gogui.EventCtx) { k.Back() }),
			),
		},
	})
}
