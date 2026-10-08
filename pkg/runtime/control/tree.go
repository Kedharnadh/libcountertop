package control

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	root := Group(
		"ctl",
		"Drive the running device",
		"Commands act on the live daemon. Input coordinates follow the viewed orientation.",
	)
	root.SilenceErrors, root.SilenceUsage = true, true
	root.CompletionOptions.DisableDefaultCmd = true
	return root
}

func Group(use, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("%s: %q is not one of its commands", cmd.Name(), args[0])
			}
			return cmd.Help()
		},
	}
}

func Says(cmd *cobra.Command, run func([]string) (string, error)) *cobra.Command {
	cmd.RunE = func(c *cobra.Command, args []string) error {
		out, err := run(args)
		if out != "" {
			fmt.Fprintln(c.OutOrStdout(), strings.TrimRight(out, "\n"))
		}
		return err
	}
	return cmd
}

func ContextSays(
	cmd *cobra.Command,
	run func(context.Context, []string) (string, error),
) *cobra.Command {
	cmd.RunE = func(c *cobra.Command, args []string) error {
		out, err := run(c.Context(), args)
		if out != "" {
			fmt.Fprintln(c.OutOrStdout(), strings.TrimRight(out, "\n"))
		}
		return err
	}
	return cmd
}

func Does(cmd *cobra.Command, run func([]string) error) *cobra.Command {
	cmd.RunE = func(_ *cobra.Command, args []string) error { return run(args) }
	return cmd
}

func Literal(cmd *cobra.Command) {
	cmd.DisableFlagParsing = true
	for _, child := range cmd.Commands() {
		Literal(child)
	}
}

var categories = map[string]string{
	"version": "device",
	"state":   "device",
	"log":     "device",
	"peers":   "device",
	"wait":    "device",
	"ready":   "device",
	"tap":     "input",
	"hold":    "input",
	"swipe":   "input",
	"button":  "input",
	"turn":    "input",
	"shot":    "display",
	"size":    "display",
	"stats":   "display",
	"fps":     "display",
	"open":    "display",
	"stack":   "display",
	"message": "display",
	"clear":   "display",
	"gltime":  "display",
	"render":  "display",
	"clip":    "display",
	"thumbs":  "display",
	"boot":    "display",
	"timer":   "display",
	"record":  "audio",
	"echo":    "audio",
	"volume":  "audio",
	"mixer":   "audio",
	"player":  "audio",
	"cancel":  "audio",
	"youtube": "media",
}

func Organize(root *cobra.Command) {
	commands := root.Commands()
	root.RemoveCommand(commands...)
	groups := make(map[string]*cobra.Command)
	var echo, record *cobra.Command
	put := func(cmd *cobra.Command) {
		name := cmd.Name()
		if name == "echo" {
			echo = cmd
			return
		}
		if name == "record" {
			record = cmd
		}
		if name == "wait" {
			cmd.RunE = wait
		}
		category := categories[name]
		if category == "" {
			root.AddCommand(cmd)
			return
		}
		parent := groups[category]
		if parent == nil {
			parent = Group(category, "", "")
			groups[category] = parent
			root.AddCommand(parent)
		}
		parent.AddCommand(cmd)
	}
	for _, cmd := range commands {
		put(cmd)
	}
	if record != nil {
		if echo != nil {
			echo.Use = "echo SECONDS [PATH]"
			echo.Args = cobra.RangeArgs(1, 2)
			run := echo.RunE
			echo.RunE = func(cmd *cobra.Command, args []string) error {
				if len(args) == 1 {
					args = append(args, "/data/local/tmp/echo.wav")
				}
				return run(cmd, args)
			}
			record.AddCommand(echo)
		} else if strings.Contains(record.Use, "raw|echo") {
			for _, mode := range []string{"raw", "echo"} {
				record.AddCommand(
					&cobra.Command{
						Use:  mode + " SECONDS [PATH]",
						Args: cobra.RangeArgs(1, 2),
						RunE: func(cmd *cobra.Command, args []string) error {
							return record.RunE(cmd, append([]string{mode}, args...))
						},
					},
				)
			}
		}
	}
	for name, group := range groups {
		var names []string
		for _, cmd := range group.Commands() {
			names = append(names, cmd.Name())
		}
		group.Short = name + ": " + strings.Join(names, ", ")
	}
	Literal(root)
}

func Execute(ctx context.Context, makeTree func() *cobra.Command, args []string) (string, error) {
	root := makeTree()
	var out strings.Builder
	root.SetOut(&out)
	root.SetErr(&out)
	if len(args) > 0 && (args[len(args)-1] == "--help" || args[len(args)-1] == "-h") {
		cmd, rest, err := root.Find(args[:len(args)-1])
		if err != nil {
			return "", err
		}
		if len(rest) > 0 && cmd.HasAvailableSubCommands() {
			return "", fmt.Errorf("%s: unknown command %q", cmd.Name(), rest[0])
		}
		err = cmd.Help()
		return out.String(), err
	}
	root.SetArgs(args)
	err := root.ExecuteContext(ctx)
	return out.String(), err
}

func WaitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "wait MILLISECONDS",
		Short: "Wait between harness steps",
		Args:  cobra.ExactArgs(1),
		RunE:  wait,
	}
}

func wait(cmd *cobra.Command, args []string) error {
	ms, err := strconv.Atoi(args[0])
	if err != nil || ms < 0 || ms > 60_000 {
		return fmt.Errorf("%q is not a number of milliseconds up to a minute", args[0])
	}
	timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-cmd.Context().Done():
		return cmd.Context().Err()
	case <-timer.C:
		return nil
	}
}
