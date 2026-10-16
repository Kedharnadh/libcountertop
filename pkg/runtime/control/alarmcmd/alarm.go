package alarmcmd

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/libcountertop/pkg/assistant/alarm"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const alarmUsage = "alarm SECONDS [sunrise|ember|white] [silent] | alarm snooze | alarm stop"

func Command(alarms func() *alarm.Alarms) *cobra.Command {
	return &cobra.Command{
		Use:   alarmUsage,
		Short: "Rehearse an alarm: wake light for SECONDS, then ring, with or without sound",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return rehearse(alarms(), args)
		},
	}
}

func rehearse(a *alarm.Alarms, args []string) error {
	switch args[0] {
	case "snooze":
		if !a.Snooze() {
			return fmt.Errorf("no alarm is ringing")
		}
		return nil
	case "stop":
		if !a.Stop() {
			return fmt.Errorf("no alarm is ringing")
		}
		return nil
	}
	seconds, err := strconv.Atoi(args[0])
	if err != nil || seconds < 0 {
		return fmt.Errorf("usage: %s", alarmUsage)
	}
	var look schema.WakeLook
	silent := false
	for _, arg := range args[1:] {
		switch {
		case arg == "silent":
			silent = true
		case slices.Contains(schema.WakeLooks(), schema.WakeLook(arg)):
			look = schema.WakeLook(arg)
		default:
			return fmt.Errorf("usage: %s", alarmUsage)
		}
	}
	a.Rehearse(time.Duration(seconds)*time.Second, look, silent)
	return nil
}
