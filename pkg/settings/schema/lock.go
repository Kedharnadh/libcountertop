package schema

import "time"

type Lock struct {
	PIN string `json:"pin"`

	Failures int       `json:"failures,omitempty"`
	Until    time.Time `json:"until,omitzero"`
}
