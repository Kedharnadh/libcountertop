package call

import (
	"errors"
	"slices"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/actions"
	"github.com/ygelfand/libcountertop/pkg/match"
	discovery "github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/say"
)

func (c *Calls) Action(peers func() []discovery.Peer, video bool) actions.Action {
	triggers := []actions.Trigger{
		{Sentence: "call [the] {peer}"},
		{Sentence: "ring [the] {peer}"},
		{Sentence: "intercom [the] {peer}"},
	}
	if video {
		triggers = append(triggers, actions.Trigger{
			Sentence: "video call [the] {peer}",
			Args:     map[string]string{"video": "true"},
		})
	}
	return actions.Action{
		Name:  "call",
		Title: say.T("actions.call"),
		Args: []esphome.Arg{
			{Name: "peer", Type: esphome.ArgString},
			{Name: "video", Type: esphome.ArgBool},
		},
		Triggers: triggers,
		Run: func(call esphome.Call) (string, error) {
			name := call.String("peer")
			p, ok := Find(peers(), name)
			if !ok {
				return say.F("actions.call.unknown", map[string]any{"Peer": name}), nil
			}
			err := c.Start(p, video && call.Bool("video") && p.Video())
			switch {
			case errors.Is(err, ErrBusy):
				return say.T("actions.call.busy"), nil
			case err != nil:
				return say.F("actions.call.unreachable", map[string]any{"Peer": p.Name}), nil
			}
			return "", nil
		},
	}
}

func Find(peers []discovery.Peer, name string) (discovery.Peer, bool) {
	candidates := make([]match.Candidate, len(peers))
	for i, p := range peers {
		candidates[i] = match.Candidate{Key: p.ID, Name: p.Name}
	}
	id, ok := match.Best(name, candidates)
	if !ok {
		return discovery.Peer{}, false
	}
	i := slices.IndexFunc(peers, func(p discovery.Peer) bool { return p.ID == id })
	return peers[i], true
}
