package schema

type Alerts struct {
	Info    Chime `json:"info"`
	Success Chime `json:"success"`
	Warning Chime `json:"warning"`
	Alert   Chime `json:"alert"`
	Timer   Chime `json:"timer"`
	RingFor Delay `json:"ring_for"`
}

func DefaultAlerts() Alerts {
	return Alerts{
		Info:    ChimeDing,
		Success: ChimeChirp,
		Warning: ChimeRise,
		Alert:   ChimeTrouble,
		Timer:   ChimeTimer,
		RingFor: Delay15m,
	}
}

func (a Alerts) Chime(t Tone) Chime {
	switch t {
	case ToneSuccess:
		return a.Success
	case ToneWarning:
		return a.Warning
	case ToneAlert:
		return a.Alert
	}
	return a.Info
}

func RingDelays() []Delay { return []Delay{Delay1m, Delay5m, Delay15m, Delay30m} }
