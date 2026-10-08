package bt

import (
	"encoding/binary"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/bluetooth/avdtp"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/avrcp"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/l2cap"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/sbc"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/sdp"
)

type phone struct {
	t    *testing.T
	sink *Sink

	id    byte
	cid   uint16
	label byte
}

func newPhone(t *testing.T) *phone {
	t.Helper()
	return &phone{t: t, sink: NewSink("Panel", l2cap.MTUDefault), cid: 0x0041}
}

func (p *phone) nextID() byte {
	p.id++
	return p.id
}

func (p *phone) wire(f l2cap.Frame) []l2cap.Frame {
	p.t.Helper()

	sent, err := l2cap.ParseFrame(f.Marshal())
	if err != nil {
		p.t.Fatalf("our own frame did not parse: %v", err)
	}

	out, err := p.sink.Receive(sent)
	if err != nil {
		p.t.Fatalf("the sink refused a frame on channel %#x: %v", f.CID, err)
	}

	for i, r := range out {
		back, err := l2cap.ParseFrame(r.Marshal())
		if err != nil {
			p.t.Fatalf("the sink's answer did not parse: %v", err)
		}
		out[i] = back
	}
	return out
}

func (p *phone) commands(c ...l2cap.Command) []l2cap.Command {
	p.t.Helper()

	frames := p.wire(l2cap.Signal(c...))
	if len(frames) == 0 {
		return nil
	}
	if len(frames) != 1 {
		p.t.Fatalf("%d frames of signalling came back, want one", len(frames))
	}

	out, err := l2cap.ParseCommands(frames[0].Payload)
	if err != nil {
		p.t.Fatalf("the sink's signalling did not parse: %v", err)
	}
	return out
}

type channel struct {
	theirs uint16
	ours   uint16
}

func (p *phone) open(psm uint16) channel {
	p.t.Helper()

	ch := channel{ours: p.cid}
	p.cid++

	answers := p.commands(l2cap.Connect{PSM: psm, SourceCID: ch.ours}.Command(p.nextID()))
	if len(answers) != 2 {
		p.t.Fatalf("connecting to %#x gave %d commands, want an answer and a configuration",
			psm, len(answers))
	}

	connected, err := l2cap.ParseConnected(answers[0])
	if err != nil {
		p.t.Fatalf("ParseConnected: %v", err)
	}
	if connected.Result != l2cap.ConnectSuccess {
		p.t.Fatalf("connecting to %#x was refused: %#x", psm, connected.Result)
	}
	ch.theirs = connected.DestinationCID

	if answers[1].Code != l2cap.CodeConfigRequest {
		p.t.Fatalf("the second command was %#x, want a configuration request", answers[1].Code)
	}
	p.commands(l2cap.Configured{
		SourceCID: ch.theirs,
		Result:    l2cap.ConfigSuccess,
	}.Command(answers[1].ID))

	answers = p.commands(l2cap.Configure{
		DestinationCID: ch.theirs,
		Options:        []l2cap.Option{l2cap.MTUOption(l2cap.MTUDefault)},
	}.Command(p.nextID()))

	if len(answers) != 1 {
		p.t.Fatalf("our configuration got %d answers, want one", len(answers))
	}
	configured, err := l2cap.ParseConfigured(answers[0])
	if err != nil {
		p.t.Fatalf("ParseConfigured: %v", err)
	}
	if configured.Result != l2cap.ConfigSuccess {
		p.t.Fatalf("our configuration was refused: %#x", configured.Result)
	}

	if state := p.sink.l2.Channel(ch.theirs).State; state != l2cap.Open {
		p.t.Fatalf("after configuring both ends the channel is %v", state)
	}
	return ch
}

func (p *phone) close(ch channel) {
	p.t.Helper()

	p.commands(l2cap.Disconnect{
		DestinationCID: ch.theirs,
		SourceCID:      ch.ours,
	}.Command(l2cap.CodeDisconnectRequest, p.nextID()))
}

