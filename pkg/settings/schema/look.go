package schema

import "github.com/ygelfand/libcountertop/pkg/say"

type Look struct {
	Place      LookPlace `json:"place"`
	Listening  string    `json:"listening"`
	Waiting    string    `json:"waiting"`
	Responding string    `json:"responding"`
}

type LookPlace string

const (
	LookPanel LookPlace = "panel"
	LookFull  LookPlace = "full"
)

func (p LookPlace) Label() string {
	switch p {
	case LookFull:
		return say.T("look.full")
	}
	return say.T("look.panel")
}

func LookPlaces() []LookPlace { return []LookPlace{LookPanel, LookFull} }

type Stage string

const (
	StageListening  Stage = "listening"
	StageWaiting    Stage = "waiting"
	StageResponding Stage = "responding"
)

func Stages() []Stage { return []Stage{StageListening, StageWaiting, StageResponding} }

func (s Stage) Label() string { return say.T("look.stage." + string(s)) }

func (s Stage) Source() Source {
	switch s {
	case StageListening:
		return SourceMic
	case StageResponding:
		return SourceSpeaker
	}
	return SourceBoth
}

func (l Look) Kind(s Stage) string {
	switch s {
	case StageListening:
		return l.Listening
	case StageWaiting:
		return l.Waiting
	case StageResponding:
		return l.Responding
	}
	return ""
}
