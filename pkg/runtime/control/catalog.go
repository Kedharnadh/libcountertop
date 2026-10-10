package control

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/libcountertop/pkg/runtime/profiler"
)

type DeviceInfo struct {
	Version, Commit, Built, Board string
	Name, Address, Model          string
	Uptime                        time.Duration
}

func DeviceCommands(read func() DeviceInfo) []*cobra.Command {
	return []*cobra.Command{
		{
			Use: "version", Short: "Build identity and board", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				info := read()
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"version %s\ncommit  %s\nbuilt   %s\nboard   %s\n",
					info.Version,
					info.Commit,
					info.Built,
					info.Board,
				)
				return nil
			},
		},
		{
			Use: "state", Short: "Device identity and uptime", Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				info := read()
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"name    %s\naddr    %s\nmodel   %s\nuptime  %s\n",
					info.Name,
					info.Address,
					info.Model,
					info.Uptime.Round(time.Second),
				)
				return nil
			},
		},
		Says(&cobra.Command{
			Use:       "profile [on|off]",
			Short:     "Open or close the CPU and memory profiler on " + profiler.Addr,
			Args:      cobra.MaximumNArgs(1),
			ValidArgs: []string{"on", "off"},
		}, profile),
	}
}

var profiling profiler.Server

func profile(args []string) (string, error) {
	if len(args) == 1 {
		switch args[0] {
		case "on":
			if _, err := profiling.Open(); err != nil {
				return "", err
			}
		case "off":
			if err := profiling.Close(); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("want on or off, not %q", args[0])
		}
	}
	if at := profiling.Listening(); at != "" {
		return "on " + at, nil
	}
	return "off", nil
}

type CallActions struct {
	Dial, State    func([]string) (string, error)
	Answer, Hangup func()
	Mute, Camera   func(bool)
}

func CallCommand(actions CallActions) *cobra.Command {
	cmd := Group("call", "Place, answer and end calls to other devices", "")
	cmd.AddCommand(
		Says(
			&cobra.Command{
				Use:   "dial NAME|ID [video]",
				Short: "Call a discovered device",
				Args:  cobra.RangeArgs(1, 2),
			},
			func(args []string) (string, error) {
				if len(args) == 2 && args[1] != "video" {
					return "", fmt.Errorf("call mode must be video, got %q", args[1])
				}
				return actions.Dial(args)
			},
		),
		Does(
			&cobra.Command{Use: "answer", Short: "Answer the ringing call", Args: cobra.NoArgs},
			func([]string) error { actions.Answer(); return nil },
		),
		Does(
			&cobra.Command{Use: "hangup", Short: "End or decline the call", Args: cobra.NoArgs},
			func([]string) error { actions.Hangup(); return nil },
		),
		toggleCommand("mute", "Stop or resume sending the microphone", actions.Mute),
		toggleCommand("camera", "Start or stop sending call video", actions.Camera),
		Says(
			&cobra.Command{Use: "state", Short: "The call in progress", Args: cobra.NoArgs},
			actions.State,
		),
	)
	return cmd
}

func toggleCommand(name, short string, action func(bool)) *cobra.Command {
	return Does(
		&cobra.Command{Use: name + " on|off", Short: short, Args: cobra.ExactArgs(1)},
		func(args []string) error {
			switch args[0] {
			case "on":
				action(true)
			case "off":
				action(false)
			default:
				return fmt.Errorf("want on or off, got %q", args[0])
			}
			return nil
		},
	)
}

func Under(run func([]string) (string, error), verb string) func([]string) (string, error) {
	return func(args []string) (string, error) { return run(append([]string{verb}, args...)) }
}

func BluetoothCommand(run func([]string) (string, error)) *cobra.Command {
	cmd := Group("bluetooth", "Inspect the Bluetooth audio sink and connected player", "")
	cmd.AddCommand(
		Says(
			&cobra.Command{
				Use:   "trace [MILLISECONDS]",
				Short: "Log incoming frames",
				Args:  cobra.MaximumNArgs(1),
			},
			Under(run, "trace"),
		),
		Says(
			&cobra.Command{
				Use:   "attrs [ID ...]",
				Short: "Read player attributes",
				Args:  cobra.ArbitraryArgs,
			},
			Under(run, "attrs"),
		),
		Says(
			&cobra.Command{
				Use:   "dump [SECONDS]",
				Short: "Capture the audio stream",
				Args:  cobra.MaximumNArgs(1),
			},
			Under(run, "dump"),
		),
		Says(
			&cobra.Command{
				Use:   "look [CLASS]",
				Short: "Inspect offered services",
				Args:  cobra.MaximumNArgs(1),
			},
			Under(run, "look"),
		),
		Says(
			&cobra.Command{
				Use:   "art PSM|hello|HANDLE",
				Short: "Fetch cover art",
				Args:  cobra.ExactArgs(1),
			},
			Under(run, "art"),
		),
		Says(
			&cobra.Command{Use: "browse", Short: "Browse the player library", Args: cobra.NoArgs},
			Under(run, "browse"),
		),
		Says(
			&cobra.Command{
				Use:   "queue [PLAYER [COUNT]]",
				Short: "Read the player queue",
				Args:  cobra.MaximumNArgs(2),
			},
			Under(run, "queue"),
		),
	)
	return cmd
}
