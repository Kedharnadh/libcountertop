package cast

import (
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func txt(t *testing.T, d Device) map[string]string {
	t.Helper()

	out := map[string]string{}
	for _, r := range d.Records() {
		k, v, ok := strings.Cut(r, "=")
		if !ok {
			t.Fatalf("%q is not a key and a value", r)
		}
		if _, seen := out[k]; seen {
			t.Errorf("%q appears twice", k)
		}
		out[k] = v
	}
	return out
}

func TestASpeakerClaimsWhatAHomeMiniClaims(t *testing.T) {
	d := Identify("00:f4:8d:47:69:21", "Kitchen", "Echo Dot", SmartSpeaker)
	if got := txt(t, d)["ca"]; got != "199172" {
		t.Errorf("the record's capabilities are %q, want a speaker's", got)
	}

	r := NewReceiver("Kitchen")
	r.Identity = func() Device { return d }
	out, err := r.Receive(from(SenderID, NSDiscovery, `{"type":"GET_DEVICE_INFO","requestId":1}`))
	if err != nil || len(out) != 1 {
		t.Fatalf("%d answers, %v", len(out), err)
	}
	var info map[string]any
	if err := json.Unmarshal([]byte(out[0].Payload), &info); err != nil {
		t.Fatal(err)
	}
	if info["deviceCapabilities"] != float64(198660) {
		t.Errorf("DEVICE_INFO capabilities are %v, want a speaker's", info["deviceCapabilities"])
	}

	if NewEureka(d, "Amazon", "1").DeviceInfo.Capabilities.DisplaySupported {
		t.Error("eureka_info says a speaker has a display")
	}
}

func TestTheRecordSaysEverythingASenderReads(t *testing.T) {
	d := Identify("00:f4:8d:47:69:21", "Kitchen", "Lenovo Smart Display 10", SmartDisplay)

	got := txt(t, d)
	for _, want := range []string{"id", "cd", "ve", "md", "fn", "ca", "st", "bs", "nf", "rs"} {
		if _, ok := got[want]; !ok {
			t.Errorf("the record has no %q", want)
		}
	}

	if got["fn"] != "Kitchen" {
		t.Errorf("the name is %q", got["fn"])
	}
	if got["md"] != "Lenovo Smart Display 10" {
		t.Errorf("the model is %q", got["md"])
	}
	if got["ca"] != "231941" {
		t.Errorf("the capabilities are %q, want what a smart display advertises", got["ca"])
	}
}

func TestTheIdIsStableForAnAddress(t *testing.T) {
	first := Identify("00:f4:8d:47:69:21", "Kitchen", "Model", SmartDisplay)
	again := Identify("00:F4:8D:47:69:21", "Renamed", "Other", SmartDisplay)

	if first.ID != again.ID {
		t.Errorf("the id changed with the case of the address: %q and %q", first.ID, again.ID)
	}

	other := Identify("00:f4:8d:47:69:22", "Kitchen", "Model", SmartDisplay)
	if other.ID == first.ID {
		t.Error("two addresses gave the same id")
	}
}

func TestTheIdIsThirtyTwoHexCharacters(t *testing.T) {
	d := Identify("00:f4:8d:47:69:21", "Kitchen", "Model", SmartDisplay)

	if len(d.ID) != 32 {
		t.Fatalf("the id is %d characters", len(d.ID))
	}
	if _, err := hex.DecodeString(d.ID); err != nil {
		t.Errorf("the id is not hexadecimal: %q", d.ID)
	}
}

func TestTheDerivedIdentifiersAreNotEachOther(t *testing.T) {
	d := Identify("00:f4:8d:47:69:21", "Kitchen", "Model", SmartDisplay)
	got := txt(t, d)

	if got["id"] == got["cd"] {
		t.Error("the device id and the cloud id are the same number")
	}
	if strings.EqualFold(got["id"][:12], got["bs"]) {
		t.Error("the bootstrap id is the front of the device id")
	}
}

func TestTheRecordSaysWhetherSomethingIsRunning(t *testing.T) {
	d := Identify("00:f4:8d:47:69:21", "Kitchen", "Model", SmartDisplay)

	if got := txt(t, d)["st"]; got != "0" {
		t.Errorf("an idle device says st=%q", got)
	}

	d.Running = true
	d.Status = "Sun Ra"

	got := txt(t, d)
	if got["st"] != "1" {
		t.Errorf("a busy device says st=%q", got["st"])
	}
	if got["rs"] != "Sun Ra" {
		t.Errorf("the status line is %q", got["rs"])
	}
}

func TestTheInstanceNameIsTheId(t *testing.T) {
	d := Identify("00:f4:8d:47:69:21", "Kitchen Speaker!", "Model", SmartDisplay)

	if d.Instance() != d.ID {
		t.Errorf("the instance is %q", d.Instance())
	}
}

func TestWhatWouldNotBeOffered(t *testing.T) {
	good := Identify("00:f4:8d:47:69:21", "Kitchen", "Model", SmartDisplay)
	if err := good.Valid(); err != nil {
		t.Fatalf("a good device was refused: %v", err)
	}

	for _, tc := range []struct {
		name   string
		change func(*Device)
	}{
		{"no id", func(d *Device) { d.ID = "" }},
		{"a short id", func(d *Device) { d.ID = "abcd" }},
		{"an id that is not hex", func(d *Device) { d.ID = strings.Repeat("z", 32) }},
		{"no name", func(d *Device) { d.Name = "" }},
		{"no capabilities", func(d *Device) { d.Profile = Profile{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := good
			tc.change(&d)

			if err := d.Valid(); err == nil {
				t.Error("was accepted")
			}
		})
	}
}

func TestASenderConnectsOverTLSAndIsAnswered(t *testing.T) {
	creds := issued(t)

	listener, err := tls.Listen("tcp", "127.0.0.1:0", TLS(func() *Credentials { return creds }))
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	s := NewService(NewReceiver("Kitchen"))
	s.Receiver.Player = &player{}

	go func() {
		for {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			go func() { _ = s.Serve(NewConn(c)) }()
		}
	}()

	conn, err := tls.Dial("tcp", listener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	phone := &sender{t: t, conn: conn}

	phone.send(from(SenderID, NSConnection, Connect()))
	phone.send(from(SenderID, NSReceiver, `{"type":"GET_STATUS","requestId":1}`))

	got := phone.expect()
	h, err := Kind(got.Payload)
	if err != nil {
		t.Fatalf("the answer did not parse: %v", err)
	}
	if h.Type != TypeStatus || h.RequestID != 1 {
		t.Errorf("came back %+v", h)
	}

	phone.send(from(SenderID, NSReceiver,
		`{"type":"LAUNCH","requestId":2,"appId":"`+DefaultMediaReceiver+`"}`))

	launched := phone.expect()
	phone.expect()
	transport := status(t, launched.Payload).Applications[0].TransportID

	long := strings.Repeat("a very long track title ", 400)
	phone.send(toApp(SenderID, transport, NSMedia,
		`{"type":"LOAD","requestId":3,"media":{"contentId":"http://example/a.mp3",`+
			`"metadata":{"metadataType":3,"title":"`+long+`"}}}`))

	loaded := mediaStatus(t, phone.expect().Payload)
	if len(loaded) != 1 {
		t.Fatalf("%d statuses", len(loaded))
	}
	if loaded[0].Media == nil || loaded[0].Media.Metadata.Title != long {
		t.Error("a message larger than one read did not come back whole")
	}
}

func TestOlderSendersCanStillNegotiate(t *testing.T) {
	creds := issued(t)

	listener, err := tls.Listen("tcp", "127.0.0.1:0", TLS(func() *Credentials { return creds }))
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	// crypto/tls handshakes lazily, on the first read or write.
	go func() {
		c, err := listener.Accept()
		if err != nil {
			return
		}
		defer c.Close()

		if tc, ok := c.(*tls.Conn); ok {
			_ = tc.Handshake()
		}
	}()

	conn, err := tls.Dial("tcp", listener.Addr().String(), &tls.Config{
		InsecureSkipVerify: true,
		MaxVersion:         tls.VersionTLS12,
	})
	if err != nil {
		t.Fatalf("a sender limited to tls 1.2 could not connect: %v", err)
	}
	_ = conn.Close()
}

func TestNoClientCertificateIsAskedFor(t *testing.T) {
	if got := TLS(func() *Credentials { return nil }).ClientAuth; got != tls.NoClientCert {
		t.Errorf("the server asks for %v", got)
	}
}

func TestThePortIsTheOneSendersUse(t *testing.T) {
	if Port != 8009 {
		t.Errorf("the port is %d", Port)
	}
	if ServiceType != "_googlecast._tcp" {
		t.Errorf("the service is %q", ServiceType)
	}
}

func TestARecordValueMayContainAnEqualsSign(t *testing.T) {
	d := Identify("00:f4:8d:47:69:21", "Kitchen", "Model", SmartDisplay)
	d.Status = "playing a=b"

	if got := txt(t, d)["rs"]; got != "playing a=b" {
		t.Errorf("the status came back %q", got)
	}
}
