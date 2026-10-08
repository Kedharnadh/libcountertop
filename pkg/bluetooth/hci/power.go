package hci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// bt_power drives only the chip's enable line.

const settle = 100 * time.Millisecond

func Power(on bool, directory string) (changed bool, err error) {
	at, err := killSwitch(directory)
	if err != nil {
		return false, err
	}

	// rfkill: soft is userspace's, hard is a physical switch.
	if hard, err := reads(filepath.Join(at, "hard")); err == nil && hard == "1" {
		return false, fmt.Errorf("ble: the radio is blocked by something physical")
	}

	was, err := reads(filepath.Join(at, "soft"))
	if err != nil {
		return false, err
	}

	want := "1"
	if on {
		want = "0"
	}
	if was == want {
		return false, nil
	}

	if err := os.WriteFile(filepath.Join(at, "soft"), []byte(want), 0o644); err != nil {
		return false, fmt.Errorf("ble: %s the radio: %w", turning(on), err)
	}
	if on {
		time.Sleep(settle)
	}
	return true, nil
}

func turning(on bool) string {
	if on {
		return "unblocking"
	}
	return "blocking"
}

// rfkill indices follow driver registration order.
func killSwitch(directory string) (string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("ble: no rfkill: %w", err)
	}

	for _, e := range entries {
		at := filepath.Join(directory, e.Name())
		if kind, err := reads(filepath.Join(at, "type")); err == nil && kind == "bluetooth" {
			return at, nil
		}
	}
	return "", fmt.Errorf("ble: nothing in %s is a bluetooth radio", directory)
}

func reads(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
