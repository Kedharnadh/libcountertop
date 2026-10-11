package schema

type Feedback struct {
	Chime   Chime `json:"chime"`
	Failure Chime `json:"failure"`
	Cancel  Chime `json:"cancel"`
	Mute    Chime `json:"mute"`
	Unmute  Chime `json:"unmute"`
	Hold    Chime `json:"hold"`
}

const DefaultChime = ChimeChirp

func DefaultFeedback() Feedback {
	return Feedback{Chime: DefaultChime, Failure: ChimeTrouble, Cancel: ChimeCancel,
		Mute: ChimeMuted, Unmute: ChimeUnmuted, Hold: ChimeMuteHold}
}