func (p *phone) data(ch channel, payload []byte) [][]byte {
	p.t.Helper()

	var out [][]byte
	for _, f := range p.wire(l2cap.Frame{CID: ch.theirs, Payload: payload}) {
		if f.CID != ch.ours {
			p.t.Fatalf("an answer came back on channel %#x, want %#x", f.CID, ch.ours)
		}
		out = append(out, f.Payload)
	}
	return out
}

func (p *phone) searchAttribute(class uint16, attrs ...sdp.AttrRange) sdp.PDU {
	p.t.Helper()

	pattern, err := sdp.Sequence(sdp.UUID16(class)).Marshal()
	if err != nil {
		p.t.Fatalf("marshalling the pattern: %v", err)
	}

	elements := make([]sdp.Element, 0, len(attrs))
	for _, a := range attrs {
		elements = append(elements, a.Element())
	}
	list, err := sdp.Sequence(elements...).Marshal()
	if err != nil {
		p.t.Fatalf("marshalling the attribute list: %v", err)
	}

	params := append([]byte(nil), pattern...)
	params = binary.BigEndian.AppendUint16(params, 0xffff)
	params = append(params, list...)
	params = append(params, 0)

	return sdp.PDU{ID: sdp.PDUSearchAttributeRequest, Transaction: 1, Params: params}
}

func (p *phone) ask(ch channel, signal byte, data ...byte) avdtp.Message {
	p.t.Helper()

	m := avdtp.Message{Label: p.label, Type: avdtp.Command, Signal: signal, Data: data}
	p.label = (p.label + 1) & 0x0f

	payloads := p.data(ch, m.Marshal())
	if len(payloads) != 1 {
		p.t.Fatalf("signal %#x got %d answers, want one", signal, len(payloads))
	}

	answer, err := avdtp.ParseMessage(payloads[0])
	if err != nil {
		p.t.Fatalf("the sink's answer did not parse: %v", err)
	}
	if answer.Label != m.Label {
		p.t.Errorf("the answer to %#x came back on label %d, want %d",
			signal, answer.Label, m.Label)
	}
	if answer.Signal != signal {
		p.t.Errorf("asked %#x and was answered %#x", signal, answer.Signal)
	}
	return answer
}

func frame(t *testing.T, bitpool int, fill byte) []byte {
	t.Helper()

	// 44100 is index 2, sixteen blocks is index 3, eight subbands is index 1.
	raw := []byte{
		sbc.Syncword,
		2<<6 | 3<<4 | byte(sbc.JointStereo)<<2 | byte(sbc.Loudness)<<1 | 1,
		byte(bitpool),
		0,
	}

	h, err := sbc.ParseHeader(raw)
	if err != nil {
		t.Fatalf("our own header did not parse: %v", err)
	}

	out := make([]byte, h.Length())
	copy(out, raw)
	for i := 4; i < len(out); i++ {
		out[i] = fill
	}

	crc, err := sbc.CRC(h, out)
	if err != nil {
		t.Fatalf("CRC: %v", err)
	}
	out[3] = crc

	return out
}

