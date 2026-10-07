package a2dp

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ygelfand/libcountertop/pkg/bluetooth"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/l2cap"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/obex"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/sdp"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

// The artwork, which is three protocols deep: a channel of its own, OBEX on top of it, and the
// imaging profile's names on top of that.
//
// The session comes first and the handles follow: a phone mints a handle for a device that has one
// connected, and answers a request for the cover art attribute with nothing until then. So on every
// link: look up what the phone serves, open the channel it named, and connect.
//
// An image arrives over as many answers as it takes and none of them says which image it is, so one
// fetch at a time. What arrives is kept against its handle, because the same handles come round
// again as the list moves.
//
// One size: the profile fixes the thumbnail at 200x200 and a target holds nothing else.

// session is one conversation with the image service.
type session struct {
	mu sync.Mutex

	// connection is what the far end handed back, and every later request carries it.
	connection uint32
	open       bool

	// body accumulates across answers, asked is the handle they belong to, and began when it was
	// asked for. An answer that never comes would otherwise hold the queue for the life of the link.
	body  []byte
	asked string
	began time.Time

	// psm is the channel the far end serves images on, once its records have said. Allocated when
	// its service starts, so it is not the same twice.
	psm uint16

	// have is what has been fetched, and want the handles still to be, nearest first.
	have map[string]*kept
	want []string
}

// kept is the artwork for one handle and when it was last wanted.
type kept struct {
	raw  []byte
	id   uint64
	used time.Time
}

// artIDs numbers what has been fetched, for the card to tell one cover from the next.
var artIDs atomic.Uint64

// stays is how long an image outlives the last list it was in. The same few handles come round as
// a queue moves, and a phone charges a round trip for each.
const stays = 10 * time.Minute

// waits is how long a fetch has to finish before the next one may start without it.
const waits = 30 * time.Second

// clear forgets the conversation, for a stack that is being built again.
func (a *session) clear() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.connection, a.open, a.body, a.asked = 0, false, nil, ""
	a.have, a.want, a.psm, a.began = nil, nil, 0, time.Time{}
}

// looked takes what the far end's records said about images.
func (s *Sink) looked(records []sdp.Record) {
	for _, r := range records {
		psm, served := sdp.MorePSM(r, sdp.UUIDOBEX)
		if !served || psm == 0 {
			continue
		}

		s.art.mu.Lock()
		first := s.art.psm != psm
		s.art.psm = psm
		s.art.mu.Unlock()

		if first {
			slog.Debug("bluetooth images are served", "psm", fmt.Sprintf("%#04x", psm))
		}

		if !s.imaging() {
			s.Art(psm)
		}
		return
	}
}

// wants lines an image up to be fetched, and starts on it if nothing else is in flight. One already
// fetched, or already waiting, is left alone. The one being played goes to the front of the queue.
func (s *Sink) wants(handle string, first bool) {
	if handle == "" {
		return
	}

	s.art.mu.Lock()
	queued := false
	if k := s.art.have[handle]; k != nil {
		k.used = time.Now()
		queued = true
	}
	for _, h := range s.art.want {
		queued = queued || h == handle
	}

	switch {
	case queued:
	case first:
		s.art.want = append([]string{handle}, s.art.want...)
	default:
		s.art.want = append(s.art.want, handle)
	}
	s.art.mu.Unlock()

	s.pump()
}

// expire drops what has not been wanted for a while, so a long session does not keep every cover
// the phone has played.
func (s *Sink) expire() {
	s.art.mu.Lock()
	defer s.art.mu.Unlock()

	for h, k := range s.art.have {
		if time.Since(k.used) > stays {
			delete(s.art.have, h)
		}
	}
}

// image is what was fetched for a handle, and nil where nothing was.
func (s *Sink) image(handle string) ([]byte, uint64) {
	if handle == "" {
		return nil, 0
	}

	s.art.mu.Lock()
	defer s.art.mu.Unlock()

	k := s.art.have[handle]
	if k == nil {
		return nil, 0
	}
	k.used = time.Now()
	return k.raw, k.id
}

// cover is the artwork for the card, fetching it where it is not held yet.
func (s *Sink) cover(handle string) ([]byte, uint64) {
	if raw, id := s.image(handle); raw != nil {
		return raw, id
	}

	s.wants(handle, true)
	return nil, 0
}

