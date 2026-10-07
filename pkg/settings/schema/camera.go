package schema

// Camera is what the camera has been set to, keyed by the setting names in
// internal/feature/livecam. Values are unvalidated; the table there does that.
type Camera struct {
	Settings map[string]string `json:"settings,omitempty"`
}

func DefaultCamera() Camera { return Camera{Settings: map[string]string{}} }

func (c Camera) Set(name string) (string, bool) {
	v, ok := c.Settings[name]
	return v, ok
}

// CameraWriter changes what the camera is set to.

// Set records one setting. An empty value forgets it.
// Put records several settings in one write. An empty value forgets that one.
