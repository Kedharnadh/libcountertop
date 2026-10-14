package pcm

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	gomp3 "github.com/hajimehoshi/go-mp3"

	"github.com/ygelfand/libcountertop/pkg/audio/resample"
	"github.com/ygelfand/libcountertop/pkg/fetch"
)

const Tail = 250

const fetchTimeout = 30 * time.Second

type decoded struct {
	samples        []int16
	rate, channels int
}

func Fetch(ctx context.Context, url string, rate int) ([]int16, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := fetch.Client(fetchTimeout).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	kind, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	d, err := decode(kind, body)
	if err != nil {
		return nil, err
	}
	return mono(d, rate), nil
}

// Home Assistant serves its converted WAV as application/octet-stream.
func decode(kind string, body []byte) (decoded, error) {
	if kind == "audio/mpeg" {
		return mp3(body)
	}
	return wav(body)
}

func mp3(body []byte) (decoded, error) {
	dec, err := gomp3.NewDecoder(bytes.NewReader(body))
	if err != nil {
		return decoded{}, fmt.Errorf("mp3: %w", err)
	}
	raw, err := io.ReadAll(dec)
	if err != nil {
		return decoded{}, fmt.Errorf("mp3: %w", err)
	}
	return decoded{samples: little(raw), rate: dec.SampleRate(), channels: 2}, nil
}

// A converted WAV can carry extra chunks before data, and a RIFF size can overflow a 32-bit int.
func wav(body []byte) (decoded, error) {
	if len(body) < 12 || string(body[0:4]) != "RIFF" || string(body[8:12]) != "WAVE" {
		return decoded{}, fmt.Errorf("not a WAVE file: %d bytes", len(body))
	}

	var d decoded
	total := uint64(len(body))
	for off := uint64(12); off+8 <= total; {
		id := string(body[off : off+4])
		size := uint64(binary.LittleEndian.Uint32(body[off+4 : off+8]))
		off += 8

		end := min(off+size, total)

		switch id {
		case "fmt ":
			chunk := body[off:end]
			if len(chunk) < 16 {
				return decoded{}, fmt.Errorf("short fmt chunk: %d bytes", len(chunk))
			}
			d.channels = int(binary.LittleEndian.Uint16(chunk[2:]))
			d.rate = int(binary.LittleEndian.Uint32(chunk[4:]))
			if bits := binary.LittleEndian.Uint16(chunk[14:]); bits != 16 {
				return decoded{}, fmt.Errorf("cannot play %d bit audio", bits)
			}
		case "data":
			if d.rate == 0 || d.channels == 0 {
				return decoded{}, fmt.Errorf("no fmt chunk before data")
			}
			d.samples = little(body[off:end])
			return d, nil
		}

		off = end
		if size%2 == 1 {
			off++
		}
	}
	return decoded{}, fmt.Errorf("no data chunk in %d bytes", total)
}

func little(raw []byte) []int16 {
	samples := make([]int16, len(raw)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(raw[i*2:]))
	}
	return samples
}

func mono(d decoded, rate int) []int16 {
	out := d.samples
	if d.channels > 1 {
		out = make([]int16, len(d.samples)/d.channels)
		for i := range out {
			var sum int
			for c := range d.channels {
				sum += int(d.samples[i*d.channels+c])
			}
			out[i] = int16(sum / d.channels)
		}
	}
	if d.rate != rate {
		out = resample.NewRational(d.rate, rate, 1).Run(out)
	}
	return out
}
