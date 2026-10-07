package schema

import "encoding/json"

// Network contains common connectivity settings.
type Network struct {
	Verify  bool   `json:"verify"`
	Address string `json:"address,omitempty"`
}

func DefaultNetwork() Network { return Network{Verify: true} }

// MigrateNetwork translates the former inverted TLS setting when no canonical setting exists.
func MigrateNetwork(data []byte, network *Network) error {
	var old struct {
		Network struct {
			Verify *bool `json:"verify"`
		} `json:"network"`
		Diag struct {
			InsecureTLS *bool `json:"insecure_tls"`
		} `json:"diag"`
	}
	if err := json.Unmarshal(data, &old); err != nil {
		return err
	}
	if old.Network.Verify == nil && old.Diag.InsecureTLS != nil {
		network.Verify = !*old.Diag.InsecureTLS
	}
	return nil
}
