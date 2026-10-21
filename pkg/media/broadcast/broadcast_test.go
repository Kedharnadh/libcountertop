package broadcast

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type settings struct{ on bool }

func (s settings) Read() schema.Call { return schema.Call{Broadcasts: s.on} }

type peers struct {
	self   peer.Peer
	others []peer.Peer
}

func (p peers) Self() peer.Peer    { return p.self }
func (p peers) Peers() []peer.Peer { return p.others }
func (p peers) Find(id string) (peer.Peer, bool) {
	for _, o := range p.others {
		if o.ID == id {
			return o, true
		}
	}
	return peer.Peer{}, false
}

type sound struct {
	mu     sync.Mutex
	played []int16
	tones  int
}

func (s *sound) Frames() (<-chan []int16, func()) { return make(chan []int16), func() {} }
func (s *sound) Play(pcm []int16) {
	s.mu.Lock()
	s.played = append(s.played, pcm...)
	s.mu.Unlock()
}
func (s *sound) Claim(_ string, run func(context.Context) error) { _ = run(context.Background()) }
func (s *sound) Tone(schema.Stream, float64, tone.Sound) {
	s.mu.Lock()
	s.tones++
	s.mu.Unlock()
}

type tones struct{}

func (tones) Sound(schema.Chime, schema.Chime) tone.Sound { return tone.Sound{} }

type turns struct{ after int }

func (t *turns) AfterTurn(func()) { t.after++ }

func (t *turns) Listen() func() { return func() {} }

type messages struct{ titles []string }

func (m *messages) Info(title, _ string, _ time.Duration) { m.titles = append(m.titles, title) }

func client() *http.Client {
	return &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test servers
	}}
}

func receiver(t *testing.T, on bool, busy bool) (*Broadcaster, *sound, *messages, peer.Peer) {
	t.Helper()
	snd, msgs := &sound{}, &messages{}
	b := New(Dependencies{
		Settings: settings{on: on},
		Peers: peers{
			self:   peer.Peer{ID: "kitchen"},
			others: []peer.Peer{{ID: "office", Name: "Office"}},
		},
		Sound:    snd,
		Tones:    tones{},
		Turns:    &turns{},
		Busy:     func() bool { return busy },
		Messages: msgs,
	})
	mux := http.NewServeMux()
	mux.Handle("POST "+Path, b.Handler())
	srv := httptest.NewTLSServer(mux)
	t.Cleanup(srv.Close)
	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	n, _ := strconv.Atoi(port)
	p := peer.Peer{
		ID:       "kitchen",
		Name:     "Kitchen",
		Addrs:    []net.IP{net.ParseIP(host)},
		Secure:   n,
		Caps:     []string{Cap},
		Standing: peer.Home,
	}
	return b, snd, msgs, p
}

func sender(targets ...peer.Peer) *Broadcaster {
	return New(Dependencies{
		Settings: settings{on: true},
		Peers:    peers{self: peer.Peer{ID: "office"}, others: targets},
		Sound:    &sound{},
		Tones:    tones{},
		Turns:    &turns{},
		Client:   client(),
	})
}

func TestABroadcastReachesAHomePeer(t *testing.T) {
	_, snd, msgs, kitchen := receiver(t, true, false)
	clip := []int16{1, -2, 3, -4, 32767, -32768}

	if err := sender(kitchen).send(clip); err != nil {
		t.Fatal(err)
	}
	snd.mu.Lock()
	defer snd.mu.Unlock()
	if len(snd.played) != len(clip) {
		t.Fatalf("played %d samples, want %d", len(snd.played), len(clip))
	}
	for i := range clip {
		if snd.played[i] != clip[i] {
			t.Fatalf("sample %d is %d, want %d", i, snd.played[i], clip[i])
		}
	}
	if snd.tones != 1 {
		t.Errorf("the receiver chimed %d times, want once", snd.tones)
	}
	if len(msgs.titles) != 1 {
		t.Errorf("the receiver showed %d notices, want one", len(msgs.titles))
	}
}

func TestATurnedOffReceiverRefuses(t *testing.T) {
	_, snd, _, kitchen := receiver(t, false, false)
	if err := sender(kitchen).send([]int16{1, 2}); !errors.Is(err, ErrNobody) {
		t.Errorf("sending to a device that refuses said %v, want ErrNobody", err)
	}
	if len(snd.played) != 0 {
		t.Error("a device with broadcasts off played one")
	}
}

func TestABusyReceiverDropsIt(t *testing.T) {
	_, snd, _, kitchen := receiver(t, true, true)
	if err := sender(kitchen).send([]int16{1, 2}); !errors.Is(err, ErrNobody) {
		t.Errorf("sending to a busy device said %v, want ErrNobody", err)
	}
	if len(snd.played) != 0 {
		t.Error("a busy device played a broadcast")
	}
}

func TestOnlyHomePeersThatTakeBroadcastsAreTargets(t *testing.T) {
	b := sender(
		peer.Peer{ID: "a", Caps: []string{Cap}, Standing: peer.Home},
		peer.Peer{ID: "b", Caps: []string{Cap}, Standing: peer.Open},
		peer.Peer{ID: "c", Caps: []string{Cap}, Standing: peer.Foreign},
		peer.Peer{ID: "d", Caps: []string{Cap}, Standing: peer.Checking},
		peer.Peer{ID: "e", Caps: []string{"audio"}, Standing: peer.Home},
	)
	var got []string
	for _, p := range b.Targets() {
		got = append(got, p.ID)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("targets %v, want [a b]", got)
	}
}

func TestNoTargetsIsAnError(t *testing.T) {
	if err := sender().send([]int16{1}); !errors.Is(err, ErrNobody) {
		t.Errorf("sending to nobody said %v, want ErrNobody", err)
	}
}

func TestAnEmptyMessageAsksAndRecordsAfterTheTurn(t *testing.T) {
	tr := &turns{}
	b := sender()
	b.o.Turns = tr
	reply, err := b.run(esphome.Call{})
	if err != nil || reply == "" {
		t.Errorf("an empty broadcast answered %q, %v; want a prompt", reply, err)
	}
	if tr.after != 1 {
		t.Errorf("recording was queued %d times after the turn, want once", tr.after)
	}
}

func TestTypedTextIsNotBroadcast(t *testing.T) {
	if reply := sender().say("dinner time"); reply == "" {
		t.Error("typed text answered nothing; want an explanation")
	}
}

func TestSilenceIsNothingSaid(t *testing.T) {
	r := recording{quietChunks: 10}
	if _, err := r.finish(); !errors.Is(err, ErrNothingSaid) {
		t.Errorf("an unstarted recording said %v, want ErrNothingSaid", err)
	}
}
