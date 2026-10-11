package schema

import "slices"

type Apps struct {
	Order []string `json:"order"`
	Dock  []string `json:"dock"`
}

const (
	AppPlayer = "player"
	AppCalls  = "calls"
)

func DefaultApps() Apps { return Apps{Dock: []string{AppPlayer, AppCalls}} }

func (a Apps) Clone() Apps {
	a.Order = slices.Clone(a.Order)
	a.Dock = slices.Clone(a.Dock)
	return a
}
