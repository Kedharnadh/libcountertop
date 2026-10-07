package pcm

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
	"io"
	"log/slog"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/noise"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

const (

	// pace is how often the stream looks to see whether the queue has room.
	pace = 100 * time.Millisecond

	// chunk is how much is taken off the wire at once, about a sixth of a second.
	chunk = 32 * 1024

	// stall is how long one read may produce nothing before the track is given up on.
	stall = 30 * time.Second
)

// frame is one sample on every channel.

// Stream plays a url through the speaker. Home Assistant converts the source with ffmpeg first, so
// what arrives is a WAV header followed by samples already at the playback rate: no decoder here,
// and nothing to resample.
//
// It streams rather than downloading. A track runs for minutes and the device has no room for one,
// so it keeps a second or so ahead of the speaker and reads no faster than it plays.
//
// It is the speaker driver's background: a reply or an announcement takes the speaker and this
// yields, carrying on from where it was rather than starting again.
type Stream struct {
	out     Sink
	options Options
	workers sync.WaitGroup
	parent  context.Context
	changed func()

	// bg is the queue of things that play for minutes. A track joins it while it has something to
	// play and leaves when it stops, so whatever it interrupted carries on afterwards.
	bg Arbitration

	mu    sync.Mutex
	track *track

	// down is where the arbiter last said this stream stands. It is told in full whenever anything
	// moves, so it is a copy of the answer rather than a tally of what has happened: nothing here has
	// to be undone as many times as it was done.
	down   bool
	paused bool

	// gate is closed to let the stream carry on, and non-nil for as long as it may not.
	gate chan struct{}

	// rewind is what was queued but not heard when the speaker was taken away, put back at the
	// front when playing resumes so the track carries on rather than jumping forward.
	rewind []int16

	// write is held while samples go into the queue, so taking the queue away cannot be followed by
	// the stream refilling it behind the sound that displaced it.
	write sync.Mutex

	// gain is what queue is multiplying by and target is what it is heading for, both under write
	// alongside the samples they scale. Moving rather than jumping: a step change of 15 dB is a click,
	// and a ramp over a few tens of milliseconds is what makes it sound deliberate.
	gain, target float32
}

// track is one thing being played. Identity is the point: a track that has been replaced knows not
// to report itself finished.
type track struct {
	// item names it in the log, and sounds is what is being generated when it is not a url.
	item   string
	sounds []string
	cancel context.CancelFunc
}

// NewStream builds the stream and joins the speaker's backgrounds. changed is called whenever what it
// is doing changes, which is what tells Home Assistant.
//
// It joins rather than registers: a track is one of several things that play for minutes, and it only
// takes the speaker while it has something to play.
// New constructs a bounded PCM stream using the supplied speaker and arbiter.
func New(options Options) *Stream {
	if options.Rate <= 0 {
		options.Rate = 48000
	}
	if options.Channels <= 0 {
		options.Channels = 2
	}
	if options.Changed == nil {
		options.Changed = func() {}
	}
	if options.DuckDB == nil {
		options.DuckDB = func() float64 { return ducking.DefaultDB }
	}
	return &Stream{options: options, out: options.Sink, bg: options.Arbitration, changed: options.Changed, gain: 1, target: 1, parent: context.Background()}
}
func (m *Stream) Start(ctx context.Context) { m.mu.Lock(); m.parent = ctx; m.mu.Unlock() }
func (m *Stream) Close() error              { m.Stop(); m.workers.Wait(); return nil }

// Duck lowers background music during a voice turn without pausing playback.
func (m *Stream) Duck(on bool) {
	if m == nil {
		return
	}
	level := float32(1)
	if on {
		level = ducking.Gain(m.options.DuckDB())
	}
	m.write.Lock()
	m.target = level
	m.write.Unlock()
}

// Play starts a url, replacing whatever was playing.
//
// It does not take the speaker from a reply or an announcement that is sounding. Those are seconds
// long and end on their own, and the track waits behind them rather than talking over them.
func (m *Stream) Play(url string) {
	if m == nil {
		slog.Warn("asked to play media with no speaker", "url", url)
		return
	}

	t, ctx := m.start(&track{item: url})
	slog.Info("playing media", "url", url)

	m.workers.Add(1)
	safe.Go("media", func() {
		defer m.workers.Done()
		err := m.run(ctx, t, url)
		if err != nil && ctx.Err() == nil {
			slog.Error("playing media failed", "err", err)
		}
		m.finished(t)
	})
}

