package homeassistant

import (
	"context"
	"strconv"
	"strings"
)

// Home Assistant's option for an empty wake word slot.
const NoWakeWord = "no_wake_word"

type WakeWords struct {
	Controller *Controller
	MAC        func() string
}

func (w WakeWords) Allowed() bool { return w.Controller.Access() == Allowed }

func (w WakeWords) Choose(ctx context.Context, slot int, phrase string) error {
	if phrase == "" {
		phrase = NoWakeWord
	}
	return w.Controller.Perform(
		ctx,
		"select.select_option",
		map[string]string{"option": phrase},
		map[string]string{"entity_id": w.selector(slot)},
	)
}

func (w WakeWords) selector(slot int) string {
	suffix := "_wake_word"
	if slot > 0 {
		suffix += "_" + strconv.Itoa(slot+1)
	}
	mac := strings.ToLower(w.MAC())
	return `{%- set ns = namespace(e='') -%}` +
		`{%- for e in integration_entities('esphome') if e is match('select[.].*` + suffix + `$') and '` +
		mac + `' in (device_attr(e, 'connections') | string) -%}` +
		`{%- set ns.e = e -%}{%- endfor -%}{{ ns.e }}`
}