// pump starts the next fetch if the session is up and nothing is in flight.
func (s *Sink) pump() {
	s.art.mu.Lock()
	id, open := s.art.connection, s.art.open

	if s.art.asked != "" {
		if time.Since(s.art.began) < waits {
			s.art.mu.Unlock()
			return
		}

		// Given up on rather than asked for again: a target that did not answer once answers the
		// same way twice, and the rest of the queue is behind it.
		gave := s.art.asked
		s.art.want = without(s.art.want, gave)
		s.art.asked, s.art.body = "", nil

		slog.Warn("bluetooth the image never came", "handle", gave)
	}

	if !open || len(s.art.want) == 0 {
		s.art.mu.Unlock()
		return
	}

	handle := s.art.want[0]
	s.art.asked, s.art.body, s.art.began = handle, nil, time.Now()
	s.art.mu.Unlock()

	s.tell(fmt.Sprintf("asking for image %s", handle), func(sink *bt.Sink) ([]l2cap.Frame, error) {
		return sink.Art(obex.Thumbnail(id, handle).Marshal())
	})
}

// fetched ends a fetch, keeps what it produced, and starts the next. One that produced nothing is
// still taken off the queue: a phone that refused once refuses again.
func (s *Sink) fetched(handle string, raw []byte) {
	s.art.mu.Lock()
	s.art.asked, s.art.body = "", nil

	if raw != nil {
		if s.art.have == nil {
			s.art.have = map[string]*kept{}
		}
		s.art.have[handle] = &kept{raw: raw, id: artIDs.Add(1), used: time.Now()}
	}

	s.art.want = without(s.art.want, handle)
	s.art.mu.Unlock()

	if raw != nil {
		s.options.Media().Changed()
	}
	s.pump()
}

// without is the queue with one handle taken out of it.
func without(queue []string, handle string) []string {
	out := queue[:0]
	for _, h := range queue {
		if h != handle {
			out = append(out, h)
		}
	}
	return out
}

// imaging reports whether the image channel is up, whatever has been said on it.
func (s *Sink) imaging() bool {
	s.mu.Lock()
	sink := s.sink
	s.mu.Unlock()

	return sink != nil && sink.Imaging()
}

// Art opens the image channel, which has to be looked up before it can be opened.
func (s *Sink) Art(psm uint16) {
	s.art.mu.Lock()
	s.art.connection, s.art.open, s.art.body, s.art.asked = 0, false, nil, ""
	s.art.psm = psm
	s.art.mu.Unlock()

	s.tell(fmt.Sprintf("opening the image channel on %#04x", psm),
		func(sink *bt.Sink) ([]l2cap.Frame, error) { return sink.OpenArt(psm) })
}

// Hello starts an OBEX session on the image channel, naming the cover art service.
func (s *Sink) Hello() {
	s.tell("connecting to the image service", func(sink *bt.Sink) ([]l2cap.Frame, error) {
		return sink.Art(obex.Connect(0x0200, obex.CoverArtTarget).Marshal())
	})
}

// Thumbnail asks for one image by hand.
func (s *Sink) Thumbnail(handle string) { s.wants(handle, true) }

// imaged takes one packet off the image channel.
//
// Which kind of answer this is depends on what was asked, which the far end does not repeat: a
// connect is answered with four fixed bytes in front of its headers and a get is not, and nothing
// in the packet says which.
func (s *Sink) imaged(packet []byte) {
	s.art.mu.Lock()
	connecting, handle := !s.art.open, s.art.asked
	s.art.mu.Unlock()

	p, err := obex.ParseAnswer(packet, connecting)
	if err != nil {
		slog.Warn("bluetooth image answer", "err", err)
		return
	}

	if p.Code != obex.OK && p.Code != obex.Continue {
		slog.Warn("bluetooth image refused", "handle", handle,
			"code", fmt.Sprintf("%#02x", p.Code))
		s.fetched(handle, nil)
		return
	}

	if connecting {
		id, ok := p.Connection()
		if !ok {
			slog.Warn("bluetooth the image service gave no session")
			return
		}

		s.art.mu.Lock()
		s.art.connection, s.art.open, s.art.asked = id, true, ""
		s.art.mu.Unlock()

		slog.Debug("bluetooth image session", "connection", fmt.Sprintf("%#08x", id), "mtu", p.MTU)
		s.pump()
		return
	}

	chunk, last := p.Body()

	s.art.mu.Lock()
	s.art.body = append(s.art.body, chunk...)
	whole, id := s.art.body, s.art.connection
	s.art.mu.Unlock()

	if !last {
		// The far end is holding the rest and will not send it unasked. The request for it carries
		// the session and nothing else; repeating the handle would start the fetch over.
		s.tell("asking for the rest of the image", func(sink *bt.Sink) ([]l2cap.Frame, error) {
			return sink.Art(obex.More(id).Marshal())
		})
		return
	}

	img, err := ui.Decode(whole)
	if err != nil {
		slog.Warn("bluetooth image", "handle", handle, "bytes", len(whole), "err", err)
		s.fetched(handle, nil)
		return
	}

	w, h := img.Size()
	slog.Debug("bluetooth image", "handle", handle, "bytes", len(whole),
		"size", fmt.Sprintf("%dx%d", w, h))

	s.fetched(handle, whole)
}
