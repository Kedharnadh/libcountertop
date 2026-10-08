package turn

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"
)

const replyResume = 80 * time.Millisecond

const chunkQueue = 64

type Reply struct {
	mu      sync.RWMutex
	closed  bool
	statsMu sync.Mutex
	chunks  chan []byte

	// Home Assistant paces itself to stay 384 ms ahead.
	buffer time.Duration

	held    []int16
	flowing bool

	bytes       int
	peak        int
	splicesAt   uint64
	underrunsAt uint64

	playingAt time.Time
}

func NewReply(splices, underruns uint64, buffer time.Duration) *Reply {
	return &Reply{
		chunks:      make(chan []byte, chunkQueue),
		buffer:      buffer,
		splicesAt:   splices,
		underrunsAt: underruns,
	}
}

func (s *Reply) Send(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return
	}
	data = slices.Clone(data)
	select {
	case s.chunks <- data:
	default:
		slog.Warn("reply chunk dropped", "bytes", len(data))
	}
}

func (s *Reply) Done() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.chunks)
	}
}

func (s *Reply) Play(ctx context.Context, p VoiceSink, started func()) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-s.chunks:
			if !ok {
				s.flush(p)
				return nil
			}
			s.take(data, p, started)
		}
	}
}

func (s *Reply) take(data []byte, p VoiceSink, started func()) {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	samples := make([]int16, len(data)/2)
	peak := 0
	for i := range samples {
		samples[i] = int16(uint16(data[i*2]) | uint16(data[i*2+1])<<8)
		peak = max(peak, int(samples[i]), -int(samples[i]))
	}

	if s.bytes == 0 {
		slog.Debug("reply audio started", "bytes", len(data))
		started()
	}

	s.bytes += len(data)
	s.peak = max(s.peak, peak)

	s.held = append(s.held, samples...)
	if p.Queued() == 0 {
		s.flowing = false
	}

	want := s.buffer
	if !s.playingAt.IsZero() {
		want = replyResume
	}
	if !s.flowing && len(s.held) < heldSamples(want) {
		return
	}

	if s.playingAt.IsZero() {
		s.playingAt = time.Now()
	}
	s.flowing = true
	s.flush(p)
}

func (s *Reply) flush(p VoiceSink) {
	out := s.held
	s.held = nil
	if len(out) > 0 {
		p.PlayVoice(out)
	}
}

func heldSamples(d time.Duration) int {
	return int(d/time.Millisecond) * 16000 / 1000
}

type VoiceSink interface {
	PlayVoice([]int16)
	Queued() int
}

type ReplyStats struct {
	Bytes, Peak            int
	SplicesAt, UnderrunsAt uint64
	PlayingAt              time.Time
}

func (s *Reply) Stats() ReplyStats {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	return ReplyStats{
		Bytes:       s.bytes,
		Peak:        s.peak,
		SplicesAt:   s.splicesAt,
		UnderrunsAt: s.underrunsAt,
		PlayingAt:   s.playingAt,
	}
}
