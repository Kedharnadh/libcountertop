package control

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Sequence reads newline-delimited scripts or semicolon-separated command-line sequences.
func Sequence(stdin io.Reader, args []string) ([]string, error) {
	if len(args) == 1 && args[0] == "-" {
		var out []string
		in := bufio.NewScanner(stdin)
		in.Buffer(make([]byte, 4096), 1<<20)
		for in.Scan() {
			line := strings.TrimSpace(in.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				out = append(out, line)
			}
		}
		return out, in.Err()
	}
	var out []string
	for _, part := range strings.Split(strings.Join(args, " "), ";") {
		if line := strings.Join(strings.Fields(part), " "); line != "" {
			out = append(out, line)
		}
	}
	return out, nil
}

// Answer reads a terminal response without discarding output preceding a failure.
func Answer(in *bufio.Scanner, out io.Writer) error {
	for in.Scan() {
		line := in.Text()
		if line == "ok" {
			return nil
		}
		if after, found := strings.CutPrefix(line, "error: "); found {
			return fmt.Errorf("%s", after)
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	if err := in.Err(); err != nil {
		return err
	}
	return fmt.Errorf("the daemon closed the connection")
}

// Send runs commands sequentially on one connection, stopping at the first failure.
func Send(ctx context.Context, socket string, commands []string, out io.Writer) error {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "unix", socket)
	if err != nil {
		return err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	in := bufio.NewScanner(conn)
	in.Buffer(make([]byte, 4096), 1<<20)
	for _, line := range commands {
		if err := conn.SetDeadline(time.Now().Add(time.Hour)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(conn, line); err != nil {
			return err
		}
		if err := Answer(in, out); err != nil {
			return err
		}
	}
	return nil
}

type ClientOptions struct {
	Name, Socket string
	Local        func([]string) (string, error)
}

// ClientCommand exposes the same harness client syntax for each product daemon.
func ClientCommand(options ClientOptions) *cobra.Command {
	localHelp := ""
	if options.Local != nil {
		localHelp = "\nUse -l to run supported commands locally without the daemon."
	}
	return &cobra.Command{
		Use: "ctl [command...]", Short: "Drive the running " + options.Name,
		Long: "Commands run against the daemon holding the hardware.\n" +
			"Use task groups such as device, input, display and audio; ctl alone shows help.\n" +
			"Separate commands with ; or use a single - to read one command per stdin line.\n" +
			"A sequence stops at its first failure. --help asks the daemon for command help." + localHelp,
		SilenceUsage: true, DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && args[0] == "-l" && options.Local != nil {
				out, err := options.Local(args[1:])
				fmt.Fprint(cmd.OutOrStdout(), out)
				return err
			}
			if len(args) == 0 {
				args = []string{"help"}
			}
			commands, err := Sequence(cmd.InOrStdin(), args)
			if len(args) > 1 {
				commands, err = argumentSequence(args)
			}
			if err != nil {
				return err
			}
			if len(commands) == 0 {
				return fmt.Errorf("nothing to run")
			}
			return Send(cmd.Context(), options.Socket, commands, cmd.OutOrStdout())
		},
	}
}

// argumentSequence preserves argv boundaries, including spaces and semicolons within values.
// A standalone semicolon separates commands; a single script argument uses Sequence instead.
func argumentSequence(args []string) ([]string, error) {
	var commands []string
	var command []string
	flush := func() error {
		if len(command) == 0 {
			return nil
		}
		line, err := json.Marshal(command)
		if err != nil {
			return err
		}
		commands = append(commands, string(line))
		command = nil
		return nil
	}
	for _, arg := range args {
		if arg == ";" {
			if err := flush(); err != nil {
				return nil, err
			}
		} else {
			command = append(command, arg)
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return commands, nil
}
