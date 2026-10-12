package vad

import "time"

const (
	speechAbove = 0.5
	quietBelow  = 0.35
	startRun    = 3
	quietRun    = 14
	leadIn      = 5
	tail        = 3
	longest     = 3500 * time.Millisecond
)

type Clip struct {
	Samples []int16
	Start   time.Duration
	End     time.Duration
}

type Segmenter struct {
	d       *Detector
	pending []int16
	lead    [][]int16
	clip    []int16
	open    bool
	voiced  int
	quiet   int
	chunks  int
	started int
}

func NewSegmenter() (*Segmenter, error) {
	d, err := New()
	if err != nil {
		return nil, err
	}
	return &Segmenter{d: d, pending: make([]int16, 0, Chunk)}, nil
}

func (s *Segmenter) Reset() {
	s.d.Reset()
	s.pending = s.pending[:0]
	s.lead = s.lead[:0]
	s.clip = nil
	s.open = false
	s.voiced, s.quiet = 0, 0
}

func (s *Segmenter) Feed(samples []int16) []Clip {
	var out []Clip
	for len(samples) > 0 {
		n := min(Chunk-len(s.pending), len(samples))
		s.pending = append(s.pending, samples[:n]...)
		samples = samples[n:]
		if len(s.pending) < Chunk {
			break
		}
		chunk := append([]int16(nil), s.pending...)
		s.pending = s.pending[:0]
		if c, ok := s.step(chunk); ok {
			out = append(out, c)
		}
	}
	return out
}

func (s *Segmenter) step(chunk []int16) (Clip, bool) {
	p, err := s.d.Speech(chunk)
	if err != nil {
		return Clip{}, false
	}
	s.chunks++
	if !s.open {
		s.lead = append(s.lead, chunk)
		if len(s.lead) > leadIn {
			s.lead = s.lead[1:]
		}
		if p >= speechAbove {
			s.voiced++
		} else {
			s.voiced = 0
		}
		if s.voiced >= startRun {
			s.open, s.quiet = true, 0
			s.started = s.chunks - len(s.lead)
			for _, c := range s.lead {
				s.clip = append(s.clip, c...)
			}
			s.lead = s.lead[:0]
		}
		return Clip{}, false
	}
	if chunkTime(len(s.clip)/Chunk) <= longest+chunkTime(leadIn+quietRun) {
		s.clip = append(s.clip, chunk...)
	}
	if p < quietBelow {
		s.quiet++
	} else if p >= speechAbove {
		s.quiet = 0
	}
	if s.quiet < quietRun {
		return Clip{}, false
	}
	drop := s.quiet - tail
	end := s.chunks - drop
	clip := s.clip
	s.clip, s.open, s.voiced, s.quiet = nil, false, 0, 0
	c := Clip{Start: chunkTime(s.started), End: chunkTime(end)}
	if c.End-c.Start-chunkTime(tail+leadIn) > longest {
		return Clip{}, false
	}
	c.Samples = clip[:len(clip)-drop*Chunk]
	return c, true
}

func chunkTime(n int) time.Duration { return time.Duration(n) * Chunk * time.Second / SampleRate }
