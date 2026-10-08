package tz

import (
	"encoding/binary"
	"fmt"
	"time"
)

func Use(spec string) error {
	if spec == "" {
		return nil
	}

	loc, err := Location(spec)
	if err != nil {
		return err
	}

	time.Local = loc
	return nil
}

// LoadLocationFromTZData drops an unparseable footer without returning an error.
func Location(spec string) (*time.Location, error) {
	loc, err := time.LoadLocationFromTZData(spec, tzif(spec))
	if err != nil {
		return nil, fmt.Errorf("tz: %q: %w", spec, err)
	}
	if zone, _ := time.Now().In(loc).Zone(); zone == placeholder {
		return nil, fmt.Errorf("tz: %q: not a POSIX TZ string", spec)
	}
	return loc, nil
}

func Current() string {
	if time.Local == time.UTC {
		return ""
	}
	return time.Local.String()
}

// Go consults the footer rules only after the last transition.
const (
	magic   = "TZif"
	version = '2'

	changes = 1
	types   = 1

	placeholder = "<>"
	name        = placeholder + "\x00"
)

// TZif version 2: a 32-bit block, the same again in 64-bit, then the POSIX TZ footer.
func tzif(spec string) []byte {
	var b []byte

	header := func() {
		b = append(b, magic...)
		b = append(b, version)
		b = append(b, make([]byte, 15)...) // reserved

		// isutcnt, isstdcnt, leapcnt, timecnt, typecnt, charcnt
		for _, n := range []uint32{0, 0, 0, changes, types, uint32(len(name))} {
			b = binary.BigEndian.AppendUint32(b, n)
		}
	}

	block := func(wide bool) {
		if wide {
			b = binary.BigEndian.AppendUint64(b, 1<<63)
		} else {
			b = binary.BigEndian.AppendUint32(b, 1<<31)
		}
		b = append(b, 0)

		b = binary.BigEndian.AppendUint32(b, 0)
		b = append(b, 0, 0)

		b = append(b, name...)
	}

	header()
	block(false)
	header()
	block(true)

	return append(append(append(b, '\n'), spec...), '\n')
}
