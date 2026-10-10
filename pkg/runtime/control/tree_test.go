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
			t := NewTree()
			t.Add(
				InputTask,
				Does(
					&cobra.Command{Use: "tap X Y", Args: cobra.ExactArgs(2)},
					func(in []string) error { got = in; return nil },
				),
			)
			return t.Root()
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
		t := NewTree()
		t.Add(DeviceTask, WaitCommand())
		return t.Root()
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
		t := NewTree()
		noop := func([]string) error { return nil }
		t.Add(InputTask, Does(&cobra.Command{Use: "tap X Y", Args: cobra.ExactArgs(2)}, noop))
		t.Add(DisplayTask, Does(&cobra.Command{Use: "shot PATH", Args: cobra.ExactArgs(1)}, noop))
		t.Add(
			AudioTask,
			Does(&cobra.Command{Use: "record SECONDS [PATH]", Args: cobra.RangeArgs(1, 2)}, noop),
		)
		t.Add(DeviceTask, WaitCommand())
		t.Add(DeviceTask, DeviceCommands(func() DeviceInfo { return DeviceInfo{} })...)
		t.Top(
			Does(&cobra.Command{Use: "settings [NAME [VALUE]]", Args: cobra.MaximumNArgs(2)}, noop),
			BluetoothCommand(func([]string) (string, error) { return "", nil }),
		)
		return t.Root()
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
		t := NewTree()
		t.Top(CallCommand(CallActions{Mute: func(bool) { called = true }}))
		return t.Root()
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

func TestGroupHelpListsItsCommands(t *testing.T) {
	tree := NewTree()
	tree.Add(DeviceTask, WaitCommand())
	tree.Add(DeviceTask, DeviceCommands(func() DeviceInfo { return DeviceInfo{} })...)
	root := tree.Root()
	device, _, err := root.Find([]string{"device"})
	if err != nil {
		t.Fatal(err)
	}
	if device.Short != "device: profile, state, version, wait" {
		t.Errorf("device group summary = %q", device.Short)
	}
	if _, _, err := root.Find([]string{"input"}); err == nil {
		t.Error("a task with no commands was added")
	}
}
