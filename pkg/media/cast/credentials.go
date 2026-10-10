package cast

import (
	"cmp"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"slices"
	"time"
)

type Credentials struct {
	Peer          tls.Certificate
	Device        []byte
	Intermediates [][]byte

	SHA1, SHA256 []byte

	NotBefore, NotAfter time.Time

	Remote bool
}

func (c *Credentials) Signature(h Hash) []byte {
	if h == SHA256 {
		return c.SHA256
	}
	return c.SHA1
}

const PeerLife = 48 * time.Hour

type Authority struct {
	device       *rsa.PrivateKey
	deviceDER    []byte
	intermediate []byte
}

type savedAuthority struct {
	Device       []byte `json:"device_key"`
	DeviceCert   []byte `json:"device_cert"`
	Intermediate []byte `json:"intermediate_cert"`
}

func (a *Authority) Marshal() ([]byte, error) {
	return json.Marshal(savedAuthority{
		Device:       x509.MarshalPKCS1PrivateKey(a.device),
		DeviceCert:   a.deviceDER,
		Intermediate: a.intermediate,
	})
}

func LoadAuthority(data []byte, name string) (*Authority, error) {
	var s savedAuthority
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("cast: reading the kept authority: %w", err)
	}
	key, err := x509.ParsePKCS1PrivateKey(s.Device)
	if err != nil {
		return nil, fmt.Errorf("cast: reading the kept device key: %w", err)
	}
	cert, err := x509.ParseCertificate(s.DeviceCert)
	if err != nil {
		return nil, fmt.Errorf("cast: reading the kept device certificate: %w", err)
	}
	if cert.Subject.CommonName != name {
		return nil, fmt.Errorf(
			"cast: the kept authority is for %q, not %q",
			cert.Subject.CommonName,
			name,
		)
	}
	if !key.PublicKey.Equal(cert.PublicKey) {
		return nil, fmt.Errorf("cast: the kept device key does not match its certificate")
	}
	if _, err := x509.ParseCertificate(s.Intermediate); err != nil {
		return nil, fmt.Errorf("cast: reading the kept intermediate: %w", err)
	}
	return &Authority{device: key, deviceDER: s.DeviceCert, intermediate: s.Intermediate}, nil
}

func NewAuthority(name string) (*Authority, error) {
	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)

	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("cast: making the root key: %w", err)
	}
	root := &x509.Certificate{
		Subject:               pkix.Name{CommonName: name + " root"},
		NotBefore:             from,
		NotAfter:              until,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	rootDER, err := sign(root, root, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, err
	}
	root, _ = x509.ParseCertificate(rootDER)

	icaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("cast: making the intermediate key: %w", err)
	}
	ica := &x509.Certificate{
		Subject:               pkix.Name{CommonName: name + " intermediate"},
		NotBefore:             from,
		NotAfter:              until,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	icaDER, err := sign(ica, root, &icaKey.PublicKey, rootKey)
	if err != nil {
		return nil, err
	}
	ica, _ = x509.ParseCertificate(icaDER)

	deviceKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("cast: making the device key: %w", err)
	}
	device := &x509.Certificate{
		Subject:   pkix.Name{CommonName: name},
		NotBefore: from,
		NotAfter:  until,
		KeyUsage:  x509.KeyUsageDigitalSignature,
	}
	deviceDER, err := sign(device, ica, &deviceKey.PublicKey, icaKey)
	if err != nil {
		return nil, err
	}

	return &Authority{device: deviceKey, deviceDER: deviceDER, intermediate: icaDER}, nil
}