func TestAPhoneFindsTheSinkAndPlaysToIt(t *testing.T) {
	p := newPhone(t)

	var started, stopped int
	var decoded [][]byte
	p.sink.Started = func(*avdtp.Endpoint) { started++ }
	p.sink.Stopped = func(*avdtp.Endpoint) { stopped++ }
	p.sink.Frame = func(f []byte) { decoded = append(decoded, f) }

	discovery := p.open(l2cap.PSMSDP)

	request := p.searchAttribute(sdp.UUIDAudioSink, sdp.AttrRange{First: 0x0000, Last: 0xffff})
	answers := p.data(discovery, request.Marshal())
	if len(answers) != 1 {
		t.Fatalf("the search got %d answers, want one", len(answers))
	}

	response, err := sdp.ParsePDU(answers[0])
	if err != nil {
		t.Fatalf("ParsePDU: %v", err)
	}
	if response.ID != sdp.PDUSearchAttributeResponse {
		t.Fatalf("the search was answered with %#x", response.ID)
	}

	psm := psmFrom(t, response)
	if psm != SinkPSM {
		t.Fatalf("the record advertises channel %#x, the stack serves %#x", psm, SinkPSM)
	}

	signalling := p.open(psm)

	seps, err := avdtp.ParseSEPs(p.ask(signalling, avdtp.SignalDiscover).Data)
	if err != nil {
		t.Fatalf("ParseSEPs: %v", err)
	}
	if len(seps) != 1 {
		t.Fatalf("%d endpoints, want one", len(seps))
	}
	if seps[0].TSEP != avdtp.Sink || seps[0].Media != avdtp.MediaAudio {
		t.Fatalf("the endpoint is %+v, want an audio sink", seps[0])
	}
	seid := seps[0].SEID

	caps := p.ask(signalling, avdtp.SignalGetCapabilities, avdtp.AddressTo(seid))
	if caps.Type != avdtp.ResponseAccept {
		t.Fatalf("get capabilities was refused: %v", caps.Data)
	}

	offered, err := avdtp.ParseCapabilities(caps.Data)
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	codec, ok := avdtp.Find(offered, avdtp.CatMediaCodec)
	if !ok {
		t.Fatal("the sink offered no codec")
	}
	offer, err := avdtp.ParseSBC(codec)
	if err != nil {
		t.Fatalf("ParseSBC: %v", err)
	}

	chosen := avdtp.SBC{
		Rates:      avdtp.Rate48000,
		Channels:   avdtp.ChannelJointStereo,
		Blocks:     avdtp.Block16,
		Subbands:   avdtp.Subbands8,
		Allocation: avdtp.AllocationLoudness,
		MinBitpool: offer.MinBitpool,
		MaxBitpool: offer.MaxBitpool,
	}

	config := append([]byte{avdtp.AddressTo(seid), avdtp.AddressTo(2)},
		avdtp.MarshalCapabilities([]avdtp.Capability{
			{Category: avdtp.CatMediaTransport},
			chosen.Capability(),
		})...)

	if m := p.ask(signalling, avdtp.SignalSetConfig, config...); m.Type != avdtp.ResponseAccept {
		t.Fatalf("the configuration was refused: %v", m.Data)
	}
	if m := p.ask(
		signalling,
		avdtp.SignalOpen,
		avdtp.AddressTo(seid),
	); m.Type != avdtp.ResponseAccept {
		t.Fatalf("open was refused: %v", m.Data)
	}

	transport := p.open(SinkPSM)
	if p.sink.transport != transport.theirs {
		t.Fatalf("the sink took channel %#x for media, want %#x",
			p.sink.transport, transport.theirs)
	}
	if p.sink.signalling != signalling.theirs {
		t.Fatalf("the sink took channel %#x for signalling, want %#x",
			p.sink.signalling, signalling.theirs)
	}

	if m := p.ask(
		signalling,
		avdtp.SignalStart,
		avdtp.AddressTo(seid),
	); m.Type != avdtp.ResponseAccept {
		t.Fatalf("start was refused: %v", m.Data)
	}
	if started != 1 {
		t.Fatalf("the stream started %d times", started)
	}

	first := frame(t, 35, 0x5a)
	second := frame(t, 35, 0xa5)

	media := avdtp.Media{
		Sequence: 1,
		Count:    2,
		Payload:  append(append([]byte(nil), first...), second...),
	}
	if out := p.data(transport, media.Marshal()); out != nil {
		t.Errorf("the sink answered a media packet with %d frames", len(out))
	}

	if len(decoded) != 2 {
		t.Fatalf("%d frames came out of one packet holding two", len(decoded))
	}

	for i, want := range [][]byte{first, second} {
		if string(decoded[i]) != string(want) {
			t.Errorf("frame %d came out %d bytes, want %d", i, len(decoded[i]), len(want))
		}
	}

	for i, f := range decoded {
		if _, err := sbc.Unpack(f); err != nil {
			t.Errorf("frame %d did not decode: %v", i, err)
		}
	}

	if m := p.ask(
		signalling,
		avdtp.SignalSuspend,
		avdtp.AddressTo(seid),
	); m.Type != avdtp.ResponseAccept {
		t.Fatalf("suspend was refused: %v", m.Data)
	}
	if stopped != 1 {
		t.Errorf("the stream stopped %d times", stopped)
	}
}

