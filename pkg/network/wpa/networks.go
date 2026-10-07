package wpa

import (
	"context"
	"fmt"
	"strings"
)

type Connection interface {
	Cmd(string) (string, error)
	Close() error
}
type NetworksClient struct{ Dial func() (Connection, error) }

// socket is the control interface as the protocol wants it: one command, one reply.
//
// Commands arrive lowercase, as wpa_cli takes them, and go out uppercase, as the control interface
// takes them. wpa_cli is the thing that uppercases them, and there is no wpa_cli here.
type Commands struct{ Connection }

func (s Commands) Cmd(words ...string) (string, error) {
	if len(words) == 0 {
		return "", fmt.Errorf("wifi: no command")
	}

	cmd := strings.ToUpper(words[0])
	if len(words) > 1 {
		cmd += " " + strings.Join(words[1:], " ")
	}

	out, err := s.Connection.Cmd(cmd)
	if err != nil {
		return out, err
	}
	if strings.HasPrefix(out, "FAIL") {
		return out, fmt.Errorf("wifi: %s said %q", words[0], strings.TrimSpace(out))
	}
	return out, nil
}

// Scan is what the radio can see, strongest first, one entry per name.
//
// A scan takes seconds and holds the control socket for its whole run, so the connection is this
// call's own rather than one shared with anything else.
func (n NetworksClient) Scan(ctx context.Context) ([]Network, error) {
	c, err := n.Dial()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	return Scanned(ctx, Commands{c})
}

// Networks is the names the supplicant already has configured.
func (n NetworksClient) Networks() ([]string, error) {
	c, err := n.Dial()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	return Configured(Commands{c})
}

// Join moves the device to another network, and puts it back on the one it was on when the new one
// does not associate.
//
// Switching drops the connection that asked for the switch, which is the whole difficulty: a wrong
// passphrase typed on the panel would otherwise strand the device with no way in. Switch keeps
// the old network configured and selects it again on failure.
func (n NetworksClient) Join(ctx context.Context, ssid, passphrase string) error {
	c, err := n.Dial()
	if err != nil {
		return err
	}
	defer c.Close()

	return Switch(ctx, Commands{c}, ssid, passphrase)
}

// Forget removes a configured network. The one in use is refused: forgetting it would drop the
// connection with nothing to fall back to, which is the failure Join exists to avoid.
func (n NetworksClient) Forget(ssid string) error {
	c, err := n.Dial()
	if err != nil {
		return err
	}
	defer c.Close()

	now, err := Status(Commands{c})
	if err != nil {
		return err
	}
	if now.SSID == ssid {
		return fmt.Errorf("wifi: %q is the network in use", ssid)
	}
	return Remove(Commands{c}, ssid)
}
