package control

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

type Labelled interface{ Label() string }

func Labels[T Labelled](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = v.Label()
	}
	return out
}

func Choose[T Labelled](values []T, use func(T)) func(string) error {
	return func(s string) error {
		for _, v := range values {
			if strings.EqualFold(v.Label(), s) || strings.EqualFold(fmt.Sprint(v), s) {
				use(v)
				return nil
			}
		}
		return fmt.Errorf("want one of %s", strings.Join(Labels(values), ", "))
	}
}

func Words(use func(string)) func(string) error {
	return func(s string) error {
		use(s)
		return nil
	}
}

func OrNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func Secret(s string) string {
	if s == "" {
		return "(none)"
	}
	return "(set)"
}

func OnOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

func Toggle(use func(bool)) func(string) error {
	return func(s string) error {
		switch strings.ToLower(s) {
		case "on", "true", "yes", "1":
			use(true)
		case "off", "false", "no", "0":
			use(false)
		default:
			return fmt.Errorf("want on or off")
		}
		return nil
	}
}

func Number(lo, hi int, use func(int)) func(string) error {
	return func(s string) error {
		v, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("want a number from %d to %d", lo, hi)
		}
		if v < lo || v > hi {
			return fmt.Errorf("want %d to %d, not %d", lo, hi, v)
		}
		use(v)
		return nil
	}
}

func LogLevel(threshold *slog.LevelVar, args []string) (string, error) {
	if len(args) == 1 {
		var l slog.Level
		if err := l.UnmarshalText([]byte(args[0])); err != nil {
			return "", err
		}
		threshold.Set(l)
		slog.Log(context.Background(), max(l, slog.LevelInfo), "log level", "level", l)
	}
	return strings.ToLower(threshold.Level().String()), nil
}
