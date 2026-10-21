package trust

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/network/peer"
)

type device struct {
	t       *Trust
	srv     *httptest.Server
	mu      sync.Mutex
	key     string
	peers   []peer.Peer
	changes int
}

func (d *device) know(p ...peer.Peer) { d.mu.Lock(); d.peers = p; d.mu.Unlock() }

func (d *device) changed() int { d.mu.Lock(); defer d.mu.Unlock(); return d.changes }

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("%s never happened", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (d *device) setKey(k string) { d.mu.Lock(); d.key = k; d.mu.Unlock() }

func (d *device) peer(id string) peer.Peer {
	host, port, _ := net.SplitHostPort(d.srv.Listener.Addr().String())
	n, _ := strconv.Atoi(port)
	return peer.Peer{ID: id, Name: id, Addrs: []net.IP{net.ParseIP(host)}, Secure: n}
}

func newDevice(t *testing.T, id, key string) *device {
	t.Helper()
	cert, err := LoadIdentity(filepath.Join(t.TempDir(), "peer.pem"))
	if err != nil {
		t.Fatal(err)
	}
	d := &device{key: key}
	d.t, err = New(Options{
		ID:       id,
		Identity: cert,
		Key:      func() string { d.mu.Lock(); defer d.mu.Unlock(); return d.key },
		Peers:    func() []peer.Peer { d.mu.Lock(); defer d.mu.Unlock(); return d.peers },
		Changed:  func() { d.mu.Lock(); d.changes++; d.mu.Unlock() },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.t.stop(); d.t.workers.Wait() })
	mux := http.NewServeMux()
	mux.Handle("POST "+ConfirmPath, d.t.ConfirmHandler())
	mux.Handle("POST "+ProvePath, d.t.ProveHandler())
	mux.Handle("GET /ping", d.t.Home(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "pong")
	})))
	d.srv = httptest.NewUnstartedServer(mux)
	d.srv.TLS = d.t.ServerConfig()
	d.srv.StartTLS()
	t.Cleanup(d.srv.Close)
	return d
}

func ping(t *testing.T, from, to *device) int {
	t.Helper()
	resp, err := from.t.Client(askWait).Get(to.srv.URL + "/ping")
	if err != nil {
		return 0
	}
	resp.Body.Close()
	return resp.StatusCode
}

func TestMatchingKeysMakeAHome(t *testing.T) {
	a, b := newDevice(t, "a", "pancake tuesday"), newDevice(t, "b", "pancake tuesday")
	a.t.confirm(context.Background(), b.peer("b"), "pancake tuesday")

	if got := a.t.Standing("b"); got != peer.Home {
		t.Errorf("a sees b as %s, want home", got)
	}
	if got := b.t.Standing("a"); got != peer.Checking {
		t.Errorf("b sees a as %s before checking a itself, want checking", got)
	}
	if got := ping(t, a, b); got != http.StatusOK {
		t.Errorf("a home request to b answered %d", got)
	}
	if got := ping(t, b, a); got != http.StatusOK {
		t.Errorf("b home request to a answered %d", got)
	}
}

func TestADifferentKeyIsForeign(t *testing.T) {
	a, b := newDevice(t, "a", "pancake tuesday"), newDevice(t, "b", "waffle wednesday")
	a.t.confirm(context.Background(), b.peer("b"), "pancake tuesday")

	if got := a.t.Standing("b"); got != peer.Foreign {
		t.Errorf("a sees b as %s, want foreign", got)
	}
	if got := ping(t, a, b); got != 0 {
		t.Errorf("a reached a foreign b: %d", got)
	}
	if got := ping(t, b, a); got == http.StatusOK {
		t.Error("a foreign b got through a's guard")
	}
}

