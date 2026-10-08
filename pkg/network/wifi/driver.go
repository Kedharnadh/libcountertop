package wifi

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const appear = 15 * time.Second

func Loaded(iface string) bool {
	_, err := os.Stat("/sys/class/net/" + iface)
	return err == nil
}

func (r *Radio) load() error {
	iface := r.opts.Interface
	if Loaded(iface) {
		return nil
	}
	if r.opts.Load != nil {
		if err := r.opts.Load(); err != nil {
			return err
		}
	}

	deadline := time.Now().Add(appear)
	for {
		if Loaded(iface) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not appear within %s", iface, appear)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func ReadMAC(iface string) (string, error) {
	b, err := os.ReadFile("/sys/class/net/" + iface + "/address")
	if err != nil {
		return "", fmt.Errorf("reading the wlan address: %w", err)
	}
	mac := strings.TrimSpace(string(b))
	if mac == "" || mac == "00:00:00:00:00:00" {
		return "", fmt.Errorf("the wlan address reads %q", mac)
	}
	return mac, nil
}
