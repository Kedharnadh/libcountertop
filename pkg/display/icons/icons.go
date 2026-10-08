package icons

import (
	_ "embed"
	"strconv"
)

//go:embed mdi.ttf
var MDI []byte

const Family = "Material Design Icons"

const (
	Wifi1      = 0xF091F
	Wifi2      = 0xF0922
	Wifi3      = 0xF0925
	Wifi4      = 0xF0928
	WifiOff    = 0xF092E
	LinkOff    = 0xF0338
	Battery    = 0xF0079
	BatteryLow = 0xF0083
)

func Codepoint(hex string) rune {
	if hex == "" || len(hex) > 6 {
		return 0
	}
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil || v < 0xF0000 || v > 0xF1FFF {
		return 0
	}
	return rune(v)
}

func Glyph(r rune) string {
	if r == 0 {
		return ""
	}
	return string(r)
}
