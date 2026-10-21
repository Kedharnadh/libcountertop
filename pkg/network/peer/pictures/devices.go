package pictures

import "embed"

//go:embed devices/*.svg
var devices embed.FS

const Aspect = 190.0 / 320.0

func For(board string) (key, svg string) {
	if b, err := devices.ReadFile("devices/" + board + ".svg"); err == nil {
		return "device-" + board, string(b)
	}
	b, _ := devices.ReadFile("devices/default.svg")
	return "device-default", string(b)
}
