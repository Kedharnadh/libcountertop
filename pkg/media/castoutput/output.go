package castoutput

import (
	"context"
	"errors"
	"fmt"
	"hash/maphash"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/analysis"
	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
	"github.com/ygelfand/libcountertop/pkg/display/surface"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	videoplayer "github.com/ygelfand/libcountertop/pkg/display/videopage"
	"github.com/ygelfand/libcountertop/pkg/fetch"
	"github.com/ygelfand/libcountertop/pkg/media/nowplaying"
	"github.com/ygelfand/libcountertop/pkg/media/pcm"
	"github.com/ygelfand/libcountertop/pkg/media/playback"
	video "github.com/ygelfand/libcountertop/pkg/media/videostream"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

const Queued = time.Minute

var _ playback.Output = (*Output)(nil)
var _ pcm.Producer = (*Output)(nil)

type Output struct {
	control sync.Mutex
	options Options
	q       Queue
	play    func(context.Context, video.Stream) (video.Report, error)

	view      *videoplayer.Page
	viewStop  context.CancelFunc
	viewDone  chan struct{}
	under     *video.Beneath
	viewGen   int
	viewTimer *time.Timer
	handing   bool
	handed    *card

	feed sync.Mutex

	mu      sync.Mutex
	ended   func()
	cur     *playing
	lastSrc playback.Source
	lastAt  time.Duration
	gain    float32
	down    bool
	sess    *playback.Session
	note    string
	sessArt picture
	standby *standby
}

type picture struct {
	raw []byte
	id  uint64
}

var pictureSeed = maphash.MakeSeed()

func pictureOf(raw []byte) picture {
	if len(raw) == 0 {
		return picture{}
	}
	return picture{raw: raw, id: maphash.Bytes(pictureSeed, raw)}
}

func (p picture) image() *ui.Image { return ui.Picture(p.raw, p.id) }

type playing struct {
	workers sync.WaitGroup
	src     playback.Source
	card    *card
	stop    context.CancelFunc

	kept    []int16
	paused  bool
	waiting bool
	fed     int64

	art     picture
	artURL  string
	mark    picture
	markURL string
	thumbs  map[string]picture

	seen     chan struct{}
	seenOnce sync.Once
}

func (p *playing) shown() { p.seenOnce.Do(func() { close(p.seen) }) }

func New(options Options) *Output {
	if options.Rate == 0 {
		options.Rate = 48000
	}
	if options.Channels == 0 {
		options.Channels = 2
	}
	if options.DuckDB == nil {
		options.DuckDB = func() float64 { return ducking.DefaultDB }
	}
	o := &Output{options: options, q: options.Queue, gain: 1}
	o.standby = &standby{o: o}
	return o
}

func (o *Output) Play(src playback.Source) {
	o.control.Lock()
	defer o.control.Unlock()
	o.releaseLocked(nil)

	ctx, stop := context.WithCancel(context.Background())
	p := &playing{src: src, stop: stop}
	p.card = &card{o: o, p: p}
	pics, pictured := src.(playback.Pictures)
	pictured = pictured && pics.Pictured()
	if pictured {
		p.seen = make(chan struct{})
	}

	art := src.Showing().Art
	o.mu.Lock()
	o.cur, o.note = p, ""
	if h := o.handed; h != nil && art != "" && h.p.artURL == art {
		p.art, p.artURL = h.p.art, art
	}
	view := o.view
	o.mu.Unlock()
	if view != nil {
		view.SetNote("")
	}

	o.q.Drain()
	o.options.Media().Began(p.card)
	o.mu.Lock()
	o.handed = nil
	o.mu.Unlock()
	o.options.Took(o)
	o.Changed(src)

	if pictured {
		if o.play == nil {
			view, _ := o.hold(p.card)
			if b, ok := src.(playback.Branded); ok && b.Logo() != "" {
				url := b.Logo()
				safe.Go("cast logo", func() {
					if img := o.logoFor(url).image(); img != nil {
						view.SetLogo(img)
					}
				})
			}
		}
	}
	o.worker(p, "cast output", func() { o.pump(ctx, p) })
	if pictured {
		o.worker(p, "cast video", func() { o.picture(ctx, p, pics) })
	}
}

const viewGrace = 5 * time.Second

func (o *Output) hold(c videoplayer.Controls) (*videoplayer.Page, *video.Beneath) {
	o.mu.Lock()
	o.viewGen++
	if o.viewTimer != nil {
		o.viewTimer.Stop()
		o.viewTimer = nil
	}
	slog.Debug("cast video view held", "new", o.view == nil, "by", fmt.Sprintf("%T", c))
	if o.view == nil {
		ctx, stop := context.WithCancel(context.Background())
		o.view, o.viewStop, o.under = o.options.NewPage(c), stop, o.options.NewUnder()
		view := o.view
		o.under.Stalled = func() { view.SetLoading(true) }
		o.under.Resumed = func() { view.SetLoading(false) }
		done := make(chan struct{})
		o.viewDone = done
		go func() { defer close(done); view.Run(ctx) }()
	} else {
		o.view.Follow(c)
		o.view.SetLoading(true)
	}
	view, under := o.view, o.under
	o.mu.Unlock()
	return view, under
}

func (o *Output) letGo() {
	o.mu.Lock()
	view := o.view
	if view == nil {
		o.mu.Unlock()
		return
	}
	handing, gen := o.handing, o.viewGen
	if o.viewTimer != nil {
		o.viewTimer.Stop()
	}
	o.viewTimer = time.AfterFunc(viewGrace, func() { o.retireView(gen) })
	o.mu.Unlock()
	if handing {
		view.SetLoading(true)
	} else {
		view.SetPicture(false)
	}
}

func (o *Output) retireView(gen int) {
	o.mu.Lock()
	if o.viewGen != gen || o.view == nil {
		o.mu.Unlock()
		return
	}
	stop, done, under := o.viewStop, o.viewDone, o.under
	o.view, o.viewStop, o.viewDone, o.under, o.viewTimer = nil, nil, nil, nil, nil
	o.mu.Unlock()
	stop()
	<-done
	under.Clear()
}

const pictureWait = 30 * time.Second

func (o *Output) picture(ctx context.Context, p *playing, pics playback.Pictures) {
	play := o.play
	var view *videoplayer.Page
	if play == nil {
		v, under := o.hold(p.card)
		view = v
		defer o.letGo()
		play = func(ctx context.Context, s video.Stream) (video.Report, error) { return o.options.Video(ctx, under, s) }
	}
	err := o.pictureOnce(ctx, p, pics, play, view)
	if err == nil || ctx.Err() != nil {
		return
	}
	slog.Warn("cast video", "err", err)
	o.completed(p, err, true)
}

var errNoPicture = errors.New("cast video: no picture")

func (o *Output) pictureOnce(
	ctx context.Context,
	p *playing,
	pics playback.Pictures,
	play func(context.Context, video.Stream) (video.Report, error),
	view *videoplayer.Page,
) error {
	actx, cancel := context.WithCancel(ctx)
	defer cancel()
	var began, late atomic.Bool
	watch := time.AfterFunc(pictureWait, func() {
		if !began.Load() {
			late.Store(true)
			cancel()
		}
	})
	defer watch.Stop()

	s, closer, err := streamOf(actx, pics)
	if err != nil {
		return err
	}
	defer closer.Close()
	s.Clock = func() (time.Duration, bool) { return o.clock(p) }
	waiter, _ := p.src.(playback.Waiter)
	s.Started = func() {
		began.Store(true)
		p.shown()
		if waiter != nil {
			waiter.Waiting(false)
		}
		if view != nil {
			view.SetFrame(int(s.Width), int(s.Height))
			view.SetPicture(true)
		}
	}
	s.Waiting = func(w bool) {
		o.waiting(p, w)
		if waiter != nil {
			waiter.Waiting(w)
		}
		if view != nil {
			view.SetLoading(w)
		}
	}
	rep, err := play(actx, s)
	slog.Info(
		"cast video over",
		"shown",
		rep.Shown,
		"dropped",
		rep.Dropped,
		"size",
		fmt.Sprintf("%dx%d", s.Width, s.Height),
		"err",
		err,
	)
	if late.Load() {
		return errNoPicture
	}
	return err
}

func streamOf(ctx context.Context, pics playback.Pictures) (video.Stream, io.Closer, error) {
	if l, ok := pics.(playback.Live); ok {
		p, err := l.Live(ctx)
		if err != nil {
			return video.Stream{}, nil, err
		}
		if p.Sealed != nil {
			if !p.H264 {
				return video.Stream{}, nil, errNoPicture
			}
			s := video.Stream{
				Codec:   video.H264,
				Width:   uint32(p.Width),
				Height:  uint32(p.Height),
				Session: p.Session,
				Source:  sealedFrames(p.Sealed),
			}
			return s, io.NopCloser(nil), nil
		}
		if p.Next != nil {
			if !p.H264 {
				return video.Stream{}, nil, errNoPicture
			}
			s := video.Stream{
				Codec:  video.H264,
				Width:  uint32(p.Width),
				Height: uint32(p.Height),
				Source: liveFrames(p.Next),
			}
			return s, io.NopCloser(nil), nil
		}
	}
	stream, from, err := pics.Pictures(ctx)
	if err != nil {
		return video.Stream{}, nil, err
	}
	if stream == nil {
		return video.Stream{}, nil, errNoPicture
	}
	s, err := video.WebM(stream, from)
	if err != nil {
		stream.Close()
		return video.Stream{}, nil, err
	}
	return s, stream, nil
}

type liveFrames func(context.Context) ([]byte, time.Duration, error)

func (f liveFrames) Next(ctx context.Context) (video.Frame, error) {
	data, at, err := f(ctx)
	return video.Frame{Data: data, At: at}, err
}

type sealedFrames func(context.Context) ([]byte, time.Duration, *surface.Crypt, error)

func (f sealedFrames) Next(ctx context.Context) (video.Frame, error) {
	data, at, k, err := f(ctx)
	return video.Frame{Data: data, At: at, Crypt: k}, err
}

func (o *Output) Stop(src playback.Source) {
	o.control.Lock()
	defer o.control.Unlock()
	o.mu.Lock()
	o.handing = false
	o.mu.Unlock()
	o.releaseLocked(src)
	o.dropHanded()
}

func (o *Output) Handoff(src playback.Source) {
	o.control.Lock()
	defer o.control.Unlock()
	o.mu.Lock()
	o.handing = true
	o.mu.Unlock()
	o.releaseLocked(src)
}

func (o *Output) releaseLocked(src playback.Source) {
	o.mu.Lock()
	p := o.cur
	if p == nil || (src != nil && p.src != src) {
		o.mu.Unlock()
		return
	}
	o.cur = nil
	attended := src != nil && o.sess != nil && !o.handing
	handing := o.handing
	if handing {
		o.handed = p.card
	}
	o.mu.Unlock()

	p.stop()
	p.workers.Wait()
	o.options.Gave(o)
	switch {
	case attended:
		o.standBy()
	case !handing:
		o.options.Media().Release(p.card)
	}
	o.q.Drain()
}

func (o *Output) dropHanded() {
	o.mu.Lock()
	c := o.handed
	o.handed = nil
	o.mu.Unlock()
	if c != nil {
		o.options.Media().Release(c)
	}
}

func (o *Output) Attend(s *playback.Session) {
	o.mu.Lock()
	o.sess = s
	if s == nil {
		o.note, o.sessArt = "", picture{}
	}
	idle, held := o.cur == nil, o.view != nil
	o.mu.Unlock()
	if s == nil {
		o.options.Media().Ended(o.standby)
	}
	switch {
	case !idle:
	case s != nil:
		o.standBy()
		o.options.Media().Began(o.standby)
	default:
		o.options.Media().Release(o.standby)
		if held {
			o.letGo()
		}
	}
}

func (o *Output) Failed(err error) { o.fail(err, false) }

func (o *Output) end() {
	if o.ended != nil {
		safe.Go("cast end", o.ended)
	}
}

func (o *Output) fail(err error, unseen bool) {
	o.dropHanded()
	o.mu.Lock()
	o.note = err.Error()
	view, idle := o.view, o.cur == nil
	o.mu.Unlock()
	if view != nil && (idle || unseen) {
		view.SetPicture(false)
		view.SetNote(err.Error())
	}
	o.options.Media().Changed()
}

func (o *Output) standBy() {
	o.mu.Lock()
	s := o.sess
	o.mu.Unlock()
	if s == nil {
		return
	}
	o.options.Media().External(o.standby)
	if s.Logo == "" {
		return
	}
	safe.Go("cast logo", func() {
		pic := o.logoFor(s.Logo)
		if pic.raw == nil {
			return
		}
		o.mu.Lock()
		if o.sess == s {
			o.sessArt = pic
		}
		view, idle := o.view, o.cur == nil
		o.mu.Unlock()
		if view != nil && idle {
			view.SetLogo(pic.image())
		}
		o.options.Media().Changed()
	})
}

func (o *Output) Changed(src playback.Source) {
	o.mu.Lock()
	p := o.cur
	o.mu.Unlock()
	if p == nil || p.src != src {
		return
	}

	shown := src.Showing()

	o.mu.Lock()
	fetchArt := shown.Art != "" && shown.Art != p.artURL
	if fetchArt {
		p.artURL = shown.Art
	}
	fetchMark := shown.Mark != "" && shown.Mark != p.markURL
	if fetchMark {
		p.markURL = shown.Mark
	}
	o.mu.Unlock()

	if fetchArt {
		safe.Go("cast art", func() { o.fetchArt(p, shown.Art) })
	}
	if fetchMark {
		safe.Go("cast mark", func() {
			pic := o.logoFor(shown.Mark)
			o.mu.Lock()
			if pic.raw != nil && o.cur == p && p.markURL == shown.Mark {
				p.mark = pic
			}
			o.mu.Unlock()
			o.options.Media().Changed()
		})
	}
	o.options.Media().Changed()
}

func (o *Output) Pause(src playback.Source) {
	o.feed.Lock()
	o.mu.Lock()
	p := o.cur
	if p == nil || p.src != src || p.paused {
		o.mu.Unlock()
		o.feed.Unlock()
		return
	}
	p.paused = true
	o.mu.Unlock()
	o.aside()
	o.feed.Unlock()
	o.options.Media().Changed()
}

func (o *Output) Resume(src playback.Source) {
	o.feed.Lock()
	o.mu.Lock()
	p := o.cur
	if p == nil || p.src != src || !p.paused {
		o.mu.Unlock()
		o.feed.Unlock()
		return
	}
	p.paused = false
	hold := o.down || p.waiting
	o.mu.Unlock()
	if !hold {
		o.resume()
	}
	o.feed.Unlock()
	o.options.Media().Changed()
}

func (o *Output) waiting(p *playing, w bool) {
	o.feed.Lock()
	defer o.feed.Unlock()
	o.mu.Lock()
	if o.cur != p || p.waiting == w {
		o.mu.Unlock()
		return
	}
	p.waiting = w
	hold := o.down || p.paused
	o.mu.Unlock()
	switch {
	case w:
		o.aside()
	case !hold:
		o.resume()
	}
}

func (o *Output) clock(p *playing) (time.Duration, bool) {
	at, paused := o.Position(p.src)
	o.mu.Lock()
	waiting := p.waiting
	o.mu.Unlock()
	return at, !paused && !waiting
}

func (o *Output) Position(src playback.Source) (time.Duration, bool) {
	o.mu.Lock()
	p := o.cur
	if p == nil || p.src != src {
		defer o.mu.Unlock()
		if src == o.lastSrc {
			return o.lastAt, false
		}
		return src.Start(), false
	}
	fed, kept, paused := p.fed, int64(len(p.kept)/o.options.Channels), p.paused
	o.mu.Unlock()
	heard := max(fed-kept-int64(o.q.Queued()), 0)
	at := src.Start() + time.Duration(heard)*time.Second/time.Duration(o.options.Rate)
	o.mu.Lock()
	o.lastSrc, o.lastAt = src, at
	o.mu.Unlock()
	return at, paused
}

func (o *Output) Behind(src playback.Source) time.Duration {
	o.mu.Lock()
	p := o.cur
	var kept int
	if p != nil {
		kept = len(p.kept) / o.options.Channels
	}
	o.mu.Unlock()
	if p == nil || p.src != src {
		return 0
	}
	frames := o.q.Queued() + kept
	return time.Duration(frames) * time.Second / time.Duration(o.options.Rate)
}

func (o *Output) pump(ctx context.Context, p *playing) {
	buf := make([]int16, o.options.Rate/50*o.options.Channels)
	ahead := int(Queued * time.Duration(o.options.Rate) / time.Second)

	if p.seen != nil {
		select {
		case <-p.seen:
		case <-ctx.Done():
			return
		}
	}
	for ctx.Err() == nil {
		if o.standing() || o.q.Queued() > ahead {
			wait(ctx, 100*time.Millisecond)
			continue
		}

		n, err := p.src.Read(ctx, buf)
		if n > 0 {
			out := make([]int16, n)
			copy(out, buf[:n])
			analysis.Scale(out, o.level())
			o.feed.Lock()
			o.mu.Lock()
			p.fed += int64(n / o.options.Channels)
			held := o.cur != p || o.down || p.paused || p.waiting
			if held {
				p.kept = append(p.kept, out...)
			}
			o.mu.Unlock()
			if !held {
				o.q.Play(out)
			}
			o.feed.Unlock()
		}
		if err != nil && ctx.Err() != nil {
			return
		}
		if errors.Is(err, io.EOF) {
			o.finish(ctx, p)
			return
		}
		if err != nil {
			slog.Warn("cast playback", "err", err)
			o.completed(p, err, false)
			return
		}
		if n == 0 {
			wait(ctx, 20*time.Millisecond)
		}
	}
}

func (o *Output) finish(ctx context.Context, p *playing) {
	for ctx.Err() == nil && o.Behind(p.src) > 0 {
		wait(ctx, 100*time.Millisecond)
	}
	o.completed(p, nil, false)
}

func wait(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func (o *Output) standing() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.down || (o.cur != nil && (o.cur.paused || o.cur.waiting))
}

func (o *Output) level() float32 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.gain
}