func psmFrom(t *testing.T, p sdp.PDU) uint16 {
	t.Helper()

	// A length, then the outer sequence of attribute lists, then the continuation.
	if len(p.Params) < 3 {
		t.Fatalf("the response is %d bytes", len(p.Params))
	}
	body := p.Params[2 : 2+int(binary.BigEndian.Uint16(p.Params))]

	outer, _, err := sdp.ParseElement(body)
	if err != nil {
		t.Fatalf("the response body: %v", err)
	}
	if len(outer.Children) != 1 {
		t.Fatalf("%d records came back, want one", len(outer.Children))
	}

	// Each record is a flat run of identifier and value.
	attrs := outer.Children[0].Children
	for i := 0; i+1 < len(attrs); i += 2 {
		id, ok := attrs[i].Uint()
		if !ok || uint16(id) != sdp.AttrProtocols {
			continue
		}
		psm, ok := sdp.PSM(attrs[i+1])
		if !ok {
			t.Fatal("the protocol list carries no channel number")
		}
		return psm
	}

	t.Fatal("the record came back with no protocol list")
	return 0
}

func TestThePhoneOpensTheRemoteControlAndPressesPlay(t *testing.T) {
	p := newPhone(t)

	var state avrcp.State
	p.sink.Remote().Changed = func(s avrcp.State) { state = s }

	if p.sink.Controlling() {
		t.Fatal("the remote control is up before anything opened it")
	}

	control := p.open(ControlPSM)
	if !p.sink.Controlling() {
		t.Fatal("the remote control channel did not come up")
	}
	if p.sink.control != control.theirs {
		t.Fatalf("the sink took channel %#x for control, want %#x",
			p.sink.control, control.theirs)
	}

	frames, err := p.sink.Press(avrcp.OpPlay)
	if err != nil {
		t.Fatalf("Press: %v", err)
	}
	if len(frames) != 2 {
		t.Fatalf("%d frames, want a press and a release", len(frames))
	}

	for i, want := range []bool{avrcp.Pressed, avrcp.Released} {
		if frames[i].CID != control.ours {
			t.Errorf("frame %d went to channel %#x, want %#x", i, frames[i].CID, control.ours)
		}

		m, err := avrcp.ParseTransport(frames[i].Payload)
		if err != nil {
			t.Fatalf("frame %d did not parse: %v", i, err)
		}
		f, err := avrcp.ParseAVC(m.Payload)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}

		op, released, err := avrcp.Button(f)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if op != avrcp.OpPlay || released != want {
			t.Errorf("frame %d is %#02x released=%v", i, op, released)
		}
	}

	command := avrcp.Transport{
		Label:   4,
		Type:    avrcp.MessageCommand,
		Payload: avrcp.SetAbsoluteVolume(avrcp.Volume(60)).Frame(avrcp.Control).Marshal(),
	}

	answers := p.data(control, command.Marshal())
	if len(answers) != 1 {
		t.Fatalf("%d answers to a volume command, want one", len(answers))
	}

	back, err := avrcp.ParseTransport(answers[0])
	if err != nil {
		t.Fatalf("the answer did not parse: %v", err)
	}
	if back.Type != avrcp.MessageResponse || back.Label != 4 {
		t.Errorf("answered as %+v", back)
	}
	if state.Volume < 59 || state.Volume > 61 {
		t.Errorf("the volume came through as %d%%, want about 60", state.Volume)
	}
}

func TestASearchForTheSinkMatchesOnlyTheSink(t *testing.T) {
	p := newPhone(t)
	discovery := p.open(l2cap.PSMSDP)

	request := p.searchAttribute(sdp.UUIDAudioSink, sdp.AttrRange{First: 0x0000, Last: 0xffff})
	answers := p.data(discovery, request.Marshal())

	response, err := sdp.ParsePDU(answers[0])
	if err != nil {
		t.Fatalf("ParsePDU: %v", err)
	}
	if got := len(recordsIn(t, response)); got != 1 {
		t.Errorf("%d records came back for the audio sink alone", got)
	}
}