func (a *Authority) Issue(name string, now time.Time) (*Credentials, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("cast: making the peer key: %w", err)
	}

	from := now.Add(-time.Hour)
	peer := &x509.Certificate{
		Subject:     pkix.Name{CommonName: name},
		NotBefore:   from,
		NotAfter:    from.Add(PeerLife),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	device, _ := x509.ParseCertificate(a.deviceDER)
	der, err := sign(peer, device, &key.PublicKey, a.device)
	if err != nil {
		return nil, err
	}

	s1, s256, err := signatures(a.device, der)
	if err != nil {
		return nil, err
	}

	leaf, _ := x509.ParseCertificate(der)
	return &Credentials{
		Peer:          tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf},
		Device:        a.deviceDER,
		Intermediates: [][]byte{a.intermediate},
		SHA1:          s1,
		SHA256:        s256,
		NotBefore:     peer.NotBefore,
		NotAfter:      peer.NotAfter,
	}, nil
}

func sign(template, parent *x509.Certificate, pub any, key crypto.Signer) ([]byte, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("cast: making a serial: %w", err)
	}
	template.SerialNumber = serial

	der, err := x509.CreateCertificate(rand.Reader, template, parent, pub, key)
	if err != nil {
		return nil, fmt.Errorf("cast: making %s: %w", template.Subject.CommonName, err)
	}
	return der, nil
}

func signatures(key *rsa.PrivateKey, peer []byte) ([]byte, []byte, error) {
	d1 := sha1.Sum(peer)
	s1, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA1, d1[:])
	if err != nil {
		return nil, nil, fmt.Errorf("cast: signing with sha1: %w", err)
	}
	d256 := sha256.Sum256(peer)
	s256, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, d256[:])
	if err != nil {
		return nil, nil, fmt.Errorf("cast: signing with sha256: %w", err)
	}
	return s1, s256, nil
}

// Certificates, keys and signatures are base64 DER; times are Unix seconds.
type Oracle struct {
	Public       string `json:"public"`
	Private      string `json:"private"`
	Device       string `json:"device"`
	Intermediate string `json:"intermediate"`
	SHA1         string `json:"sha1"`
	SHA256       string `json:"sha256"`
	NotBefore    int64  `json:"notBefore"`
	NotAfter     int64  `json:"notAfter"`
	Now          int64  `json:"now"`
}

func ParseOracle(body []byte) (*Credentials, time.Time, error) {
	var o Oracle
	if err := json.Unmarshal(body, &o); err != nil {
		return nil, time.Time{}, fmt.Errorf("cast: the oracle's answer: %w", err)
	}

	raw := map[string][]byte{}
	for name, v := range map[string]string{
		"public": o.Public, "private": o.Private, "device": o.Device,
		"intermediate": o.Intermediate, "sha1": o.SHA1, "sha256": o.SHA256,
	} {
		if v == "" {
			return nil, time.Time{}, fmt.Errorf("cast: the oracle's answer has no %s", name)
		}
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, time.Time{}, fmt.Errorf("cast: the oracle's %s: %w", name, err)
		}
		raw[name] = b
	}

	key, err := privateKey(raw["private"])
	if err != nil {
		return nil, time.Time{}, err
	}
	leaf, err := x509.ParseCertificate(raw["public"])
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("cast: the oracle's public certificate: %w", err)
	}

	return &Credentials{
		Peer: tls.Certificate{
			Certificate: [][]byte{raw["public"]},
			PrivateKey:  key,
			Leaf:        leaf,
		},
		Device:        raw["device"],
		Intermediates: [][]byte{raw["intermediate"]},
		SHA1:          raw["sha1"],
		SHA256:        raw["sha256"],
		NotBefore:     time.Unix(o.NotBefore, 0),
		NotAfter:      time.Unix(o.NotAfter, 0),
		Remote:        true,
	}, time.Unix(o.Now, 0), nil
}

// Certificates and keys are PEM; signatures are base64 over the peer certificate's DER.
type Bundle struct {
	Device       string       `json:"cpu"`
	Intermediate string       `json:"ica"`
	Peers        []BundlePeer `json:"certs"`
}

// NotBefore and NotAfter are Unix seconds, filled from the certificate when the bundle is first read.
type BundlePeer struct {
	Public    string `json:"pu"`
	Private   string `json:"pr"`
	SHA1      string `json:"sig_sha1"`
	SHA256    string `json:"sig_sha256"`
	NotBefore int64  `json:"notBefore,omitempty"`
	NotAfter  int64  `json:"notAfter,omitempty"`
}

