package widgets

import (
	gogui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/display/interaction"
	"github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

// UI owns the common controls, editing, gestures and palette for one display.
type UI struct {
	settings storage.Section[schema.Screen]
	Presses  *interaction.Presses
	Editor   *Editor
	Gestures *Gestures
	Toolkit  Toolkit
}

// NewUI resolves saved presentation settings when drawing, never during construction.
func NewUI(settings storage.Section[schema.Screen]) *UI {
	u := &UI{settings: settings}
	u.Reset()
	u.Toolkit = Toolkit{Kit: u.Kit, Pressable: func(layout func(gogui.ContainerCfg) gogui.View, cfg gogui.ContainerCfg, do func(gogui.EventCtx)) gogui.View {
		return u.Presses.Pressable(layout, cfg, do)
	}}
	return u
}

// Reset clears editing and interaction state before a display session starts.
// The UI's identity stays stable so existing renderers use the fresh state.
func (u *UI) Reset() {
	u.Presses = interaction.New()
	u.Editor = NewEditor()
	u.Gestures = &Gestures{Release: func(e gogui.EventCtx) { u.Presses.Release(e) }}
}

func (u *UI) Kit() style.Kit { return style.ByName(u.settings.Read().Style).Kit }
func (u *UI) Palette() theme.Theme {
	s := u.settings.Read()
	if p, ok := theme.ByName(style.Theme(s.Style, s.Theme)); ok {
		return p
	}
	return theme.Default()
}
func (u *UI) Grip(v gogui.View) gogui.View { return u.Gestures.Grip(v) }
func (u *UI) HoldStill(e gogui.EventCtx)   { u.Gestures.HoldStill(e) }
func (u *UI) Input(cfg gogui.InputCfg, save func()) gogui.View {
	return u.Editor.Field(u.Kit(), cfg, save)
}
func (u *UI) Field(id string, r model.Row) gogui.View { return u.Editor.Row(u.Kit(), id, r) }
func (u *UI) Row(id, page string, r model.Row, tap func(int), glyphs bool) gogui.View {
	return u.Toolkit.Row(id, page, r, tap, glyphs, RowRenderer{
		Field:   func(id string, r model.Row) gogui.View { return u.Editor.Row(u.Kit(), id, r) },
		Preview: func(page string, r model.Row) gogui.View { return RowPreview(page, r, u.Palette()) },
		Grip:    u.Grip,
	})
}
