package callview

import (
	"log/slog"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/peerboard"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/say"
)

func (cv *View) callsBoard(at ui.Rect, pal theme.Theme) gogui.View {
	return peerboard.Board{
		UI:      cv.o.UI,
		ID:      "peers",
		Empty:   say.T("call.empty"),
		Actions: cv.actions,
	}.View(at, cv.o.Peers.Peers(), pal)
}

func (cv *View) actions(p peer.Peer) []peerboard.Action {
	out := []peerboard.Action{
		{Name: "audio", Icon: gogui.IconPhone, Do: func(gogui.EventCtx) { cv.dial(p, false) }},
		{Name: "info", Icon: gogui.IconInfo, Do: func(gogui.EventCtx) { cv.open(p) }},
	}
	if cv.video(p) {
		out = append(out, peerboard.Action{
			Name: "video",
			Icon: gogui.IconVideo,
			Do:   func(gogui.EventCtx) { cv.dial(p, true) },
		})
	}
	return out
}

func (cv *View) video(p peer.Peer) bool { return p.Video() && cv.o.Video }

func (cv *View) dial(p peer.Peer, video bool) {
	if err := cv.o.Calls.Start(p, video); err != nil {
		slog.Warn("call not placed", "to", p.Name, "err", err)
	}
}