func (o *Output) aside() {
	queued := o.q.Take()
	o.mu.Lock()
	if o.cur != nil {
		o.cur.kept = append(o.cur.kept, queued...)
	}
	o.mu.Unlock()
}

func (o *Output) resume() {
	o.mu.Lock()
	var kept []int16
	if o.cur != nil {
		kept, o.cur.kept = o.cur.kept, nil
	}
	o.mu.Unlock()
	if len(kept) > 0 {
		o.q.Play(kept)
	}
}

func (o *Output) Stand(down bool) {
	o.feed.Lock()
	defer o.feed.Unlock()
	o.mu.Lock()
	first := down && !o.down
	back := !down && o.down && (o.cur == nil || (!o.cur.paused && !o.cur.waiting))
	o.down = down
	o.mu.Unlock()

	switch {
	case first:
		o.aside()
	case back:
		o.resume()
	}
}

func (o *Output) Duck(on bool) {
	gain := float32(1)
	if on {
		gain = ducking.Gain(o.options.DuckDB())
	}
	o.mu.Lock()
	o.gain = gain
	o.mu.Unlock()
}

func (o *Output) Requeue() {
	gain := o.level()
	if gain == 1 {
		return
	}
	o.q.Adjust(func(samples []int16) { analysis.Scale(samples, gain) })
}

