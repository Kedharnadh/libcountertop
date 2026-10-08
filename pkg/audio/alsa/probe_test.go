package alsa

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestProbeMixer(t *testing.T) {
	if os.Getenv("ALSA_PROBE") == "" {
		t.Skip("set ALSA_PROBE=1 on a device with a sound card")
	}

	m, err := OpenMixer(0)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	controls, err := m.Controls()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf(
		"elem_value is %d bytes, elem_info %d; %d controls",
		elemValueSize,
		elemInfoSize,
		len(controls),
	)

	var integers, enums, multi int
	for _, c := range controls {
		v, err := m.Get(c)
		if err != nil {
			t.Errorf("%s: %v", c.Name, err)
			continue
		}
		switch c.Type {
		case TypeEnumerated:
			enums++
			if len(v) > 0 && int(v[0]) >= len(c.Items) {
				t.Logf("%s reports item %d with only %d items", c.Name, v[0], len(c.Items))
			}
		case TypeInteger, TypeBoolean:
			integers++
		}

		if c.Count > 1 {
			multi++
			t.Logf("%-28s type %d count %d value %v", c.Name, c.Type, c.Count, v)
		}
	}
	t.Logf("read %d integer or boolean controls, %d enumerated, %d with several values",
		integers, enums, multi)

	for name := range strings.SplitSeq(os.Getenv("ALSA_PROBE_CONTROLS"), ",") {
		if name == "" {
			continue
		}
		c, err := m.Find(name)
		if err != nil {
			t.Logf("%s: %v", name, err)
			continue
		}
		v, err := m.Get(c)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		t.Logf("%-24s type %d count %d value %v items %v", c.Name, c.Type, c.Count, v, c.Items)
	}
}

func TestProbeDevices(t *testing.T) {
	if os.Getenv("ALSA_PROBE") == "" {
		t.Skip("set ALSA_PROBE=1 on a device with a sound card")
	}
	nodes, err := filepath.Glob("/dev/snd/pcmC0D*")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		name := strings.TrimPrefix(filepath.Base(node), "pcmC0D")
		if name == "" {
			continue
		}
		capture := strings.HasSuffix(name, "c")
		device, err := strconv.Atoi(name[:len(name)-1])
		if err != nil {
			continue
		}
		r, err := Refine(0, device, capture)
		t.Logf("device %d capture=%v: %v %v", device, capture, r, err)
	}
}

func TestProbeQuietChannels(t *testing.T) {
	if os.Getenv("ALSA_PROBE") == "" {
		t.Skip("set ALSA_PROBE=1 on a device with a sound card")
	}
	device, err := strconv.Atoi(os.Getenv("ALSA_PROBE_DEVICE"))
	if err != nil {
		t.Skip("set ALSA_PROBE_DEVICE to a capture device number")
	}
	r, err := Refine(0, device, true)
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Channels:   int(r.MaxChannels),
		Rate:       int(max(r.MinRate, 16000)),
		PeriodSize: 320,
		Periods:    8,
	}
	switch {
	case slices.Contains(r.Formats, FormatS24_3LE):
		cfg.Format, cfg.Bits = FormatS24_3LE, 24
	case slices.Contains(r.Formats, FormatS16LE):
		cfg.Format, cfg.Bits = FormatS16LE, 16
	case slices.Contains(r.Formats, FormatS32LE):
		cfg.Format, cfg.Bits = FormatS32LE, 32
	default:
		t.Skipf("no supported format in %v", r.Formats)
	}

	c, err := Open(0, device, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	width := cfg.Bits / 8
	buf := make([]byte, c.FrameBytes()*cfg.PeriodSize)
	sum := make([]float64, cfg.Channels)
	peak := make([]int64, cfg.Channels)
	frames := 0
	for frames < 2*cfg.Rate {
		n, err := c.Read(buf)
		if err != nil {
			continue
		}
		for f := 0; f+c.FrameBytes() <= n; f += c.FrameBytes() {
			for ch := range cfg.Channels {
				b := buf[f+ch*width : f+(ch+1)*width]
				v := int64(int8(b[width-1]))
				for i := width - 2; i >= 0; i-- {
					v = v<<8 | int64(b[i])
				}
				sum[ch] += float64(v) * float64(v)
				peak[ch] = max(peak[ch], v, -v)
			}
			frames++
		}
	}
	for ch := range cfg.Channels {
		t.Logf(
			"channel %d: rms %.1f peak %d over %d frames",
			ch,
			math.Sqrt(sum[ch]/float64(frames)),
			peak[ch],
			frames,
		)
	}
}
