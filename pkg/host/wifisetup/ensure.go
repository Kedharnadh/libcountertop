package wifisetup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/ygelfand/libcountertop/pkg/host/prompt"
	"github.com/ygelfand/libcountertop/pkg/network/wpa"
)

const joinTimeout = 45 * time.Second

var (
	styleDone   = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleFail   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styleDetail = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

type Request struct {
	SSID     string
	Password string
	WPS      bool
	Force    bool
}

func Ensure(ctx context.Context, out io.Writer, s Supplicant, r Request) error {
	ssid, password, wps := r.SSID, r.Password, r.WPS
	given := ssid != ""

	if ssid == "" && !wps && !r.Force {
		done, err := check(ctx, out, s)
		if err != nil || done {
			return err
		}
	}

	for {
		if !wps && ssid == "" {
			var err error
			if ssid, password, err = askNetwork(ctx, out, s); err != nil {
				return err
			}
		}

		state, err := join(ctx, out, s, ssid, password, wps)
		if err == nil {
			fmt.Fprintf(out, "%s %s\n", styleDone.Render("✓ connected:"), state)
			return nil
		}

		fmt.Fprintf(out, "%s %v\n", styleFail.Render("✗"), err)

		again, err := prompt.Confirm(ctx, out, "Try again?")
		if err != nil || !again {
			return err
		}

		if !given {
			ssid, password = "", ""
		}
		wps = false
	}
}

func join(
	ctx context.Context,
	out io.Writer,
	s Supplicant,
	ssid, password string,
	wps bool,
) (wpa.State, error) {
	if wps {
		fmt.Fprintf(out, "%s\n", styleDetail.Render("Press the WPS button on the router now."))
		if err := wpa.WPS(s); err != nil {
			return wpa.State{}, err
		}
		state, err := attempt(ctx, s, "")
		if err != nil {
			return state, err
		}
		return state, wpa.Only(s, state.ID)
	}

	now, err := wpa.Status(s)
	if err != nil {
		return now, err
	}
	if now.Joined(ssid) {
		return now, wpa.Only(s, now.ID)
	}
	if err := s.Join(ctx, ssid, password); err != nil {
		return now, err
	}
	return wpa.Status(s)
}

func askNetwork(ctx context.Context, out io.Writer, s Supplicant) (string, string, error) {
	fmt.Fprintf(out, "%s\n", styleDetail.Render("Scanning…"))

	networks, err := wpa.Scanned(ctx, s)
	if err != nil {
		return "", "", err
	}
	if len(networks) == 0 {
		return "", "", errors.New("no networks found; pass --ssid to join a hidden one")
	}

	chosen, err := prompt.Choose(
		ctx,
		out,
		"Which network?",
		networks,
		wpa.Network.String,
		"Other network…",
	)
	if err != nil {
		return "", "", err
	}

	if chosen.SSID == "" {
		name, err := prompt.Line(ctx, out, "Network name", "", false)
		if err != nil {
			return "", "", err
		}
		password, err := prompt.Line(ctx, out, "Passphrase for "+name, "", true)
		return name, password, err
	}

	if !chosen.Security.Supported() {
		return "", "", fmt.Errorf(
			"%s is %s, which this device's supplicant cannot join",
			chosen.SSID,
			chosen.Security,
		)
	}
	if chosen.Security == wpa.Open {
		return chosen.SSID, "", nil
	}

	password, err := prompt.Line(ctx, out, "Passphrase for "+chosen.SSID, "", true)
	return chosen.SSID, password, err
}

func check(ctx context.Context, out io.Writer, s Supplicant) (bool, error) {
	state, err := wpa.Status(s)
	if err != nil {
		return false, err
	}

	configured, err := wpa.Configured(s)
	if err != nil {
		return false, err
	}

	if len(configured) > 0 && !state.Joined("") {
		fmt.Fprintf(out, "%s\n", styleDetail.Render("Connecting…"))

		if err := wpa.Reconnect(s); err != nil {
			return false, err
		}

		settle, cancel := context.WithTimeout(ctx, joinTimeout)
		state, _ = wpa.Wait(settle, s, "")
		cancel()
	}

	if state.Joined("") {
		fmt.Fprintf(out, "%s %s\n", styleDone.Render("✓ already connected:"), state)
		return true, nil
	}
	if len(configured) == 0 {
		return false, nil
	}

	fmt.Fprintf(out, "%s %s\n", styleDetail.Render("configured but not connected:"),
		strings.Join(configured, ", "))
	fmt.Fprintf(out, "%s %s\n", styleDetail.Render("state:"), state)

	again, err := prompt.Confirm(ctx, out, "Reconfigure?")
	return !again, err
}

func attempt(ctx context.Context, s Supplicant, ssid string) (wpa.State, error) {
	ctx, cancel := context.WithTimeout(ctx, joinTimeout)
	defer cancel()

	return wpa.Wait(ctx, s, ssid)
}
