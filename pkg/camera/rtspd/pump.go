package rtspd

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/session"
	"github.com/ygelfand/libcountertop/pkg/media/rtsp"
)

const linger = 15 * time.Second

type pump struct {
	at     int
	stream *rtsp.Stream
	source Source
	fps    int
	helper string

	mu      sync.Mutex
	running bool
	wanted  time.Time
	stop    chan struct{}
}

func (p *pump) demand() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wanted = time.Now()
	if p.running {
		if err := p.source.RequestKey(); err != nil {
			slog.Debug("asking for a keyframe failed", "stream", StreamName(p.at), "err", err)
		}
		return
	}
	p.running = true
	p.stop = make(chan struct{})
	go p.run(p.stop)
}

func (p *pump) halt() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running && p.stop != nil {
		close(p.stop)
		p.stop = nil
	}
}

func (p *pump) idle() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stream.Viewers() == 0 && time.Since(p.wanted) > linger
}

func (p *pump) run(stop chan struct{}) {
	defer func() {
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
	}()
	if err := p.feed(stop); err != nil {
		slog.Warn("the rtsp stream stopped", "stream", StreamName(p.at), "err", err)
	}
}

func (p *pump) feed(stop chan struct{}) error {
	for {
		again, err := p.serve(stop)
		if !again {
			return err
		}
	}
}

func (p *pump) serve(stop chan struct{}) (bool, error) {
	s, frames, err := p.source.JoinOrCard(p.at)
	if err != nil {
		return false, err
	}
	defer p.source.Leave(s, frames)
	sz := p.source.Sizes()[p.at]
	slog.Info(
		"rtsp stream up",
		"stream", StreamName(p.at),
		"size", fmt.Sprintf("%dx%d", sz.Width, sz.Height),
		"fps", p.fps,
		"source", p.helper,
	)

	for {
		select {
		case <-stop:
			return false, nil
		case <-s.Done():
			err := s.Err()
			switched := errors.Is(err, session.ErrMuted) || errors.Is(err, session.ErrShutter) ||
				errors.Is(err, session.ErrRestarted)
			return switched && p.stream.Viewers() > 0, err
		case f := <-frames:
			if p.idle() {
				slog.Info("rtsp stream down", "stream", StreamName(p.at))
				return false, nil
			}
			p.stream.Write(f.Data, f.PTS)
		}
	}
}
