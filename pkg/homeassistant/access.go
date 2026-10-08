package homeassistant

import (
	"errors"
)

var (
	ErrNotConnected = errors.New("homeassistant: not connected")
	ErrNotAllowed   = errors.New(
		"homeassistant: actions are not allowed for this device; enable them in the ESPHome integration's options",
	)
)

type Access int

const (
	Unknown Access = iota
	Allowed
	Refused
)

func (a Access) String() string {
	switch a {
	case Allowed:
		return "allowed"
	case Refused:
		return "refused"
	}
	return "unknown"
}
