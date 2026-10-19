package rtspd

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/media/rtsp"
)

const linger = 15 * time.Second

type pump struct {
	at      int
	stream  *rtsp.Stream
	source  Source
	fps     int
	helper  string
	allowed func() bool

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
	if !p.allowed() {
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
	s, frames, err := p.source.Join(p.at)
	if err != nil {
		return err
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
			return nil
		case <-s.Done():
			return s.Err()
		case f := <-frames:
			if p.idle() || !p.allowed() {
				slog.Info("rtsp stream down", "stream", StreamName(p.at))
				return nil
			}
			p.stream.Write(f.Data, f.PTS)
		}
	}
}
