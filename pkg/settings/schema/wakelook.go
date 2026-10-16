package schema

import "github.com/ygelfand/libcountertop/pkg/say"

type WakeLook string

const (
	WakeSunrise WakeLook = "sunrise"
	WakeWhite   WakeLook = "white"
	WakeEmber   WakeLook = "ember"
)

func WakeLooks() []WakeLook { return []WakeLook{WakeSunrise, WakeWhite, WakeEmber} }

func (w WakeLook) Label() string {
	switch w {
	case WakeSunrise:
		return say.T("wake.sunrise")
	case WakeWhite:
		return say.T("wake.white")
	case WakeEmber:
		return say.T("wake.ember")
	}
	return string(w)
}
