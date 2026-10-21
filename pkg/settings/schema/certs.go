package schema

type Certs struct {
	Server     string `json:"server"`
	Cert       string `json:"cert"`
	CertAPIKey string `json:"cert_api_key"`
	KeyAPIKey  string `json:"key_api_key"`
}

func (c Certs) Configured() bool {
	return c.Server != "" && c.Cert != "" && c.CertAPIKey != "" && c.KeyAPIKey != ""
}
