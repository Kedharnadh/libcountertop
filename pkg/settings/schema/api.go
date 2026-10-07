package schema

// API records whether Home Assistant has adopted the device.
type API struct {
	Adopted bool `json:"adopted"`
}

func DefaultAPI() API { return API{} }
