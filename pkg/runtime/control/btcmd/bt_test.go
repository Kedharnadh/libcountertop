package btcmd

import (
	"strings"
	"testing"
	"time"
)

type sink struct {
	thumb string
	art   uint16
	attrs []uint32
}

func (*sink) Looking()                           {}
func (*sink) Look(uint16)                        {}
func (*sink) Hello()                             {}
func (s *sink) Thumbnail(h string)               { s.thumb = h }
func (s *sink) Art(psm uint16)                   { s.art = psm }
func (*sink) Browsing()                          {}
func (*sink) Queue()                             {}
func (*sink) Dump(time.Duration) (string, error) { return "dumped", nil }
func (s *sink) Attributes(ids ...uint32)         { s.attrs = ids }

func TestSevenDigitsAreAnImageAndHexIsAChannel(t *testing.T) {
	s := &sink{}
	run := Run(s, "test")
	if _, err := run([]string{"art", "1234567"}); err != nil || s.thumb != "1234567" {
		t.Errorf("handle: %v, asked for %q", err, s.thumb)
	}
	if _, err := run([]string{"art", "0x1005"}); err != nil || s.art != 0x1005 {
		t.Errorf("channel: %v, asked for %#x", err, s.art)
	}
}

func TestTheLogTagIsNamed(t *testing.T) {
	out, err := Run(&sink{}, "lanovod")([]string{"browse"})
	if err != nil || !strings.Contains(out, "adb logcat -s lanovod") {
		t.Errorf("%q, %v", out, err)
	}
}

func TestAnUnknownVerbFails(t *testing.T) {
	for _, args := range [][]string{nil, {"nonesuch"}} {
		if _, err := Run(&sink{}, "x")(args); err == nil {
			t.Errorf("%v was accepted", args)
		}
	}
}

func TestAttributesDefaultToCoverArt(t *testing.T) {
	s := &sink{}
	if _, err := Run(s, "x")([]string{"attrs"}); err != nil || len(s.attrs) != 1 {
		t.Errorf("%v, asked for %v", err, s.attrs)
	}
}
