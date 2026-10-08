package schema

type API struct {
	Adopted bool `json:"adopted"`
}

func DefaultAPI() API { return API{} }
