package wpa

import (
	"crypto/pbkdf2"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type Conn interface {
	Cmd(words ...string) (string, error)
}

// The supplicant reports key_mgmt "NONE IEEE8021X WPA-EAP WPA-PSK": no SAE.
type Security int

const (
	Open Security = iota
	PSK
	SAE
	Enterprise
)

func (s Security) String() string {
	switch s {
	case Open:
		return "open"
	case PSK:
		return "WPA2"
	case SAE:
		return "WPA3"
	}
	return "enterprise"
}

func (s Security) Supported() bool { return s == Open || s == PSK }

// An access point advertising both PSK and SAE is in WPA2/WPA3 transition mode.
func SecurityOf(flags string) Security {
	switch {
	case strings.Contains(flags, "EAP"):
		return Enterprise
	case strings.Contains(flags, "PSK"):
		return PSK
	case strings.Contains(flags, "SAE"):
		return SAE
	}
	return Open
}

type Network struct {
	SSID     string
	Signal   int
	Flags    string
	Security Security
}

func (n Network) String() string {
	if !n.Security.Supported() {
		return fmt.Sprintf("%s (%d dBm, %s — not supported)", n.SSID, n.Signal, n.Security)
	}
	return fmt.Sprintf("%s (%d dBm, %s)", n.SSID, n.Signal, n.Security)
}

func ParseScan(out string) map[string]Network {
	best := map[string]Network{}
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(fields) < 5 || fields[0] == "bssid / frequency / signal level / flags / ssid" {
			continue
		}

		ssid := fields[4]
		signal, err := strconv.Atoi(fields[2])
		if ssid == "" || err != nil {
			continue
		}
		if seen, ok := best[ssid]; !ok || signal > seen.Signal {
			best[ssid] = Network{
				SSID:     ssid,
				Signal:   signal,
				Flags:    fields[3],
				Security: SecurityOf(fields[3]),
			}
		}
	}
	return best
}

// list_networks: a header line, then one tab-separated row per network.
func ParseNetworks(out string) []string {
	var names []string
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(fields) < 2 || fields[0] == "network id / ssid / bssid / flags" {
			continue
		}
		if _, err := strconv.Atoi(fields[0]); err == nil && fields[1] != "" {
			names = append(names, fields[1])
		}
	}
	return names
}

func IDs(out, ssid string) []string {
	var ids []string
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(fields) < 2 || fields[1] != ssid {
			continue
		}
		if _, err := strconv.Atoi(fields[0]); err != nil {
			continue
		}
		ids = append(ids, fields[0])
	}
	return ids
}

type State struct {
	ID      string
	SSID    string
	State   string
	Address string
}

func (s State) Associated() bool { return s.State == "COMPLETED" }

// COMPLETED still names the previous network for a moment after select_network.
func (s State) Joined(ssid string) bool {
	if !s.Associated() {
		return false
	}
	return ssid == "" || s.SSID == ssid
}

func (s State) String() string {
	if s.SSID == "" {
		return s.State
	}
	return fmt.Sprintf("%s, %s, %s", s.SSID, strings.ToLower(s.State), s.Address)
}

func ParseStatus(out string) State {
	var s State
	for line := range strings.SplitSeq(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "id":
			s.ID = value
		case "ssid":
			s.SSID = value
		case "wpa_state":
			s.State = value
		case "ip_address":
			s.Address = value
		}
	}
	return s
}

// PSK = PBKDF2-SHA1(passphrase, ssid, 4096, 32); 64 hex characters are already a PSK.
func Key(ssid, passphrase string) (string, error) {
	if len(passphrase) == 64 && isHex(passphrase) {
		return passphrase, nil
	}
	if len(passphrase) < 8 || len(passphrase) > 63 {
		return "", fmt.Errorf(
			"wpa: a WPA passphrase is 8 to 63 characters, this one is %d",
			len(passphrase),
		)
	}

	psk, err := pbkdf2.Key(sha1.New, passphrase, []byte(ssid), 4096, 32)
	if err != nil {
		return "", fmt.Errorf("wpa: deriving the key: %w", err)
	}
	return hex.EncodeToString(psk), nil
}

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}

// wpa_cli prints the interface it selected before the answer.
func LastLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	return lines[len(lines)-1]
}