func TestNoKeyOnTheFarSideIsForeign(t *testing.T) {
	a, b := newDevice(t, "a", "pancake tuesday"), newDevice(t, "b", "")
	a.t.confirm(context.Background(), b.peer("b"), "pancake tuesday")

	if got := a.t.Standing("b"); got != peer.Foreign {
		t.Errorf("a sees a keyless b as %s, want foreign", got)
	}
	if got := b.t.Standing("a"); got != peer.Checking {
		t.Errorf("keyless b sees a as %s before checking a itself, want checking", got)
	}
	if got := ping(t, a, b); got != 0 {
		t.Errorf("a reached a keyless b: %d", got)
	}
	if got := ping(t, b, a); got == http.StatusOK {
		t.Error("a keyless b got through a's guard")
	}
}

func TestAFailedProofCannotNameAnotherDevice(t *testing.T) {
	a, b := newDevice(t, "a", "pancake tuesday"), newDevice(t, "b", "pancake tuesday")
	b.t.confirm(context.Background(), a.peer("a"), "pancake tuesday")
	if got := b.t.Standing("a"); got != peer.Home {
		t.Fatalf("b sees a as %s, want home", got)
	}

	stranger := newDevice(t, "a", "waffle wednesday")
	stranger.t.confirm(context.Background(), b.peer("b"), "waffle wednesday")

	if got := b.t.Standing("a"); got != peer.Home {
		t.Errorf("a stranger claiming a's id left b seeing a as %s, want home", got)
	}
	if got := ping(t, stranger, b); got == http.StatusOK {
		t.Error("the stranger got through b's guard")
	}
}

func TestEachAddressHasItsOwnTries(t *testing.T) {
	d := newDevice(t, "a", "pancake tuesday")
	now := time.Unix(1_800_000_000, 0)
	d.t.o.Now = func() time.Time { return now }

	for range burst {
		if !d.t.admit("10.0.0.66") {
			t.Fatal("an address was refused inside its tries")
		}
	}
	if d.t.admit("10.0.0.66") {
		t.Error("an address went past its tries")
	}
	if !d.t.admit("10.0.0.7") {
		t.Error("another address was refused because the first used up its tries")
	}

	now = now.Add(window)
	if !d.t.admit("10.0.0.66") {
		t.Error("an address was still refused after the window passed")
	}
}

func TestAllAddressesTogetherStopAtTheCeiling(t *testing.T) {
	d := newDevice(t, "a", "pancake tuesday")
	now := time.Unix(1_800_000_000, 0)
	d.t.o.Now = func() time.Time { return now }

	for i := range ceiling {
		if !d.t.admit("10.0.0." + strconv.Itoa(i/burst)) {
			t.Fatalf("try %d was refused below the ceiling", i)
		}
	}
	if d.t.admit("10.0.1.1") {
		t.Error("a fresh address went past the ceiling")
	}
}

func TestNoKeysMakeAHome(t *testing.T) {
	a, b := newDevice(t, "a", ""), newDevice(t, "b", "")
	if got := a.t.Standing("b"); got != peer.Checking {
		t.Errorf("an unconfirmed keyless peer is %s, want checking", got)
	}
	if got := ping(t, a, b); got != 0 {
		t.Errorf("an unconfirmed keyless peer was reached: %d", got)
	}
	a.t.confirm(context.Background(), b.peer("b"), "")
	if got := a.t.Standing("b"); got != peer.Home {
		t.Errorf("two keyless devices: a sees b as %s, want home", got)
	}
	if got := ping(t, a, b); got != http.StatusOK {
		t.Errorf("a home request between keyless devices answered %d", got)
	}
}

func TestAnOldPeerIsOpenOnlyWithoutAKey(t *testing.T) {
	a := newDevice(t, "a", "")
	old := peer.Peer{ID: "old", Port: 80}
	a.t.confirm(context.Background(), old, "")
	if got := a.t.Standing("old"); got != peer.Open {
		t.Errorf("a keyless device sees an old peer as %s, want open", got)
	}
	a.setKey("pancake tuesday")
	a.t.confirm(context.Background(), old, "pancake tuesday")
	if got := a.t.Standing("old"); got != peer.Foreign {
		t.Errorf("a keyed device sees an old peer as %s, want foreign", got)
	}
}

