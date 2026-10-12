package schema

type Voice struct {
	Commands bool `json:"commands"`
}

func DefaultVoice() Voice { return Voice{} }