func TestSearchingTheBrowseGroupFindsEverything(t *testing.T) {
	p := newPhone(t)
	discovery := p.open(l2cap.PSMSDP)

	request := p.searchAttribute(sdp.UUIDBrowseRoot, sdp.AttrRange{First: 0x0000, Last: 0xffff})
	answers := p.data(discovery, request.Marshal())

	response, err := sdp.ParsePDU(answers[0])
	if err != nil {
		t.Fatalf("ParsePDU: %v", err)
	}
	if got := len(recordsIn(t, response)); got != 3 {
		t.Errorf("%d records came back, want the sink and both ends of the remote control", got)
	}
}

func recordsIn(t *testing.T, p sdp.PDU) []sdp.Element {
	t.Helper()

	if len(p.Params) < 3 {
		t.Fatalf("the response is %d bytes", len(p.Params))
	}
	body := p.Params[2 : 2+int(binary.BigEndian.Uint16(p.Params))]

	outer, _, err := sdp.ParseElement(body)
	if err != nil {
		t.Fatalf("the response body: %v", err)
	}
	return outer.Children
}

func TestTheRemoteControlRecordsNameTheControlChannel(t *testing.T) {
	records := Records("Panel")
	if len(records) != 3 {
		t.Fatalf("%d records", len(records))
	}

	for _, r := range records[1:] {
		protocols, ok := r.Attribute(sdp.AttrProtocols)
		if !ok {
			t.Fatal("a remote control record has no protocol list")
		}

		psm, ok := sdp.PSM(protocols)
		if !ok {
			t.Fatal("a remote control record carries no channel number")
		}
		if psm != ControlPSM {
			t.Errorf("a remote control record advertises %#x, want %#x", psm, ControlPSM)
		}
	}
}

func TestTheRemoteControlRecordsAgreeOnTheirCategories(t *testing.T) {
	for _, r := range Records("Panel")[1:] {
		classes, ok := r.Attribute(sdp.AttrServiceClasses)
		if !ok {
			t.Fatal("a remote control record has no service classes")
		}

		generic := false
		for _, c := range classes.Children {
			if v, ok := c.Uint(); ok && v == sdp.UUIDAVRemoteControl {
				generic = true
			}
		}
		if !generic {
			t.Error("a remote control record does not carry the generic class")
		}

		features, ok := r.Attribute(sdp.AttrFeatures)
		if !ok {
			t.Fatal("a remote control record has no supported features")
		}

		v, ok := features.Uint()
		if !ok {
			t.Fatal("supported features is not a number")
		}

		if v&RemoteFeatures != RemoteFeatures {
			t.Errorf("a remote control record claims %#04x, missing a category from %#04x",
				v, uint32(RemoteFeatures))
		}
	}
}

func TestEveryRecordHasItsOwnHandle(t *testing.T) {
	seen := map[uint32]bool{}

	for _, r := range Records("Panel") {
		handle, ok := r.Attribute(sdp.AttrRecordHandle)
		if !ok {
			t.Fatal("a record has no handle")
		}

		v, ok := handle.Uint()
		if !ok {
			t.Fatal("a handle is not a number")
		}
		if seen[v] {
			t.Errorf("handle %#x is used twice", v)
		}
		seen[v] = true
	}
}

func TestTheRecordAdvertisesTheChannelTheStackServes(t *testing.T) {
	protocols, ok := Record("Panel").Attribute(sdp.AttrProtocols)
	if !ok {
		t.Fatal("the record has no protocol list")
	}

	psm, ok := sdp.PSM(protocols)
	if !ok {
		t.Fatal("the protocol list carries no channel number")
	}
	if psm != SinkPSM {
		t.Errorf("the record advertises %#x and the stack serves %#x", psm, SinkPSM)
	}
}

