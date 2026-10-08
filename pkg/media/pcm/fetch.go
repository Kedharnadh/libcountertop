package pcm

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ygelfand/libcountertop/pkg/fetch"
)

const Tail = 250

const fetchTimeout = 30 * time.Second

func Fetch(ctx context.Context, url string) ([]int16, error) {
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
	return Samples(body)
}

// A converted WAV can carry extra chunks before data, and a RIFF size can overflow a 32-bit int.
func Samples(body []byte) ([]int16, error) {
	if len(body) < 12 || string(body[0:4]) != "RIFF" || string(body[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a WAVE file: %d bytes", len(body))
	}

	total := uint64(len(body))
	for off := uint64(12); off+8 <= total; {
		id := string(body[off : off+4])
		size := uint64(binary.LittleEndian.Uint32(body[off+4 : off+8]))
		off += 8

		end := off + size
		if end > total {
			end = total
		}

		if id == "data" {
			pcm := body[off:end]
			samples := make([]int16, len(pcm)/2)
			for i := range samples {
				samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
			}
			return samples, nil
		}

		off = end
		if size%2 == 1 {
			off++
		}
	}
	return nil, fmt.Errorf("no data chunk in %d bytes", total)
}
