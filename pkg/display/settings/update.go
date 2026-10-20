package settings

import (
	"slices"
	"strings"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/system/firmware"
)

type Updates interface {
	Version() string
	Channels() []string
	Channel() string
	Choose(channel string)
	CheckLater()
	InstallLater()
	Offer() firmware.Offer
	Catalog() firmware.Catalog
	ListLater()
	InstallBuildLater(version string)
}

func (p *BasicPages) Update() *shell.Page {
	u := p.options.Updates
	return &shell.Page{Title: say.T("update.title"), Build: func() ([]menu.Row, []func(int)) {
		offer := u.Offer()
		channels := u.Channels()
		labels := make([]string, len(channels))
		for i, c := range channels {
			labels[i] = say.T("update.channel." + c)
		}
		checking := offer.Finding == firmware.Checking

		rows := []menu.Row{
			{Label: say.T("update.running"), Kind: menu.Plain, Value: u.Version()},
			{
				Label:   say.T("update.channel"),
				Kind:    menu.Segments,
				Options: labels,
				Level:   max(slices.Index(channels, u.Channel()), 0),
			},
			{
				Glyph: gogui.IconSync,
				Label: say.T("update.check"),
				Value: UpdateSays(offer),
				Dim:   checking,
			},
		}
		acts := []func(int){
			nil,
			func(i int) { u.Choose(channels[i]) },
			func(int) {
				if !checking {
					u.CheckLater()
				}
			},
		}
		if offer.Finding == firmware.Available {
			rows = append(rows, menu.Row{
				Glyph: gogui.IconDownload,
				Label: say.F("update.install", map[string]any{"Version": offer.Release.Version}),
				Hint:  headline(offer.Release.Notes),
			})
			acts = append(acts, func(int) { u.InstallLater() })
		}
		rows = append(rows, menu.Row{Glyph: gogui.IconTimeline, Label: say.T("update.versions")})
		acts = append(acts, func(int) {
			u.ListLater()
			p.options.Shell.Push(p.Versions())
		})
		return rows, acts
	}}
}

func (p *BasicPages) Versions() *shell.Page {
	u := p.options.Updates
	return &shell.Page{Title: say.T("update.versions"), Build: func() ([]menu.Row, []func(int)) {
		catalog := u.Catalog()
		if len(catalog.Builds) == 0 {
			says := say.T("update.checking")
			switch catalog.Finding {
			case firmware.Unreachable:
				says = say.T("update.unreachable")
			case firmware.Current:
				says = say.T("update.versions.none")
			}
			return []menu.Row{{Label: says, Kind: menu.Plain}}, []func(int){nil}
		}

		running := u.Version()
		rows := make([]menu.Row, 0, len(catalog.Builds))
		acts := make([]func(int), 0, len(catalog.Builds))
		for _, b := range catalog.Builds {
			channel := say.T("update.channel.stable")
			if b.Prerelease {
				channel = say.T("update.channel.dev")
			}
			if b.Tag == running {
				rows = append(rows, menu.Row{
					Label: b.Tag,
					Hint:  channel,
					Value: say.T("update.versions.installed"),
					Kind:  menu.Plain,
				})
				acts = append(acts, nil)
				continue
			}
			rows = append(rows, menu.Row{Glyph: gogui.IconDownload, Label: b.Tag, Hint: channel})
			tag := map[string]any{"Version": b.Tag, "Running": running}
			acts = append(acts, func(int) {
				p.options.Shell.Push(ConfirmPage(
					say.F("update.versions.confirm", tag),
					say.F("update.versions.what", tag),
					func() { u.InstallBuildLater(b.Tag) },
				))
			})
		}
		return rows, acts
	}}
}

func UpdateSays(o firmware.Offer) string {
	switch o.Finding {
	case firmware.Checking:
		return say.T("update.checking")
	case firmware.Current:
		return say.T("update.current")
	case firmware.Available:
		return say.F("update.available", map[string]any{"Version": o.Release.Version})
	case firmware.Unreachable:
		return say.T("update.unreachable")
	}
	return ""
}

func (p *BasicPages) SystemSays() string {
	if p.options.Updates == nil || p.options.Updates.Offer().Finding != firmware.Available {
		return ""
	}
	return say.T("update.ready")
}

func headline(notes string) string {
	first, _, _ := strings.Cut(strings.TrimSpace(notes), "\n")
	return strings.TrimSpace(first)
}
