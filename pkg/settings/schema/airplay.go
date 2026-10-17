package schema

type AirPlay struct {
	Enabled bool `json:"enabled"`
}

func DefaultAirPlay() AirPlay { return AirPlay{} }