func TestASearchForSomethingElseMatchesNothing(t *testing.T) {
	p := newPhone(t)
	discovery := p.open(l2cap.PSMSDP)

	request := p.searchAttribute(0x1108, sdp.AttrRange{First: 0x0000, Last: 0xffff})

	answers := p.data(discovery, request.Marshal())
	if len(answers) != 1 {
		t.Fatalf("%d answers", len(answers))
	}

	response, err := sdp.ParsePDU(answers[0])
	if err != nil {
		t.Fatalf("ParsePDU: %v", err)
	}
	if response.ID != sdp.PDUSearchAttributeResponse {
		t.Fatalf("answered with %#x", response.ID)
	}

	body := response.Params[2 : 2+int(binary.BigEndian.Uint16(response.Params))]
	outer, _, err := sdp.ParseElement(body)
	if err != nil {
		t.Fatalf("the response body: %v", err)
	}
	if len(outer.Children) != 0 {
		t.Errorf("%d records came back for a service this device does not offer",
			len(outer.Children))
	}
}

func TestAnUnservedChannelIsRefused(t *testing.T) {
	p := newPhone(t)

	answers := p.commands(l2cap.Connect{PSM: 0x0003, SourceCID: 0x0041}.Command(p.nextID()))
	if len(answers) != 1 {
		t.Fatalf("%d commands came back, want one refusal", len(answers))
	}

	connected, err := l2cap.ParseConnected(answers[0])
	if err != nil {
		t.Fatalf("ParseConnected: %v", err)
	}
	if connected.Result != l2cap.ConnectBadPSM {
		t.Errorf("refused with %#x, want the one that names the channel", connected.Result)
	}
}

func TestDataOnAChannelThatIsNotOpen(t *testing.T) {
	p := newPhone(t)

	if _, err := p.sink.Receive(l2cap.Frame{CID: 0x0050, Payload: []byte{1, 2}}); err == nil {
		t.Error("data on an unopened channel was accepted")
	}
}

func TestASessionEndingDropsWhatThisEndOpened(t *testing.T) {
	p := newPhone(t)

	control := p.open(ControlPSM)
	browse := p.open(BrowsePSM)

	if p.sink.control == 0 || p.sink.browse == 0 {
		t.Fatalf("control %#x and browsing %#x, want both", p.sink.control, p.sink.browse)
	}

	p.close(control)

	if p.sink.control != 0 || p.sink.browse != 0 {
		t.Errorf("the session ended and control is %#x, browsing %#x",
			p.sink.control, p.sink.browse)
	}
	if p.sink.l2.Channel(browse.theirs) != nil {
		t.Error("the browsing channel outlived the session it belonged to")
	}
}

func TestANewSessionTakesTheNewStreamChannels(t *testing.T) {
	p := newPhone(t)

	first := p.open(SinkPSM)
	if p.sink.signalling != first.theirs {
		t.Fatalf("signalling is %#x, want %#x", p.sink.signalling, first.theirs)
	}

	p.close(first)
	if p.sink.signalling != 0 {
		t.Errorf("a closed channel is still the stream: %#x", p.sink.signalling)
	}

	second := p.open(SinkPSM)
	if p.sink.signalling != second.theirs {
		t.Errorf("signalling is %#x, want the new channel %#x", p.sink.signalling, second.theirs)
	}
}

func TestForgettingALink(t *testing.T) {
	p := newPhone(t)
	p.open(l2cap.PSMSDP)
	signalling := p.open(SinkPSM)

	p.ask(signalling, avdtp.SignalDiscover)
	p.sink.Forget()

	if got := p.sink.l2.Open(); len(got) != 0 {
		t.Errorf("%d channels survived", len(got))
	}
	if p.sink.signalling != 0 || p.sink.transport != 0 {
		t.Errorf("the avdtp channels survived: %#x and %#x", p.sink.signalling, p.sink.transport)
	}
	if e := p.sink.Endpoint(); e.State != avdtp.Idle {
		t.Errorf("the endpoint is %v, want idle", e.State)
	}
}
