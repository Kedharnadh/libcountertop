package schema

// Time remembers the server's zone and an optional local IANA override.
type Time struct {
	Home   string `json:"zone"`
	Chosen string `json:"chosen"`
}
