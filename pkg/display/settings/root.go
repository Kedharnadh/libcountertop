package settings

import (
	"fmt"

	gogui "github.com/go-gui-org/go-gui/gui"
	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
)

// Section is an optional settings page supplied by a device capability.
type Section struct {
	Label, Glyph string
	Page         func() shell.View
}
type RootOptions struct {
	Network, Theme, Assistants                                                                                     func() string
	MediaLevel                                                                                                     func() int
	Version                                                                                                        string
	Push                                                                                                           func(shell.View)
	NetworkPage, FeaturesPage, DisplayPage, HomePage, VolumePage, AssistantsPage, SystemPage, DebugPage, AboutPage func() shell.View
	Sections                                                                                                       func() []Section
}

// RootPage composes the common settings navigation. Pages are created only when opened.
func RootPage(o RootOptions) *shell.Page {
	return &shell.Page{Title: say.T("settings.title"), Build: func() ([]widget.Row, []func(int)) {
		rows := []widget.Row{}
		acts := []func(int){}
		add := func(glyph, label, value string, page func() shell.View) {
			rows = append(rows, widget.Row{Glyph: glyph, Label: say.T(label), Kind: widget.Chevron, Value: value})
			acts = append(acts, func(int) { o.Push(page()) })
		}
		add(gogui.IconGlobe, "settings.network", o.Network(), o.NetworkPage)
		add(gogui.IconPlug, "settings.features", "", o.FeaturesPage)
		add(gogui.IconSunnyO, "settings.display", o.Theme(), o.DisplayPage)
		add(gogui.IconHome, "home.title", "", o.HomePage)
		add(gogui.IconSpeaker, "settings.volume", fmt.Sprintf("%d%%", o.MediaLevel()), o.VolumePage)
		add(gogui.IconComment, "settings.assistants", o.Assistants(), o.AssistantsPage)
		if o.Sections != nil {
			for _, s := range o.Sections() {
				add(s.Glyph, s.Label, "", s.Page)
			}
		}
		add(gogui.IconGear, "settings.system", "", o.SystemPage)
		add(gogui.IconCode, "settings.debug", "", o.DebugPage)
		add(gogui.IconInfo, "settings.about", o.Version, o.AboutPage)
		return rows, acts
	}}
}
