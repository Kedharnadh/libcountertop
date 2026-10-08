package wpa

import (
	"context"
	"fmt"
	"time"
)

const Settle = 20 * time.Second

func Switch(ctx context.Context, c Conn, ssid, passphrase string) error {
	if ssid == "" {
		return fmt.Errorf("wpa: no network named")
	}

	was, err := Status(c)
	if err != nil {
		return err
	}
	if was.SSID == ssid {
		return fmt.Errorf("wpa: already on %q", ssid)
	}

	if err := Remove(c, ssid); err != nil {
		return err
	}

	id, err := Add(c, ssid, passphrase)
	if err != nil {
		return err
	}

	if _, err := c.Cmd("enable_network", id); err != nil {
		_, _ = c.Cmd("remove_network", id)
		return err
	}
	if _, err := c.Cmd("select_network", id); err != nil {
		_, _ = c.Cmd("remove_network", id)
		return err
	}

	// select_network disables every other network.
	within, stop := context.WithTimeout(ctx, Settle)
	defer stop()

	if _, err := Wait(within, c, ssid); err != nil {
		return back(c, id, was, err)
	}

	_, err = c.Cmd("save_config")
	return err
}

func back(c Conn, id string, was State, why error) error {
	_, _ = c.Cmd("remove_network", id)
	_, _ = c.Cmd("enable_network", "all")
	_, _ = c.Cmd("reassociate")

	if was.SSID == "" {
		return fmt.Errorf("wpa: did not join %w", why)
	}
	return fmt.Errorf("wpa: did not join, back on %q: %w", was.SSID, why)
}
