package schema

// Visual is the live audio visual the device draws.
type Visual struct {
	// Kind names one of the visuals in internal/ui/visual.
	Kind string `json:"kind"`

	Label string `json:"label"`

	MaxFPS int `json:"max_fps"`

	Seed int `json:"seed"`
}

var MaxFPSSteps = []int{10, 15, 20, 30, 45, 60}

const (
	DefaultVisual = "orb"
	DefaultLabel  = ""
	DefaultMaxFPS = 60
)

func DefaultVisualSettings() Visual {
	return Visual{Kind: DefaultVisual, Label: DefaultLabel, MaxFPS: DefaultMaxFPS}
}
