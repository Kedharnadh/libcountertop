package screencmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/go-esphome-device/api"

	"github.com/ygelfand/libcountertop/pkg/assistant/timer"
)

const harnessTimer = "ctl"

func Timer(timers func() *timer.Timers) *cobra.Command {
	return &cobra.Command{
		Use:   "timer SECONDS [NAME] | timer ring [silent] | timer stop | timer cancel",
		Short: "Count a timer down on the panel, ring it with or without sound, stop the ringing, or cancel it",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return countdown(timers(), args)
		},
	}
}

func countdown(ts *timer.Timers, args []string) error {
	switch strings.Join(args, " ") {
	case "ring":
		ts.Event(esphome.TimerEvent{
			Type:    api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_FINISHED,
			TimerID: harnessTimer,
		})
		return nil
	case "ring silent":
		ts.RingSilently()
		return nil
	case "stop":
		ts.Stop()
		return nil
	case "cancel":
		ts.Event(esphome.TimerEvent{
			Type:    api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_CANCELLED,
			TimerID: harnessTimer,
		})
		return nil
	}
	seconds, err := strconv.Atoi(args[0])
	if err != nil || seconds <= 0 {
		return fmt.Errorf(
			"usage: timer SECONDS [NAME] | timer ring [silent] | timer stop | timer cancel",
		)
	}
	ts.Event(esphome.TimerEvent{
		Type:         api.VoiceAssistantTimerEvent_VOICE_ASSISTANT_TIMER_STARTED,
		TimerID:      harnessTimer,
		Name:         strings.Join(args[1:], " "),
		TotalSeconds: uint32(seconds),
		SecondsLeft:  uint32(seconds),
		IsActive:     true,
	})
	return nil
}
