package schema

// Cast is whether the device answers as a Chromecast.
type Cast struct {
	// Receiver is whether it listens and advertises itself.
	Receiver bool `json:"receiver"`

	// Oracle is where device credentials come from. Empty is credentials made on the device.
	Oracle string `json:"oracle,omitempty"`

	// YouTube is this device's lounge identity, made the first time it is needed.
	YouTube YouTube `json:"youtube,omitzero"`

	Prime Prime `json:"prime,omitzero"`
}

type Prime struct {
	Persist   bool `json:"persist,omitempty"`
	SkipIntro bool `json:"skipIntro,omitempty"`
}

// YouTube is the device id a lounge screen binds with and the screen id YouTube issued per theme.
type YouTube struct {
	Device string `json:"device,omitempty"`
	Music  string `json:"music,omitempty"`
	Video  string `json:"video,omitempty"`

	Skip []string `json:"skip"`

	OnDemand bool `json:"onDemand,omitempty"`

	LiveDelay int `json:"liveDelay"`
}

const (
	LiveDelayLeast = 4
	LiveDelayMost  = 60
)

// Off by default. This one puts the device on the network under a name anyone in the house can
// cast to, which is not something to turn on for somebody.
func DefaultCast() Cast {
	return Cast{Receiver: false, YouTube: YouTube{Skip: []string{"sponsor"}, LiveDelay: 10}}
}