var (
	logoMu sync.Mutex
	logos  = map[string]picture{}
)

func (o *Output) logoFor(url string) picture {
	logoMu.Lock()
	pic, ok := logos[url]
	logoMu.Unlock()
	if ok {
		return pic
	}
	pic = fetchImage(url, "cast logo")
	if pic.raw != nil {
		logoMu.Lock()
		logos[url] = pic
		logoMu.Unlock()
	}
	return pic
}

func fetchImage(url, what string) picture {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return picture{}
	}
	resp, err := fetch.Client(15 * time.Second).Do(req)
	if err != nil {
		slog.Debug(what, "err", err)
		return picture{}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return picture{}
	}
	pic := pictureOf(body)
	if pic.image() == nil {
		return picture{}
	}
	return pic
}

func (o *Output) fetchThumb(p *playing, url string) {
	pic := fetchImage(url, "cast queue art")
	if pic.raw == nil {
		return
	}
	o.mu.Lock()
	if o.cur == p {
		p.thumbs[url] = pic
	}
	o.mu.Unlock()
	o.options.Media().Changed()
}

func (o *Output) fetchArt(p *playing, url string) {
	pic := fetchImage(url, "cast art")
	if pic.raw == nil {
		return
	}

	o.mu.Lock()
	if o.cur == p && p.artURL == url {
		p.art = pic
	}
	o.mu.Unlock()
	o.options.Media().Changed()
}

