package schema

import (
	"slices"
	"strings"

	"github.com/ygelfand/libcountertop/pkg/say"
)

type Call struct {
	Incoming    bool         `json:"incoming"`
	AutoAnswer  bool         `json:"auto_answer"`
	AutoVideo   bool         `json:"auto_video"`
	AnswerFrom  CallAudience `json:"answer_from"`
	AnswerPeers string       `json:"answer_peers"`
	PauseWake   bool         `json:"pause_wake"`
	Stream      CallStream   `json:"stream"`
	Ring        Chime        `json:"ring"`
	VideoRing   Chime        `json:"video_ring"`
	Ringback    Chime        `json:"ringback"`
	Broadcasts  bool         `json:"broadcasts"`
}

type CallAudience string

const (
	AnswerEveryone CallAudience = "everyone"
	AnswerSelected CallAudience = "selected"
)

func CallAudiences() []CallAudience { return []CallAudience{AnswerEveryone, AnswerSelected} }

func (a CallAudience) Label() string { return say.T("call.answer." + string(a)) }

func (c Call) Peers() []string { return strings.Fields(c.AnswerPeers) }

func (c Call) Answers(peer string) bool {
	if !c.AutoAnswer {
		return false
	}
	return c.AnswerFrom != AnswerSelected || slices.Contains(c.Peers(), peer)
}

type CallStream string

const (
	CallMain CallStream = "main"
	CallSub  CallStream = "sub"
)

func CallStreams() []CallStream { return []CallStream{CallMain, CallSub} }

func (s CallStream) Label() string {
	if s == CallSub {
		return say.T("call.stream.sub")
	}
	return say.T("call.stream.main")
}

func DefaultCall() Call {
	return Call{
		Incoming:   true,
		PauseWake:  true,
		Stream:     CallMain,
		AnswerFrom: AnswerEveryone,
		Ring:       ChimeRing,
		VideoRing:  ChimeRing,
		Ringback:   ChimeRingback,
		Broadcasts: true,
	}
}
