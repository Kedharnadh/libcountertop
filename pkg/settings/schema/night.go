package schema

import "github.com/ygelfand/libcountertop/pkg/say"

type Night string

const (
	NightNone  Night = "none"
	NightDim   Night = "dim"
	NightClock Night = "clock"
	NightOff   Night = "off"
)

const (
	DefaultNightLux   = 5
	DefaultNightAfter = Delay5m
)

func (n Night) Label() string {
	switch n {
	case NightNone:
		return say.T("night.none")
	case NightDim:
		return say.T("night.dim")
	case NightClock:
		return say.T("night.clock")
	case NightOff:
		return say.T("night.off")
	}
	return string(n)
}

func Nights() []Night { return []Night{NightNone, NightDim, NightClock, NightOff} }

func (n Night) On() bool { return n == NightDim || n == NightClock || n == NightOff }

func (n Night) Quiet() bool { return n == NightClock || n == NightOff }

func NightDelays() []Delay { return []Delay{Delay1m, Delay5m, Delay15m, Delay30m} }
