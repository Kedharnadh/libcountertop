package widgets

import (
	"strconv"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/system/metrics"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const (
	radioEvery = 10 * time.Second
	weakDBM    = -78
	linkGlyph  = 0xF0319
)

var wifiGlyphs = [5]rune{icons.WifiOff, icons.Wifi1, icons.Wifi2, icons.Wifi3, icons.Wifi4}

type radio struct {
	at  time.Time
	dbm int
	ok  bool
}

func device(kind string) bool { return kind == "wifi" || kind == "link" || kind == "battery" }

func bars(dbm int) int {
	switch {
	case dbm >= 0:
		return 0
	case dbm >= -60:
		return 4
	case dbm >= -70:
		return 3
	case dbm >= weakDBM:
		return 2
	}
	return 1
}

func (v *View) signal(now time.Time) (int, bool) {
	v.mu.Lock()
	r := v.radio
	v.mu.Unlock()
	if now.Sub(r.at) < radioEvery {
		return r.dbm, r.ok
	}
	s, _, _ := metrics.Reader{}.Wifi()
	r = radio{at: now, dbm: int(s.Value), ok: s.Known}
	v.mu.Lock()
	v.radio = r
	v.mu.Unlock()
	return r.dbm, r.ok
}

func (v *View) deviceItem(it tessera.BarItem, fresh bool, now time.Time) (rune, string, bool) {
	switch it.Kind {
	case "link":
		return linkGlyph, "", !fresh
	case "wifi":
		dbm, ok := v.signal(now)
		strength := 0
		if ok {
			strength = bars(dbm)
		}
		if it.Weak == 1 && strength != 0 && dbm >= weakDBM {
			return 0, "", false
		}
		words := ""
		switch {
		case strength != 0 && it.Text == "%":
			words = strconv.Itoa(min(max(2*(dbm+100), 0), 100)) + "%"
		case strength != 0 && it.Text == "dBm":
			words = strconv.Itoa(dbm) + " dBm"
		}
		return wifiGlyphs[strength], words, true
	}
	return 0, "", false
}