func ReadBundle(body []byte) (*Bundle, error) {
	var b Bundle
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, fmt.Errorf("cast: the bundle: %w", err)
	}
	if _, err := pemBlock(b.Device, "device certificate"); err != nil {
		return nil, err
	}
	if _, err := pemBlock(b.Intermediate, "intermediate"); err != nil {
		return nil, err
	}
	if len(b.Peers) == 0 {
		return nil, fmt.Errorf("cast: the bundle has no peer certificates")
	}
	for i := range b.Peers {
		p := &b.Peers[i]
		if p.NotAfter != 0 {
			continue
		}
		leaf, err := p.leaf()
		if err != nil {
			return nil, fmt.Errorf("cast: the bundle's peer %d: %w", i, err)
		}
		p.NotBefore, p.NotAfter = leaf.NotBefore.Unix(), leaf.NotAfter.Unix()
	}
	slices.SortFunc(
		b.Peers,
		func(x, y BundlePeer) int { return cmp.Compare(x.NotBefore, y.NotBefore) },
	)
	return &b, nil
}

func (b *Bundle) Credentials(now time.Time) (current *Credentials, next time.Time, err error) {
	at := now.Unix()
	pick := -1
	for i, p := range b.Peers {
		switch {
		case at < p.NotBefore:
			if next.IsZero() {
				next = time.Unix(p.NotBefore, 0)
			}
		case at < p.NotAfter:
			if pick < 0 || p.NotAfter > b.Peers[pick].NotAfter {
				pick = i
			}
		}
	}
	if pick < 0 {
		return nil, next, nil
	}
	device, _ := pemBlock(b.Device, "device certificate")
	ica, _ := pemBlock(b.Intermediate, "intermediate")
	c, err := b.Peers[pick].credentials(device, ica)
	if err != nil {
		return nil, next, fmt.Errorf("cast: the bundle's peer %d: %w", pick, err)
	}
	return c, next, nil
}

func (p BundlePeer) leaf() (*x509.Certificate, error) {
	der, err := pemBlock(p.Public, "certificate")
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("the certificate: %w", err)
	}
	return leaf, nil
}

func (p BundlePeer) credentials(device, ica []byte) (*Credentials, error) {
	leaf, err := p.leaf()
	if err != nil {
		return nil, err
	}
	der := leaf.Raw
	keyDER, err := pemBlock(p.Private, "private key")
	if err != nil {
		return nil, err
	}
	key, err := privateKey(keyDER)
	if err != nil {
		return nil, err
	}
	s1, err := signature(p.SHA1, "sha1")
	if err != nil {
		return nil, err
	}
	s256, err := signature(p.SHA256, "sha256")
	if err != nil {
		return nil, err
	}
	return &Credentials{
		Peer:          tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf},
		Device:        device,
		Intermediates: [][]byte{ica},
		SHA1:          s1,
		SHA256:        s256,
		NotBefore:     leaf.NotBefore,
		NotAfter:      leaf.NotAfter,
		Remote:        true,
	}, nil
}

func signature(text, hash string) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("no %s signature", hash)
	}
	b, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("the %s signature: %w", hash, err)
	}
	return b, nil
}

func pemBlock(text, what string) ([]byte, error) {
	block, _ := pem.Decode([]byte(text))
	if block == nil {
		return nil, fmt.Errorf("cast: the bundle's %s is not PEM", what)
	}
	return block.Bytes, nil
}

func privateKey(der []byte) (crypto.Signer, error) {
	if k, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return k, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		return nil, fmt.Errorf(
			"cast: the oracle's private key is neither PKCS#1 nor PKCS#8: %w",
			err,
		)
	}
	signer, ok := k.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("cast: the oracle's private key is a %T", k)
	}
	return signer, nil
}