// PlayNoise runs generated sound instead of a url, and does not stop until it is stopped: that is the
// point of it. Everything else about it is a track, so a turn ducks it, the buttons set its level and
// the media player reports it.
func (m *Stream) PlayNoise(sounds ...string) {
	if m == nil {
		slog.Warn("asked to play noise with no speaker", "sounds", sounds)
		return
	}

	fill := noise.Mix(m.options.Rate, sounds...)
	if fill == nil {
		slog.Warn("no such sound", "sounds", sounds)
		return
	}

	t, ctx := m.start(&track{item: strings.Join(sounds, " and "), sounds: sounds})
	slog.Info("playing noise", "sounds", sounds)

	m.workers.Add(1)
	safe.Go("noise", func() {
		defer m.workers.Done()
		err := m.generate(ctx, t, fill)
		if err != nil && ctx.Err() == nil {
			slog.Error("playing noise failed", "err", err)
		}
		m.finished(t)
	})
}

// Noise is what is being generated, empty when what is playing came from somewhere else. It is what
// keeps the entities honest when a track or a stop displaces the noise.
func (m *Stream) Noise() []string {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.track == nil {
		return nil
	}
	return slices.Clone(m.track.sounds)
}

// start makes a track the one being played, dropping whatever was.
func (m *Stream) start(t *track) (*track, context.Context) {
	m.mu.Lock()
	parent := m.parent
	m.mu.Unlock()
	ctx, cancel := context.WithCancel(parent)
	t.cancel = cancel

	m.mu.Lock()
	previous := m.track
	m.track, m.paused, m.rewind = t, false, nil
	m.regate()
	m.mu.Unlock()

	previous.stop()
	m.flush()

	// After the previous track is stopped, so taking over from ourselves is not mistaken for a second
	// background wanting the speaker.
	m.bg.Took(m)

	m.changed()
	return t, ctx
}

// Pause stops the track where it is. What was queued but not heard is kept, so resuming does not
// skip it.
func (m *Stream) Pause() {
	if m == nil {
		return
	}

	m.mu.Lock()
	if m.track == nil || m.paused {
		m.mu.Unlock()
		return
	}
	m.paused = true
	m.regate()
	m.mu.Unlock()

	// Asked of the arbiter rather than of this stream's own standing: what is queued is only ours to
	// put aside while nothing else is being heard.
	if m.bg.Owns(m) {
		m.keep()
	}
	m.changed()
}

// Unpause carries on from where Pause stopped.
func (m *Stream) Unpause() {
	if m == nil {
		return
	}

	m.mu.Lock()
	if m.track == nil || !m.paused {
		m.mu.Unlock()
		return
	}
	m.paused = false
	m.regate()
	m.mu.Unlock()

	m.changed()
}

// Stop ends the track. There is nothing to come back to afterwards.
func (m *Stream) Stop() {
	if m == nil {
		return
	}

	m.mu.Lock()
	t := m.track
	m.track, m.paused, m.rewind = nil, false, nil
	m.unblock()
	m.mu.Unlock()

	m.bg.Gave(m)

	if t == nil {
		return
	}
	t.stop()
	m.flush()
	m.changed()
}

// Stand implements speaker.Producer: the arbiter says whether this stream may be heard. The same
// standing arriving twice is the arbiter repeating itself, not a second thing wanting the speaker.
func (m *Stream) Stand(down bool) {
	if m == nil {
		return
	}

	m.mu.Lock()
	first := down && !m.down
	take := first && m.track != nil
	m.down = down
	if down {
		m.block()
	} else if !m.paused {
		m.unblock()
	}
	m.mu.Unlock()

	// Only on the way down, and only with a track: what is queued is this stream's to put aside then,
	// and taking it at any other time would be taking someone else's.
	if take {
		m.keep()
	}
}

