package widgets

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func clockText(at time.Time, h24 bool) string {
	if h24 {
		return at.Format("15:04")
	}
	return at.Format("3:04 PM")
}

func lastRun(epoch float64, now time.Time, h24, compact bool) string {
	if epoch <= 0 {
		return say.T("tessera.never")
	}
	when := time.Unix(int64(epoch), 0).In(now.Location())
	clock := clockText(when, h24)
	y1, d1 := when.Year(), when.YearDay()
	y2, d2 := now.Year(), now.YearDay()
	switch {
	case y1 == y2 && d1 == d2:
		return say.F("tessera.last.time", map[string]any{"T": clock})
	case y1 == y2 && d1+1 == d2 && compact:
		return say.F("tessera.last.yest", map[string]any{"T": clock})
	case y1 == y2 && d1+1 == d2:
		return say.F("tessera.last.yesterday", map[string]any{"T": clock})
	}
	return say.F("tessera.last.date", map[string]any{"D": when.Format("2 Jan")})
}

func countdown(seconds int) string {
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func durationSeconds(text string) int {
	parts := strings.Split(text, ":")
	total := 0
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || v < 0 {
			return 0
		}
		total = total*60 + v
	}
	if len(parts) < 2 || len(parts) > 3 {
		return 0
	}
	return total
}

func timerText(t tessera.Tile, now time.Time) string {
	full := durationSeconds(t.ExtraText("dur"))
	switch t.State {
	case "active":
		end, _ := t.ExtraNumber("end")
		left := max(int(int64(end)-now.Unix()), 0)
		if full > 0 && left > full {
			left = full
		}
		return countdown(left)
	case "paused":
		return say.F(
			"tessera.paused",
			map[string]any{"T": countdown(durationSeconds(t.ExtraText("rem")))},
		)
	}
	if t.ExtraText("dur") == "" {
		return say.T("home.tile.off")
	}
	return countdown(full)
}

var lockStates = map[string]bool{
	"locked": true, "unlocked": true, "locking": true, "unlocking": true,
	"open": true, "opening": true, "jammed": true,
}

var binaryClasses = map[string]bool{
	"battery": true, "battery_charging": true, "carbon_monoxide": true, "cold": true,
	"connectivity": true, "door": true, "garage_door": true, "gas": true, "heat": true,
	"light": true, "lock": true, "moisture": true, "motion": true, "moving": true,
	"occupancy": true, "opening": true, "plug": true, "power": true, "presence": true,
	"problem": true, "running": true, "safety": true, "smoke": true, "sound": true,
	"tamper": true, "update": true, "vibration": true, "window": true,
}

func binaryWord(class string, on bool) string {
	if !binaryClasses[class] {
		if on {
			return say.T("home.tile.on")
		}
		return say.T("home.tile.off")
	}
	side := "off"
	if on {
		side = "on"
	}
	return say.T("tessera.binary." + class + "." + side)
}

func effectRunning(effect string) bool {
	switch strings.ToLower(strings.TrimSpace(effect)) {
	case "", "solid", "off", "none", "unknown", "unavailable":
		return false
	}
	return true
}

func domainLine(t tessera.Tile, l tessera.Layout, now time.Time, narrow bool) string {
	n := l.Numbers
	switch d := t.Domain(); {
	case d == "light" && t.State == "on" && effectRunning(t.Text("effect")):
		return t.Text("effect")
	case d == "light" && t.State == "on":
		if level, ok := t.Number("brightness"); ok {
			return n.Percentage(int(math.Round(min(max(level, 0), 255) * 100 / 255)))
		}
	case d == "timer":
		return timerText(t, now)
	case d == "script" || d == "scene" || d == "button" || d == "input_button":
		if t.State == "on" {
			return say.T("tessera.running")
		}
		last, _ := t.ExtraNumber("last")
		return lastRun(last, now, l.Clock24, narrow)
	case runs(t):
		switch {
		case running(t):
			return say.T("tessera.running")
		case t.State == "off":
			return say.T("home.tile.off")
		}
		last, _ := t.ExtraNumber("last")
		return lastRun(last, now, l.Clock24, narrow)
	case d == "lock":
		if w := t.ExtraText("w"); w != "" {
			return w
		}
		if lockStates[t.State] {
			return say.T("tessera.lock." + t.State)
		}
		return t.State
	case d == "binary_sensor" && (t.State == "on" || t.State == "off"):
		return binaryWord(t.Text("device_class"), t.State == "on")
	}
	switch t.State {
	case "on":
		return say.T("home.tile.on")
	case "off":
		return say.T("home.tile.off")
	}
	return word(t, n)
}
