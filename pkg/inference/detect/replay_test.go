package detect

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/zserge/microwakeword"

	assets "github.com/ygelfand/libcountertop/pkg/inference/wakeassets"
)

func TestReplayStopScores(t *testing.T) {
	path := os.Getenv("COUNTERTOP_REPLAY")
	if path == "" {
		t.Skip("COUNTERTOP_REPLAY not set")
	}

	channels, samples, err := readWAV(path)
	if err != nil {
		t.Fatal(err)
	}
	pick := channels - 1
	if s := os.Getenv("COUNTERTOP_REPLAY_CHANNEL"); s != "" {
		if pick, err = strconv.Atoi(s); err != nil || pick < 0 || pick >= channels {
			t.Fatalf("COUNTERTOP_REPLAY_CHANNEL %q: the file has %d channels", s, channels)
		}
	}

	model, err := assets.Stop(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var cfg microwakeword.Config
	cfg.ModelPath = model
	cfg.SlidingWindowSize = assets.StopWindowSize
	cfg.FeaturesStepMs = assets.StopFeatureStep
	cfg.ProbabilityCutoff = neverFires
	det, err := microwakeword.NewDetector(cfg)
	if err != nil {
		t.Fatal(err)
	}

	const frame = 320
	mono := make([]int16, len(samples)/channels)
	for i := range mono {
		mono[i] = samples[i*channels+pick]
	}

	var peak float64
	var peakAt int
	var report []string
	for at := 0; at+frame <= len(mono); at += frame {
		det.ProcessAudio(mono[at : at+frame])
		score := det.SlidingAverage()
		switch {
		case score > peak:
			peak, peakAt = score, at
		case peak >= 0.3 && at-peakAt > 16000/2:
			report = append(report, fmt.Sprintf("%6.2fs %.3f", float64(peakAt)/16000, peak))
			peak = 0
		}
	}
	if peak >= 0.3 {
		report = append(report, fmt.Sprintf("%6.2fs %.3f", float64(peakAt)/16000, peak))
	}
	t.Logf("channel %d of %d, peaks over 0.3:\n%s", pick, channels, strings.Join(report, "\n"))
}

func readWAV(path string) (int, []int16, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, err
	}
	if len(b) < 44 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return 0, nil, fmt.Errorf("%s is not a WAV", path)
	}
	channels := int(binary.LittleEndian.Uint16(b[22:]))
	if rate := binary.LittleEndian.Uint32(b[24:]); rate != 16000 {
		return 0, nil, fmt.Errorf("%s is %d Hz, want 16000", path, rate)
	}
	pcm := b[44:]
	out := make([]int16, len(pcm)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(pcm[2*i:]))
	}
	return channels, out, nil
}