type card struct {
	o *Output
	p *playing
}

var _ nowplaying.Source = (*card)(nil)

func (c *card) Now() nowplaying.Now {
	s := c.p.src.Showing()
	elapsed, paused := c.o.Position(c.p.src)

	c.o.mu.Lock()
	art, mark := c.p.art, c.p.mark
	c.o.mu.Unlock()

	can := nowplaying.CanPause | nowplaying.CanStop
	if s.HasNext {
		can |= nowplaying.CanNext
	}
	if s.HasPrevious {
		can |= nowplaying.CanPrevious
	}
	queue := make([]nowplaying.Track, 0, len(s.Upcoming))
	var wanted []string
	c.o.mu.Lock()
	if c.p.thumbs == nil {
		c.p.thumbs = map[string]picture{}
	}
	for _, u := range s.Upcoming {
		pic, seen := c.p.thumbs[u.Art]
		if u.Art != "" && !seen {
			c.p.thumbs[u.Art] = picture{}
			wanted = append(wanted, u.Art)
		}
		queue = append(
			queue,
			nowplaying.Track{
				Title:  u.Title,
				Artist: u.Artist,
				Length: u.Length,
				Art:    pic.raw,
				ArtID:  pic.id,
				Play:   u.Play,
			},
		)
	}
	c.o.mu.Unlock()
	for _, url := range wanted {
		safe.Go("cast queue art", func() { c.o.fetchThumb(c.p, url) })
	}
	return nowplaying.Now{
		Queue:   queue,
		Playing: !paused,
		Paused:  paused,
		Title:   s.Title,
		Artist:  s.Artist,
		Album:   s.Album,
		Art:     art.raw,
		ArtID:   art.id,
		Mark:    mark.raw,
		MarkID:  mark.id,
		Elapsed: elapsed,
		Length:  s.Length,

		LiveWithin: s.LiveWithin,
		Can:        can,
	}
}

