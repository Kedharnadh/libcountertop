package cast

import (
	"encoding/json"
	"fmt"
)

const (
	NSConnection = "urn:x-cast:com.google.cast.tp.connection"
	NSHeartbeat  = "urn:x-cast:com.google.cast.tp.heartbeat"
	NSReceiver   = "urn:x-cast:com.google.cast.receiver"
	NSMedia      = "urn:x-cast:com.google.cast.media"
	NSDeviceAuth = "urn:x-cast:com.google.cast.tp.deviceauth"
)

const (
	TypeConnect    = "CONNECT"
	TypeClose      = "CLOSE"
	TypePing       = "PING"
	TypePong       = "PONG"
	TypeGetStatus  = "GET_STATUS"
	TypeStatus     = "RECEIVER_STATUS"
	TypeLaunch     = "LAUNCH"
	TypeStop       = "STOP"
	TypeLaunchFail = "LAUNCH_ERROR"
	TypeInvalid    = "INVALID_REQUEST"
)

const (
	TypeLoad        = "LOAD"
	TypePlay        = "PLAY"
	TypePause       = "PAUSE"
	TypeSeek        = "SEEK"
	TypeStopMedia   = "STOP"
	TypeMediaStatus = "MEDIA_STATUS"
	TypeSetVolume   = "SET_VOLUME"
	TypeQueueNext   = "QUEUE_NEXT"
	TypeQueuePrev   = "QUEUE_PREV"
	TypeQueueUpdate = "QUEUE_UPDATE"
	TypeLoadFailed  = "LOAD_FAILED"
)

type Header struct {
	Type string `json:"type"`

	// Zero means unsolicited.
	RequestID int `json:"requestId"`
}

func Kind(payload string) (Header, error) {
	var h Header
	if err := json.Unmarshal([]byte(payload), &h); err != nil {
		return Header{}, fmt.Errorf("cast: reading a payload: %w", err)
	}
	if h.Type == "" {
		return Header{}, fmt.Errorf("cast: a payload with no type")
	}
	return h, nil
}

func payload(v any) string {
	out, err := json.Marshal(v)
	if err != nil {
		return `{"type":"INVALID_REQUEST","reason":"INVALID_PARAMS"}`
	}
	return string(out)
}

type Volume struct {
	// Level is 0 to 1.
	Level *float64 `json:"level,omitempty"`
	Muted *bool    `json:"muted,omitempty"`

	StepInterval float64 `json:"stepInterval,omitempty"`

	// "attenuation", "fixed" or "master".
	ControlType string `json:"controlType,omitempty"`
}

type Application struct {
	AppID          string `json:"appId"`
	UniversalAppID string `json:"universalAppId"`
	AppType        string `json:"appType"`
	IconURL        string `json:"iconUrl,omitempty"`

	LaunchedFromCloud bool `json:"launchedFromCloud"`

	SessionID   string `json:"sessionId"`
	TransportID string `json:"transportId"`

	DisplayName string `json:"displayName"`
	StatusText  string `json:"statusText"`

	Namespaces []Namespace `json:"namespaces"`

	IsIdleScreen bool `json:"isIdleScreen"`
}

type Namespace struct {
	Name string `json:"name"`
}

type ReceiverStatus struct {
	Applications []Application `json:"applications,omitempty"`

	Volume Volume `json:"volume"`

	IsActiveInput bool `json:"isActiveInput"`
	IsStandBy     bool `json:"isStandBy"`
}

type statusEnvelope struct {
	Header
	Status ReceiverStatus `json:"status"`
}

func Status(request int, s ReceiverStatus) string {
	return payload(statusEnvelope{
		Header: Header{Type: TypeStatus, RequestID: request},
		Status: s,
	})
}

func Pong() string { return payload(Header{Type: TypePong}) }

func Ping() string { return payload(Header{Type: TypePing}) }

const (
	ReasonInvalidParams  = "INVALID_PARAMS"
	ReasonNotSupported   = "NOT_SUPPORTED"
	ReasonUnknownAppID   = "UNKNOWN_APP_ID"
	ReasonInvalidCommand = "INVALID_COMMAND"
)

type refusal struct {
	Header
	Reason string `json:"reason"`
}

func Invalid(request int, reason string) string {
	return payload(refusal{
		Header: Header{Type: TypeInvalid, RequestID: request},
		Reason: reason,
	})
}

func LaunchError(request int, reason string) string {
	return payload(refusal{
		Header: Header{Type: TypeLaunchFail, RequestID: request},
		Reason: reason,
	})
}

type LaunchRequest struct {
	Header
	AppID string `json:"appId"`
}

type StopRequest struct {
	Header
	SessionID string `json:"sessionId"`
}

type SetVolumeRequest struct {
	Header
	Volume Volume `json:"volume"`
}

func ParseLaunch(p string) (LaunchRequest, error) {
	var r LaunchRequest
	if err := json.Unmarshal([]byte(p), &r); err != nil {
		return r, fmt.Errorf("cast: reading a launch: %w", err)
	}
	if r.AppID == "" {
		return r, fmt.Errorf("cast: a launch naming no application")
	}
	return r, nil
}

func ParseStop(p string) (StopRequest, error) {
	var r StopRequest
	if err := json.Unmarshal([]byte(p), &r); err != nil {
		return r, fmt.Errorf("cast: reading a stop: %w", err)
	}
	return r, nil
}

func ParseSetVolume(p string) (SetVolumeRequest, error) {
	var r SetVolumeRequest
	if err := json.Unmarshal([]byte(p), &r); err != nil {
		return r, fmt.Errorf("cast: reading a volume: %w", err)
	}
	if r.Volume.Level == nil && r.Volume.Muted == nil {
		return r, fmt.Errorf("cast: a volume request setting neither level nor mute")
	}
	if l := r.Volume.Level; l != nil && (*l < 0 || *l > 1) {
		return r, fmt.Errorf("cast: a volume of %v, which is not between nought and one", *l)
	}
	return r, nil
}

func Connect() string { return payload(Header{Type: TypeConnect}) }
func Close() string   { return payload(Header{Type: TypeClose}) }