func TestTheOldRoutesServeOnlyWithoutAKey(t *testing.T) {
	a := newDevice(t, "a", "")
	h := a.t.Legacy(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, c := range []struct {
		key  string
		want int
	}{{"", http.StatusNoContent}, {"pancake tuesday", http.StatusForbidden}} {
		a.setKey(c.key)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/call/offer", nil))
		if rec.Code != c.want {
			t.Errorf("key %q: an old route answered %d, want %d", c.key, rec.Code, c.want)
		}
	}
}

func TestChangingTheKeyForgetsTheHome(t *testing.T) {
	a, b := newDevice(t, "a", "pancake tuesday"), newDevice(t, "b", "pancake tuesday")
	a.t.confirm(context.Background(), b.peer("b"), "pancake tuesday")

	a.setKey("waffle wednesday")
	if got := a.t.Standing("b"); got != peer.Checking {
		t.Errorf("after a key change b is %s, want checking", got)
	}
	if got := ping(t, a, b); got != 0 {
		t.Errorf("a reached b on a forgotten home: %d", got)
	}
}

func TestAFoundPeerIsConfirmedInTheBackground(t *testing.T) {
	a, b := newDevice(t, "a", "pancake tuesday"), newDevice(t, "b", "pancake tuesday")
	b.know(a.peer("a"))
	a.t.Found(b.peer("b"))
	eventually(t, "a confirming b", func() bool { return a.t.Standing("b") == peer.Home })
	eventually(t, "a announcing the change", func() bool { return a.changed() > 0 })
	eventually(t, "b checking a in turn", func() bool { return b.t.Standing("a") == peer.Home })
	eventually(t, "b hearing about a", func() bool { return b.changed() > 0 })
}

func TestALostPeerIsForgotten(t *testing.T) {
	a, b := newDevice(t, "a", "pancake tuesday"), newDevice(t, "b", "pancake tuesday")
	a.t.confirm(context.Background(), b.peer("b"), "pancake tuesday")
	a.t.Lost(b.peer("b"))
	if got := a.t.Standing("b"); got != peer.Checking {
		t.Errorf("a lost peer is %s, want checking", got)
	}
}

func TestANewKeyReconfirmsEveryKnownPeer(t *testing.T) {
	a, b := newDevice(t, "a", "pancake tuesday"), newDevice(t, "b", "pancake tuesday")
	a.know(b.peer("b"))
	a.t.confirm(context.Background(), b.peer("b"), "pancake tuesday")

	b.setKey("waffle wednesday")
	a.setKey("waffle wednesday")
	a.t.Rekey()
	eventually(t, "a reconfirming b", func() bool { return a.t.Standing("b") == peer.Home })
}

func TestAnUnreachablePeerIsRetriedUntilLost(t *testing.T) {
	a := newDevice(t, "a", "")
	gone := peer.Peer{ID: "gone", Addrs: []net.IP{net.ParseIP("127.0.0.1")}, Secure: 1}
	a.t.Found(gone)
	a.t.mu.Lock()
	chasing := a.t.chasing["gone"] != nil
	a.t.mu.Unlock()
	if !chasing {
		t.Fatal("an unreachable peer is not being retried")
	}
	a.t.Lost(gone)
	a.t.mu.Lock()
	chasing = a.t.chasing["gone"] != nil
	a.t.mu.Unlock()
	if chasing {
		t.Error("a lost peer is still being retried")
	}
}

func TestTheIdentityIsKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peer.pem")
	first, err := LoadIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	again, err := LoadIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	if Fingerprint(first.Certificate[0]) != Fingerprint(again.Certificate[0]) {
		t.Error("a second load made a new certificate")
	}
}

func TestADamagedIdentityIsReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "peer.pem")
	if err := os.WriteFile(
		path,
		[]byte("-----BEGIN CERTIFICATE-----\ntruncat"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	cert, err := LoadIdentity(path)
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("no certificate came back")
	}
	again, err := LoadIdentity(path)
	if err != nil {
		t.Fatalf("loading the replacement: %v", err)
	}
	if Fingerprint(cert.Certificate[0]) != Fingerprint(again.Certificate[0]) {
		t.Error("the replacement was not kept")
	}
}
