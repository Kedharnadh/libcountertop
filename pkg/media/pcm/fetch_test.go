package pcm

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func serve(t *testing.T, kind string, body []byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", kind)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func tone(rate, channels int, seconds float64) []byte {
	frames := int(float64(rate) * seconds)
	var b bytes.Buffer
	for i := range frames {
		v := int16(8000 * math.Sin(2*math.Pi*440*float64(i)/float64(rate)))
		for range channels {
			_ = binary.Write(&b, binary.LittleEndian, v)
		}
	}
	return wave(fmtChunk(uint16(channels), uint32(rate), 16), b.Bytes())
}

func TestFetchDecodesWhatHomeAssistantServes(t *testing.T) {
	mp3, err := os.ReadFile("testdata/tone.mp3")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		kind string
		body []byte
	}{
		{"voice wav", "application/octet-stream", tone(16000, 1, 1)},
		{"stereo wav", "audio/x-wav", tone(48000, 2, 1)},
		{"tts mp3", "audio/mpeg", mp3},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := Fetch(context.Background(), serve(t, c.kind, c.body), 16000)
			if err != nil {
				t.Fatal(err)
			}
			if n := len(got); n < 15000 || n > 18500 {
				t.Errorf("%d samples, want about one second at 16 kHz", n)
			}
			var peak int16
			for _, s := range got {
				peak = max(peak, s)
			}
			if peak < 4000 {
				t.Errorf("peak %d, want the tone", peak)
			}
		})
	}
}

func TestFetchRefusesWhatItCannotPlay(t *testing.T) {
	for _, c := range []struct {
		name  string
		kind  string
		body  []byte
		wants string
	}{
		{"mp3 as octet stream", "application/octet-stream", []byte("ID3 then frames"), "not a WAVE"},
		{"garbage mp3", "audio/mpeg", []byte("not an mp3 at all"), "mp3"},
		{"24 bit wav", "audio/wav", wave(fmtChunk(1, 16000, 24), nil), "24 bit"},
		{"data before fmt", "audio/wav", wave(nil, []byte{1, 2}), "no fmt chunk"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Fetch(context.Background(), serve(t, c.kind, c.body), 16000)
			if err == nil {
				t.Fatal("played it anyway")
			}
			if !strings.Contains(err.Error(), c.wants) {
				t.Errorf("said %q, want something about %q", err, c.wants)
			}
		})
	}
}
