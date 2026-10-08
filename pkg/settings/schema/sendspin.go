package schema

type Sendspin struct {
	Enabled bool `json:"enabled"`
}

func DefaultSendspin() Sendspin { return Sendspin{} }
