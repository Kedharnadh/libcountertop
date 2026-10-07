package schema

// Sendspin is the device's participation in synchronized playback.
type Sendspin struct {
	Enabled bool `json:"enabled"`
}

// DefaultSendspin requires an explicit choice before opening a listening port.
func DefaultSendspin() Sendspin { return Sendspin{} }
