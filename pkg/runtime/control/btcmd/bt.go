package btcmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/libcountertop/pkg/bluetooth/a2dp"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/avrcp"
	"github.com/ygelfand/libcountertop/pkg/runtime/control"
)

type Sink interface {
	Looking()
	Look(class uint16)
	Hello()
	Thumbnail(handle string)
	Art(psm uint16)
	Browsing()
	Queue()
	Dump(within time.Duration) (string, error)
	Attributes(ids ...uint32)
}

func Command(sink Sink, logTag string) *cobra.Command {
	return control.BluetoothCommand(Run(sink, logTag))
}

func Run(sink Sink, logTag string) func([]string) (string, error) {
	logcat := "adb logcat -s " + logTag
	return func(args []string) (string, error) {
		if len(args) == 0 {
			return "", fmt.Errorf("bt: %q is not one of its commands", "")
		}

		switch args[0] {
		case "trace":
			within := 30 * time.Second
			if len(args) > 1 {
				ms, err := strconv.Atoi(args[1])
				if err != nil || ms < 0 {
					return "", fmt.Errorf("bt: %q is not a number of milliseconds", args[1])
				}
				within = time.Duration(ms) * time.Millisecond
			}

			a2dp.Trace(within)
			return fmt.Sprintf("tracing frames for %s, then: %s\n", within, logcat), nil

		case "look":
			a2dp.Trace(10 * time.Second)

			if len(args) < 2 {
				sink.Looking()
				return "asked for a lookup channel, then: bt look 110c\n", nil
			}

			class, err := strconv.ParseUint(strings.TrimPrefix(args[1], "0x"), 16, 16)
			if err != nil {
				return "", fmt.Errorf("bt: %q is not a service class", args[1])
			}

			sink.Look(uint16(class))
			return fmt.Sprintf("looking up %#04x, watch the trace\n", class), nil

		case "art":
			a2dp.Trace(10 * time.Second)

			if len(args) < 2 {
				return "", fmt.Errorf("bt: art PSM | art hello | art HANDLE")
			}

			switch {
			case args[1] == "hello":
				sink.Hello()
				return "connecting to the image service, then: bt art HANDLE\n", nil

			// A BIP image handle is seven digits and a channel is hex.
			case len(args[1]) == 7 && isDigits(args[1]):
				sink.Thumbnail(args[1])
				return fmt.Sprintf("asked for image %s, watch the trace\n", args[1]), nil
			}

			psm, err := strconv.ParseUint(strings.TrimPrefix(args[1], "0x"), 16, 16)
			if err != nil {
				return "", fmt.Errorf(
					"bt: %q is neither a channel nor a seven digit handle",
					args[1],
				)
			}

			sink.Art(uint16(psm))
			return fmt.Sprintf(
				"asked for the image channel on %#04x, then: bt art hello\n",
				psm,
			), nil

		case "browse":
			a2dp.Trace(10 * time.Second)
			sink.Browsing()
			return "asked for the browsing channel, then: " + logcat + "\n", nil

		case "queue":
			a2dp.Trace(10 * time.Second)
			sink.Queue()
			return "asked the addressed player what is lined up, then: " + logcat + "\n", nil

		case "dump":
			within := 5 * time.Second
			if len(args) > 1 {
				secs, err := strconv.Atoi(args[1])
				if err != nil || secs <= 0 {
					return "", fmt.Errorf("bt: %q is not a number of seconds", args[1])
				}
				within = time.Duration(secs) * time.Second
			}
			return sink.Dump(within)

		case "attrs":
			ids := []uint32{avrcp.AttrCoverArt}
			if len(args) > 1 {
				ids = nil
				for _, a := range args[1:] {
					n, err := strconv.ParseUint(a, 0, 32)
					if err != nil {
						return "", fmt.Errorf("bt: %q is not an attribute id", a)
					}
					ids = append(ids, uint32(n))
				}
			}

			a2dp.Trace(3 * time.Second)
			sink.Attributes(ids...)
			return fmt.Sprintf("asked for %v, watch the trace\n", ids), nil
		}
		return "", fmt.Errorf("bt: %q is not one of its commands", args[0])
	}
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
