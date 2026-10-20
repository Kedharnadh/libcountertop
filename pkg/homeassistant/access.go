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

const (
	Ping      = "echolocal.ping"
	Protocol  = 1
	pingReply = "{{ response | tojson }}"
)

type Failed struct{ Message string }

func (f *Failed) Error() string { return "homeassistant: " + f.Message }

type Integration int

const (
	IntegrationUnknown Integration = iota
	IntegrationMissing
	IntegrationOutdated
	IntegrationReady
)

func (i Integration) String() string {
	switch i {
	case IntegrationMissing:
		return "missing"
	case IntegrationOutdated:
		return "outdated"
	case IntegrationReady:
		return "ready"
	}
	return "unknown"
}

func (a Access) String() string {
	switch a {
	case Allowed:
		return "allowed"
	case Refused:
		return "refused"
	}
	return "unknown"
}