func (c *card) Open() bool {
	c.o.mu.Lock()
	view, cur := c.o.view, c.o.cur
	c.o.mu.Unlock()
	if cur != c.p || c.p.seen == nil {
		return false
	}
	if view != nil {
		view.Show()
		return true
	}
	c.o.hold(c)
	return true
}

func (c *card) Playing() (playing, paused bool) {
	_, paused = c.o.Position(c.p.src)
	return !paused, paused
}

func (c *card) Kind() nowplaying.Kind { return nowplaying.FromCast }
func (c *card) Label() string         { return c.p.src.Label() }

func (c *card) Play()  { c.p.src.Play() }
func (c *card) Pause() { c.p.src.Pause() }

func (c *card) Stop() {
	c.p.src.Stop()
	c.o.end()
}
func (c *card) Next()     { c.p.src.Next() }
func (c *card) Previous() { c.p.src.Previous() }

func (c *card) Seek(to time.Duration) {
	if s, ok := c.p.src.(playback.Seeker); ok {
		s.Seek(to)
	}
}

func (c *card) Marks() []videoplayer.Mark {
	m, ok := c.p.src.(playback.Marked)
	if !ok {
		return nil
	}
	var out []videoplayer.Mark
	for _, k := range m.Marks() {
		out = append(out, videoplayer.Mark{From: k.From, To: k.To,
			Color: theme.Color{R: byte(k.RGB >> 16), G: byte(k.RGB >> 8), B: byte(k.RGB)}})
	}
	return out
}

