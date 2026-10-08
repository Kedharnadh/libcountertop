package schema

type Camera struct {
	Settings map[string]string `json:"settings,omitempty"`
}

func DefaultCamera() Camera { return Camera{Settings: map[string]string{}} }

func (c Camera) Set(name string) (string, bool) {
	v, ok := c.Settings[name]
	return v, ok
}
