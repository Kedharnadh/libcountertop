package configsync

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/network/peer"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
)

type store struct {
	mu   sync.Mutex
	snap setting.Snapshot
	got  setting.Snapshot
}

func (s *store) Export() setting.Snapshot { return s.snap }

func (s *store) Import(snap setting.Snapshot) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = snap
	n := 0
	for _, values := range snap {
		n += len(values)
	}
	return n, nil
}

type peers struct {
	self  peer.Peer
	other *peer.Peer
}

func (p peers) Self() peer.Peer { return p.self }

func (p peers) Find(id string) (peer.Peer, bool) {
	if p.other != nil && p.other.ID == id {
		return *p.other, true
	}
	return peer.Peer{}, false
}

type device struct {
	sync  *Sync
	store *store
	peer  peer.Peer
}

func pair(t *testing.T) (a, b *device) {
	t.Helper()
	a, b = &device{store: &store{}}, &device{store: &store{}}
	a.store.snap = setting.Snapshot{"screen": {"theme": "ocean", "style": "glass"}}
	var aPeer, bPeer peer.Peer
	for _, d := range []struct {
		dev   *device
		self  *peer.Peer
		other *peer.Peer
		id    string
	}{{a, &aPeer, &bPeer, "aaaa"}, {b, &bPeer, &aPeer, "bbbb"}} {
		d.dev.sync = New(
			Dependencies{
				Store: d.dev.store,
				Peers: peers{self: peer.Peer{ID: d.id}, other: d.other},
				Client: &http.Client{Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					}, //nolint:gosec // test servers
				}},
			},
		)
		mux := http.NewServeMux()
		mux.Handle("POST /config/offer", d.dev.sync.OfferHandler())
		mux.Handle("POST /config/ask", d.dev.sync.AskHandler())
		mux.Handle("POST /config/pull", d.dev.sync.PullHandler())
		mux.Handle("POST /config/cancel", d.dev.sync.CancelHandler())
		srv := httptest.NewTLSServer(mux)
		t.Cleanup(srv.Close)
		host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
		n, _ := strconv.Atoi(port)
		*d.self = peer.Peer{ID: d.id, Name: d.id, Addrs: []net.IP{net.ParseIP(host)}, Secure: n}
		d.dev.peer = *d.self
		d.dev.sync.o.Peers = peers{self: *d.self, other: d.other}
	}
	return a, b
}

func until(t *testing.T, d *device, ok func(Exchange) bool) Exchange {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if x, there := d.sync.Now(); there && ok(x) {
			return x
		}
		time.Sleep(10 * time.Millisecond)
	}
	x, _ := d.sync.Now()
	t.Fatalf("exchange never got there: %+v", x)
	return Exchange{}
}

func TestSendingCarriesTheSettingsAcross(t *testing.T) {
	a, b := pair(t)
	if err := a.sync.Send(b.peer); err != nil {
		t.Fatal(err)
	}
	code := until(t, a, func(x Exchange) bool { return x.Role == Sending && x.Code != "" }).Code
	until(t, b, func(x Exchange) bool { return x.Role == Receiving && x.State == Waiting })

	b.sync.Enter(code)
	got := until(t, b, func(x Exchange) bool { return x.State == Done })
	if got.Applied != 2 || b.store.got["screen"]["theme"] != "ocean" {
		t.Fatalf("received %+v, stored %v", got, b.store.got)
	}
	until(t, a, func(x Exchange) bool { return x.State == Done })
}

func TestFetchingAsksTheOtherSideForACode(t *testing.T) {
	a, b := pair(t)
	if err := b.sync.Fetch(a.peer); err != nil {
		t.Fatal(err)
	}
	code := until(t, a, func(x Exchange) bool { return x.Role == Sending && x.Code != "" }).Code
	b.sync.Enter(code)
	until(t, b, func(x Exchange) bool { return x.State == Done && x.Applied == 2 })
}

func TestAWrongCodeCostsATryAndFiveEndIt(t *testing.T) {
	a, b := pair(t)
	if err := a.sync.Send(b.peer); err != nil {
		t.Fatal(err)
	}
	code := until(t, a, func(x Exchange) bool { return x.Code != "" }).Code
	until(t, b, func(x Exchange) bool { return x.State == Waiting })
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}

	b.sync.Enter(wrong)
	x := until(t, b, func(x Exchange) bool { return x.State == Waiting && x.Reason == ReasonWrong })
	if x.Left != tries-1 {
		t.Fatalf("after one wrong code %d tries left, want %d", x.Left, tries-1)
	}
	for i := 1; i < tries; i++ {
		until(t, b, func(x Exchange) bool { return x.State == Waiting })
		b.sync.Enter(wrong)
	}
	until(t, b, func(x Exchange) bool { return x.State == Failed })
	until(t, a, func(x Exchange) bool { return x.State == Failed && x.Reason == ReasonWrong })
	if b.store.got != nil {
		t.Fatal("settings arrived without the code")
	}
}

func TestOneExchangeAtATime(t *testing.T) {
	a, b := pair(t)
	if err := a.sync.Send(b.peer); err != nil {
		t.Fatal(err)
	}
	if err := a.sync.Fetch(b.peer); !errors.Is(err, ErrBusy) {
		t.Fatalf("second exchange = %v, want ErrBusy", err)
	}
	until(t, b, func(x Exchange) bool { return x.State == Waiting })
	b.sync.Close()
	until(t, a, func(x Exchange) bool { return x.State == Failed && x.Reason == ReasonCancelled })
	if _, there := b.sync.Now(); there {
		t.Fatal("closing left the exchange up")
	}
}
