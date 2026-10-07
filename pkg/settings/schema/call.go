package schema

import "github.com/ygelfand/libcountertop/pkg/say"

type Call struct {
	Incoming   bool       `json:"incoming"`
	AutoAnswer bool       `json:"auto_answer"`
	AutoVideo  bool       `json:"auto_video"`
	PauseWake  bool       `json:"pause_wake"`
	Stream     CallStream `json:"stream"`
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

func DefaultCall() Call { return Call{Incoming: true, PauseWake: true, Stream: CallMain} }