// Playing reports whether a track is loaded and not paused, which is what Home Assistant is told.
// A track that is only waiting for a reply to finish is still playing: it is going to carry on
// without anyone asking it to.
func (m *Stream) Playing() (playing, paused bool) {
	if m == nil {
		return false, false
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.track != nil && !m.paused, m.track != nil && m.paused
}

// block and unblock hold and release the stream. Both want mu.
// regate opens the gate when nothing wants this stream quiet and closes it when anything does. Every
// reason is a state of its own, so this can be worked out again at any time rather than having to be
// arrived at by the right sequence of steps. Called with mu held.
func (m *Stream) regate() {
	if m.down || m.paused {
		m.block()
		return
	}
	m.unblock()
}

func (m *Stream) block() {
	if m.gate == nil {
		m.gate = make(chan struct{})
	}
}

func (m *Stream) unblock() {
	if m.gate != nil {
		close(m.gate)
		m.gate = nil
	}
}

// flush throws away audio that is ours to throw away. While something else holds the speaker the
// queue belongs to it, and emptying it would cut off a reply.
func (m *Stream) flush() {
	if !m.bg.Owns(m) {
		return
	}

	m.write.Lock()
	defer m.write.Unlock()
	m.out.Drain()
}

// keep empties the queue and remembers what was in it. Taken rather than dropped: this is the
// middle of a song, and what has not been heard is where playing has to start again from.
func (m *Stream) keep() {
	m.write.Lock()
	defer m.write.Unlock()

	kept := m.out.Take()
	if len(kept) == 0 {
		// Anything already stashed is still what has not been heard.
		return
	}

	m.mu.Lock()
	m.rewind = kept
	m.mu.Unlock()
}

// replay puts back what Suspend took, once the speaker is ours again.
func (m *Stream) replay() {
	m.write.Lock()
	defer m.write.Unlock()

	m.mu.Lock()
	back := m.rewind
	if m.gate != nil || len(back) == 0 {
		m.mu.Unlock()
		return
	}
	m.rewind = nil
	m.mu.Unlock()

	m.out.Play(back)
}

// finished clears the track once it has played out, unless it has already been replaced.
func (m *Stream) finished(t *track) {
	m.mu.Lock()
	if m.track != t {
		m.mu.Unlock()
		return
	}
	m.track, m.paused, m.rewind = nil, false, nil
	m.unblock()
	m.mu.Unlock()

	m.bg.Gave(m)

	slog.Info("media finished", "item", t.item)
	m.changed()
}

func (t *track) stop() {
	if t != nil {
		t.cancel()
	}
}

// run fetches the url and feeds it to the speaker as it arrives.
func (m *Stream) run(ctx context.Context, t *track, url string) error {
	// No timeout on the client: a track takes as long as it takes. What is bounded is a single read,
	// because a wedged connection otherwise holds the track open for as long as the kernel keeps
	// retrying — minutes of a player reporting that it is playing while nothing comes out.
	fetch, giveUp := context.WithCancel(ctx)
	defer giveUp()

	req, err := http.NewRequestWithContext(fetch, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}

	body := bufio.NewReaderSize(resp.Body, chunk)
	watchHeader := time.AfterFunc(stall, giveUp)
	err = Header(body, Format{Rate: m.options.Rate, Channels: m.options.Channels})
	watchHeader.Stop()
	if err != nil {
		return err
	}

	buf := make([]byte, chunk)
	for {
		if err := m.wait(ctx); err != nil {
			return err
		}

		// Armed only around the read: a track waiting for a turn to finish is not stalled, and
		// counting that time would end it for being interrupted.
		watchdog := time.AfterFunc(stall, giveUp)
		n, err := io.ReadFull(body, buf)
		watchdog.Stop()

		frame := m.options.Channels * 2
		if n >= frame {
			m.feed(t, buf[:n-n%frame])
		}
		switch {
		case err == nil:
		case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
			return m.settle(ctx)
		case fetch.Err() != nil && ctx.Err() == nil:
			return fmt.Errorf("nothing arrived for %s", stall)
		default:
			return err
		}
	}
}

// wait holds the stream until it may play and the queue has room for more.
func (m *Stream) wait(ctx context.Context) error {
	for {
		m.mu.Lock()
		gate := m.gate
		m.mu.Unlock()

		if gate != nil {
			select {
			case <-gate:
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		m.replay()
		if m.out.Queued() < m.options.Rate {
			return nil
		}

		select {
		case <-time.After(pace):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// settle waits for what has been queued to play out, so the track is not reported finished while
// the last of it is still sounding.
func (m *Stream) settle(ctx context.Context) error {
	for {
		if err := m.wait(ctx); err != nil {
			return err
		}
		if m.out.Queued() == 0 {
			return nil
		}

		select {
		case <-time.After(pace):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// generate feeds the speaker from a generator instead of a socket, for as long as the track lasts.
// Both channels get the same samples: one enclosure, one driver.
//
// The buffer is filled again each time round, so queue is free to scale it in place on its way out.
func (m *Stream) generate(ctx context.Context, t *track, fill noise.Fill) error {
	mono := make([]float32, chunk/frame)
	samples := make([]int16, len(mono)*m.options.Channels)

	for {
		if err := m.wait(ctx); err != nil {
			return err
		}

		fill(mono)
		for i, v := range mono {
			s := int16(v * math.MaxInt16)
			for channel := range m.options.Channels {
				samples[i*m.options.Channels+channel] = s
			}
		}
		m.queue(t, samples)
	}
}

func (m *Stream) feed(t *track, pcm []byte) {
	samples := make([]int16, len(pcm)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	m.queue(t, samples)
}

// queue hands samples over, unless the speaker was taken away in the meantime or this is no longer
// the track being played.
//
// The track has to be checked as well as the gate. Stopping opens the gate to release the goroutine
// waiting on it, and what that goroutine does next is read whatever the socket already holds — so
// without this it can put a chunk of a track that has been stopped into a queue that was just
// flushed, and the speaker plays it.
func (m *Stream) queue(t *track, samples []int16) {
	m.write.Lock()
	defer m.write.Unlock()

	m.mu.Lock()
	stale := m.gate != nil || m.track != t
	m.mu.Unlock()
	if stale {
		return
	}

	m.attenuate(samples)
	m.out.Play(samples)
}

// Requeue ducks what is already queued, which is up to a second of music the room would otherwise hear
// at full volume before anything scaled on its way in. Only the track is in there: a chime is mixed in
// after ducking, and so keeps its own level rather than fading with what is underneath it.
func (m *Stream) Requeue() {
	m.write.Lock()
	defer m.write.Unlock()

	m.out.Adjust(m.attenuate)
}

// attenuate applies the duck, moving towards the target rather than jumping to it. Wants write, which
// queue already holds.
//
// It rewrites the caller's slice, which is only ever a buffer feed just decoded. What replay puts back
// has been through here already and must not be scaled twice.
func (m *Stream) attenuate(samples []int16) {
	if m.gain == 1 && m.target == 1 {
		return
	}

	step := float32(1.0) / float32(m.options.Rate*m.options.Channels*60/1000)
	for i, s := range samples {
		switch {
		case m.gain < m.target:
			m.gain = min(m.gain+step, m.target)
		case m.gain > m.target:
			m.gain = max(m.gain-step, m.target)
		}
		samples[i] = int16(float32(s) * m.gain)
	}
}

// header reads past the WAV header and leaves the reader on the first sample.
//
// Home Assistant streams the file as ffmpeg produces it, which means the sizes in the header were
// written before the length was known: the data chunk runs until the connection ends, whatever it
// claims. The format is worth checking, though — the wrong rate or channel count is a track played
// at the wrong speed rather than an error anyone would see.
func Header(r *bufio.Reader, format Format) error {
	var riff [12]byte
	if _, err := io.ReadFull(r, riff[:]); err != nil {
		return fmt.Errorf("reading the WAVE header: %w", err)
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return fmt.Errorf("not a WAVE stream: %q", riff[0:4])
	}

	for {
		var head [8]byte
		if _, err := io.ReadFull(r, head[:]); err != nil {
			return fmt.Errorf("reading a WAVE chunk: %w", err)
		}
		id := string(head[0:4])
		size := int64(binary.LittleEndian.Uint32(head[4:8]))

		if id == "data" {
			return nil
		}

		// Everything before the samples is a handful of bytes. A size larger than that is a stream
		// that is not what it says it is, and allocating from it is how that becomes our problem.
		if size > chunk {
			return fmt.Errorf("%q chunk is %d bytes", id, size)
		}

		body := make([]byte, size+size%2)
		if _, err := io.ReadFull(r, body); err != nil {
			return fmt.Errorf("reading the %q chunk: %w", id, err)
		}
		if id == "fmt " {
			if err := supported(body[:size], format); err != nil {
				return err
			}
		}
	}
}

// supported checks a fmt chunk against what the codec takes.
func supported(fmtChunk []byte, format Format) error {
	if len(fmtChunk) < 16 {
		return fmt.Errorf("short fmt chunk: %d bytes", len(fmtChunk))
	}

	channels := binary.LittleEndian.Uint16(fmtChunk[2:])
	rate := binary.LittleEndian.Uint32(fmtChunk[4:])
	bits := binary.LittleEndian.Uint16(fmtChunk[14:])

	if int(channels) != format.Channels || int(rate) != format.Rate || bits != 16 {
		return fmt.Errorf("cannot play %d Hz %d channel %d bit audio", rate, channels, bits)
	}
	return nil
}

// Sink is the PCM queue owned by the platform speaker.
type Sink interface {
	Play([]int16)
	Take() []int16
	Adjust(func([]int16))
	Drain()
	Queued() int
}

type Producer interface {
	Stand(bool)
	Duck(bool)
	Requeue()
}

// Arbitration binds the platform's existing background speaker arbiter.
type Arbitration struct {
	Took func(Producer)
	Gave func(Producer)
	Owns func(Producer) bool
}
type Options struct {
	Sink           Sink
	Arbitration    Arbitration
	Rate, Channels int
	DuckDB         func() float64
	Changed        func()
}
type Format struct{ Rate, Channels int }
