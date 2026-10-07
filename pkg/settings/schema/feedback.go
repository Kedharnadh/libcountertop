package schema

type Feedback struct {
	Chime Chime `json:"chime"`
}

const DefaultChime = ChimeChirp

func DefaultFeedback() Feedback { return Feedback{Chime: DefaultChime} }