func (c *card) CanSeek() bool {
	_, ok := c.p.src.(playback.Seeker)
	return ok && c.p.src.Showing().Length > 0
}

type standby struct{ o *Output }

var _ nowplaying.Source = (*standby)(nil)

func (b *standby) Now() nowplaying.Now {
	b.o.mu.Lock()
	defer b.o.mu.Unlock()
	return nowplaying.Now{
		Hold:  b.o.sess != nil,
		Title: b.o.note,
		Art:   b.o.sessArt.raw,
		ArtID: b.o.sessArt.id,
	}
}

func (b *standby) Playing() (playing, paused bool) { return false, false }

func (b *standby) Kind() nowplaying.Kind { return nowplaying.FromCast }

func (b *standby) Open() bool {
	b.o.mu.Lock()
	s, note, art, playing := b.o.sess, b.o.note, b.o.sessArt, b.o.cur != nil
	b.o.mu.Unlock()
	if s == nil || !s.Pictured || playing {
		return false
	}
	view, _ := b.o.hold(b)
	view.SetPicture(false)
	view.SetLoading(false)
	view.SetNote(note)
	if img := art.image(); img != nil {
		view.SetLogo(img)
	}
	return true
}

func (b *standby) Wakes() bool {
	b.o.mu.Lock()
	defer b.o.mu.Unlock()
	return b.o.sess != nil && b.o.sess.Pictured
}

