package control

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGroupedPathsPassLiteralArguments(t *testing.T) {
	for _, args := range [][]string{{"input", "tap", "-1", "20"}} {
		var got []string
		factory := func() *cobra.Command {
			root := NewRoot()
			root.AddCommand(
				Does(
					&cobra.Command{Use: "tap X Y", Args: cobra.ExactArgs(2)},
					func(in []string) error { got = in; return nil },
				),
			)
			Organize(root)
			return root
		}
		if _, err := Execute(t.Context(), factory, args); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, []string{"-1", "20"}) {
			t.Errorf("Execute(%q) arguments = %q", args, got)
		}
		out, err := Execute(t.Context(), factory, []string{"input", "tap", "--help"})
		if err != nil || !strings.Contains(out, "tap X Y") {
			t.Errorf("grouped help = %q, %v", out, err)
		}
	}
}

func TestWaitIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	factory := func() *cobra.Command {
		root := NewRoot()
		root.AddCommand(WaitCommand())
		Organize(root)
		return root
	}
	if _, err := Execute(
		ctx,
		factory,
		[]string{"device", "wait", "60000"},
	); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Errorf("canceled wait returned %v", err)
	}
}

func TestOldCommandPathsAreRejected(t *testing.T) {
	factory := func() *cobra.Command {
		root := NewRoot()
		noop := func([]string) error { return nil }
		root.AddCommand(
			Does(&cobra.Command{Use: "tap X Y", Args: cobra.ExactArgs(2)}, noop),
			Does(&cobra.Command{Use: "shot PATH", Args: cobra.ExactArgs(1)}, noop),
			Does(&cobra.Command{Use: "settings [NAME [VALUE]]", Args: cobra.MaximumNArgs(2)}, noop),
			Does(
				&cobra.Command{
					Use:  "record [raw|echo] SECONDS [PATH]",
					Args: cobra.RangeArgs(1, 3),
				},
				noop,
			),
			WaitCommand(),
			BluetoothCommand(func([]string) (string, error) { return "", nil }),
		)
		root.AddCommand(DeviceCommands(func() DeviceInfo { return DeviceInfo{} })...)
		Organize(root)
		return root
	}
	for _, args := range [][]string{
		{"tap", "1", "2"}, {"screen", "tap", "1", "2"}, {"input", "drag", "1", "2", "3", "4"},
		{"shot", "/tmp/panel.png"}, {"version"}, {"state"}, {"wait", "0"},
		{"set", "name", "value"}, {"bt", "browse"}, {"record", "2"},
		{"echo", "2"}, {"audio", "echo", "2"},
	} {
		if _, err := Execute(t.Context(), factory, args); err == nil {
			t.Errorf("old command path still accepted: %q", args)
		}
	}
}

func TestCallToggleRejectsUnknownValueBeforeAdapter(t *testing.T) {
	called := false
	factory := func() *cobra.Command {
		root := NewRoot()
		root.AddCommand(CallCommand(CallActions{Mute: func(bool) { called = true }}))
		Organize(root)
		return root
	}
	if _, err := Execute(
		t.Context(),
		factory,
		[]string{"call", "mute", "maybe"},
	); err == nil ||
		called {
		t.Errorf("invalid mute reached adapter=%t, error=%v", called, err)
	}
}

func TestArgumentFramesPreserveSpacesAndSemicolons(t *testing.T) {
	want := []string{"settings", "title", "Kitchen display; still one value"}
	commands, err := argumentSequence(want)
	if err != nil || len(commands) != 1 {
		t.Fatalf("argumentSequence = %q, %v", commands, err)
	}
	got, err := arguments(commands[0])
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("argument round trip = %q, %v, want %q", got, err, want)
	}
	commands, err = argumentSequence([]string{"device", "state", ";", "device", "version"})
	if err != nil || len(commands) != 2 {
		t.Errorf("standalone semicolon sequence = %q, %v", commands, err)
	}
}

func TestRecordingModesUseOnePathAcrossAdapters(t *testing.T) {
	for _, separateEcho := range []bool{false, true} {
		var got []string
		factory := func() *cobra.Command {
			root := NewRoot()
			record := &cobra.Command{
				Use:  "record [raw|echo] SECONDS [PATH]",
				RunE: func(_ *cobra.Command, args []string) error { got = args; return nil },
			}
			root.AddCommand(record)
			if separateEcho {
				record.Use = "record SECONDS [PATH]"
				root.AddCommand(
					&cobra.Command{
						Use:  "echo SECONDS PATH",
						RunE: func(_ *cobra.Command, args []string) error { got = args; return nil },
					},
				)
			}
			Organize(root)
			return root
		}
		if _, err := Execute(
			t.Context(),
			factory,
			[]string{"audio", "record", "echo", "2"},
		); err != nil {
			t.Fatal(err)
		}
		want := []string{"echo", "2"}
		if separateEcho {
			want = []string{"2", "/data/local/tmp/echo.wav"}
		}
		if !slices.Equal(got, want) {
			t.Errorf("separateEcho=%t: arguments=%q, want %q", separateEcho, got, want)
		}
	}
}
