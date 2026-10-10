package schema

type Cast struct {
	Receiver bool `json:"receiver"`

	Oracle string `json:"oracle,omitempty"`
	Bundle string `json:"bundle,omitempty"`

	YouTube YouTube `json:"youtube,omitzero"`

	Prime Prime `json:"prime,omitzero"`
}

type Prime struct {
	Persist   bool `json:"persist,omitempty"`
	SkipIntro bool `json:"skipIntro,omitempty"`
}

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

func DefaultCast() Cast {
	return Cast{Receiver: false, YouTube: YouTube{Skip: []string{"sponsor"}, LiveDelay: 10}}
}
