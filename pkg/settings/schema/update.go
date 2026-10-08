package schema

type Update struct {
	Channel     string `json:"channel"`
	LastVersion string `json:"last_version"`
}

const DefaultChannel = "stable"

func DefaultUpdate() Update { return Update{Channel: DefaultChannel} }