func (b *standby) Seek(time.Duration) {}
func (b *standby) CanSeek() bool      { return false }

func (b *standby) Label() string {
	b.o.mu.Lock()
	defer b.o.mu.Unlock()
	if b.o.sess == nil {
		return ""
	}
	return b.o.sess.Label
}

func (b *standby) Play()     {}
func (b *standby) Pause()    {}
func (b *standby) Stop()     { b.o.end() }
func (b *standby) Next()     {}
func (b *standby) Previous() {}

type Queue interface {
	Play([]int16)
	Take() []int16
	Queued() int
	Drain()
	Adjust(func([]int16))
}
type Media interface {
	Began(nowplaying.Source)
	Ended(nowplaying.Source)
	Release(nowplaying.Source)
	External(nowplaying.Source)
	Changed()
}
type Options struct {
	Queue          Queue
	Rate, Channels int
	DuckDB         func() float64
	Media          func() Media
	Took, Gave     func(pcm.Producer)
	NewPage        func(videoplayer.Controls) *videoplayer.Page
	NewUnder       func() *video.Beneath
	Video          func(context.Context, *video.Beneath, video.Stream) (video.Report, error)
}

func (o *Output) SetEnded(f func()) { o.mu.Lock(); o.ended = f; o.mu.Unlock() }

func (o *Output) worker(p *playing, name string, run func()) {
	p.workers.Add(1)
	safe.Go(name, func() { defer p.workers.Done(); run() })
}
func (o *Output) completed(p *playing, err error, unseen bool) {
	safe.Go("cast completion", func() {
		o.control.Lock()
		defer o.control.Unlock()
		o.mu.Lock()
		current := o.cur == p
		o.mu.Unlock()
		if !current {
			return
		}
		if err != nil {
			o.fail(err, unseen)
		}
		o.releaseLocked(p.src)
		if err != nil {
			safe.Go("cast stop", p.src.Stop)
		}
	})
}
