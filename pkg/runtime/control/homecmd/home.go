package homecmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/libcountertop/pkg/homeassistant"
	harness "github.com/ygelfand/libcountertop/pkg/runtime/control"
)

const haWait = 10 * time.Second

func Commands(client Client) []*cobra.Command {
	ha := harness.Group("ha", "Ask Home Assistant", "")
	ha.AddCommand(
		harness.Says(&cobra.Command{
			Use:   "access",
			Short: "Whether Home Assistant answers actions from this device",
			Args:  cobra.NoArgs,
		}, func([]string) (string, error) {
			return client.Access().String(), nil
		}),
		harness.Says(&cobra.Command{
			Use:   "entities [DOMAIN...]",
			Short: "The entities Home Assistant has, in every domain or the ones named",
		}, func(args []string) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), haWait)
			defer cancel()
			got, err := client.Entities(ctx, homeassistant.Filter{Domains: args})
			if err != nil {
				return "", err
			}
			var out strings.Builder
			for _, e := range got {
				fmt.Fprintf(&out, "%s\t%s\t%s\t%s\n", e.ID, e.State, e.Area, e.Name)
			}
			return out.String(), nil
		}),
		harness.Says(&cobra.Command{
			Use:   "labels",
			Short: "The labels Home Assistant has",
			Args:  cobra.NoArgs,
		}, func([]string) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), haWait)
			defer cancel()
			got, err := client.Labels(ctx)
			if err != nil {
				return "", err
			}
			var out strings.Builder
			for _, l := range got {
				fmt.Fprintf(&out, "%s\t%s\n", l.ID, l.Name)
			}
			return out.String(), nil
		}),
		harness.Says(&cobra.Command{
			Use:   "light on|off ENTITY",
			Short: "Turn a light on or off",
			Args:  cobra.ExactArgs(2),
		}, func(args []string) (string, error) {
			action := map[string]string{"on": "light.turn_on", "off": "light.turn_off"}[args[0]]
			if action == "" {
				return "", fmt.Errorf("on or off, not %q", args[0])
			}
			ctx, cancel := context.WithTimeout(context.Background(), haWait)
			defer cancel()
			if err := client.Call(
				ctx,
				action,
				map[string]string{"entity_id": args[1]},
			); err != nil {
				return "", err
			}
			return args[1] + " " + args[0], nil
		}),
	)
	return []*cobra.Command{ha}
}

type Client interface {
	Access() homeassistant.Access
	Entities(context.Context, homeassistant.Filter) ([]homeassistant.Entity, error)
	Labels(context.Context) ([]homeassistant.Label, error)
	Call(context.Context, string, map[string]string) error
}
