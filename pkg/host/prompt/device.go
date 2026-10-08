package prompt

import (
	"context"
	"fmt"
	"io"

	"github.com/ygelfand/libcountertop/pkg/host/adb"
)

func Serial(
	ctx context.Context,
	out io.Writer,
	serial string,
	list func() ([]adb.Info, error),
	other string,
) (string, error) {
	if serial != "" {
		return serial, nil
	}
	if !IsTerminal() {
		return "", nil
	}

	devices, err := list()
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", adb.ErrUnreachable
	}

	chosen, err := Choose(ctx, out, "Select a device", devices,
		func(d adb.Info) string { return fmt.Sprintf("%s  %s", d, d.Serial) }, other)
	if err != nil {
		return "", err
	}
	if chosen.Serial == "" {
		return "", adb.ErrUnreachable
	}
	return chosen.Serial, nil
}
