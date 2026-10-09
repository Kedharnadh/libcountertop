package schema

type Visual struct {
	Kind string `json:"kind"`

	Label string `json:"label"`

	MaxFPS int `json:"max_fps"`

	Seed int `json:"seed"`
}

var MaxFPSSteps = []int{10, 15, 20, 30, 45, 60}

const (
	DefaultVisual = "orb"
	DefaultMaxFPS = 60
)

func DefaultVisualSettings() Visual {
	return Visual{Kind: DefaultVisual, MaxFPS: DefaultMaxFPS}
}
