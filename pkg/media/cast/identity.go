package cast

import (
	"crypto/tls"
	"errors"
)

var ErrNoCredentials = errors.New("cast: no credentials yet")

func TLS(current func() *Credentials) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.NoClientCert,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			c := current()
			if c == nil {
				return nil, ErrNoCredentials
			}
			return &c.Peer, nil
		},
	}
}
