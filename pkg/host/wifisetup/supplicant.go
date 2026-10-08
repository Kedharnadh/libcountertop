package wifisetup

import (
	"fmt"
	"strings"

	"github.com/ygelfand/libcountertop/pkg/host/adb"
	"github.com/ygelfand/libcountertop/pkg/network/wpa"
)

type Supplicant struct {
	Device  *adb.Device
	Sockets string
	Iface   string
	DHCP    string
}

func (s Supplicant) Cmd(words ...string) (string, error) {
	cmd := fmt.Sprintf("wpa_cli -p %s -i %s %s", s.Sockets, s.Iface, strings.Join(words, " "))

	out, err := s.Device.Shell(cmd)
	if err != nil {
		return out, err
	}
	if strings.Contains(out, "FAIL") {
		return out, fmt.Errorf("wifi: %s said %q", words[0], strings.TrimSpace(wpa.LastLine(out)))
	}
	return out, nil
}

func (s Supplicant) Answering() bool {
	out, err := s.Cmd("ping")
	return err == nil && strings.Contains(out, "PONG")
}

func (s Supplicant) Join(ssid, passphrase string) error {
	if err := wpa.Join(s, ssid, passphrase); err != nil {
		return err
	}
	if s.DHCP != "" {
		_ = s.Device.Setprop("ctl.start", s.DHCP)
	}
	return nil
}
