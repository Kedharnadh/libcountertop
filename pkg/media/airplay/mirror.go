package airplay

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/analysis"
	"github.com/ygelfand/libcountertop/pkg/audio/resample"
	"github.com/ygelfand/libcountertop/pkg/media/playback"
)

const (
	frameDepth  = 64
	silenceLead = 100 * time.Millisecond
	audioGap    = 150 * time.Millisecond
	mostAudio   = 500 * time.Millisecond
)

var (
	_ playback.Source   = (*mirror)(nil)
	_ playback.Pictures = (*mirror)(nil)
	_ playback.Live     = (*mirror)(nil)
)

type mirror struct {
	client     string
	size       mirroring
	rate       int
	channels   int
	disconnect func()

	frames chan frame
	done   chan struct{}
	ended  sync.Once

	mu       sync.Mutex
	first    time.Duration
	quiet    time.Time
	silenced int64
	pending  []int16
	heard    time.Time
	from     int
	convert  *resample.Rational
}

func newMirror(client string, size mirroring, rate, channels int, disconnect func()) *mirror {
	return &mirror{
		client:     client,
		size:       size,
		rate:       rate,
		channels:   channels,
		disconnect: disconnect,
		frames:     make(chan frame, frameDepth),
		done:       make(chan struct{}),
		first:      -1,
	}
}

func (m *mirror) push(ctx context.Context, f frame) {
	select {
	case m.frames <- f:
	case <-m.done:
	case <-ctx.Done():
	}
}

func (m *mirror) hear(a audio) {
	pcm := remix(a.pcm, a.channels, m.channels)
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.rate != m.rate {
		if m.convert == nil || m.from != a.rate {
			m.convert, m.from = resample.NewRational(a.rate, m.rate, m.channels), a.rate
		}
		pcm = m.convert.Run(pcm)
	}
	m.pending = append(m.pending, pcm...)
	most := int(mostAudio*time.Duration(m.rate)/time.Second) * m.channels
	if len(m.pending) > most {
		m.pending = append(m.pending[:0], m.pending[len(m.pending)-most:]...)
	}
	m.heard = time.Now()
}

func (m *mirror) end() { m.ended.Do(func() { close(m.done) }) }

func (m *mirror) Read(ctx context.Context, pcm []int16) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	select {
	case <-m.done:
		return 0, io.EOF
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if len(m.pending) > 0 {
		n := copy(pcm[:len(pcm)/m.channels*m.channels], m.pending)
		m.pending = m.pending[n:]
		m.quiet = time.Time{}
		return n, nil
	}
	if !m.heard.IsZero() && now.Sub(m.heard) < audioGap {
		return 0, nil
	}
	if m.quiet.IsZero() {
		m.quiet, m.silenced = now, 0
	}
	due := int64((now.Sub(m.quiet) + silenceLead) * time.Duration(m.rate) / time.Second)
	n := min(int64(len(pcm)/m.channels), due-m.silenced)
	if n <= 0 {
		return 0, nil
	}
	m.silenced += n
	out := pcm[:int(n)*m.channels]
	clear(out)
	return len(out), nil
}

func (m *mirror) Start() time.Duration { return 0 }

func (m *mirror) Showing() playback.Showing { return playback.Showing{Title: m.client} }

func (m *mirror) Label() string { return "AirPlay" }

func (m *mirror) Play()     {}
func (m *mirror) Pause()    {}
func (m *mirror) Stop()     { m.disconnect() }
func (m *mirror) Next()     {}
func (m *mirror) Previous() {}

func (m *mirror) Pictured() bool { return true }

func (m *mirror) Pictures(context.Context) (io.ReadCloser, time.Duration, error) {
	return nil, 0, nil
}

func (m *mirror) Live(context.Context) (playback.Picture, error) {
	return playback.Picture{
		H264:   true,
		Width:  int(m.size.width),
		Height: int(m.size.height),
		Next:   m.next,
		Sparse: true,
	}, nil
}

func (m *mirror) next(ctx context.Context) ([]byte, time.Duration, error) {
	select {
	case f := <-m.frames:
		m.mu.Lock()
		if m.first < 0 {
			m.first = f.at
		}
		at := f.at - m.first
		m.mu.Unlock()
		return f.data, at, nil
	case <-m.done:
		return nil, 0, io.EOF
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}
}

func remix(pcm []int16, from, to int) []int16 {
	if from == to {
		return pcm
	}
	mono := analysis.Mono(pcm, from, nil)
	if to == 1 {
		return mono
	}
	out := make([]int16, 0, len(mono)*to)
	for _, v := range mono {
		for range to {
			out = append(out, v)
		}
	}
	return out
}
