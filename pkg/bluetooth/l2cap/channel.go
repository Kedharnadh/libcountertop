package l2cap

import "fmt"

type State int

const (
	Closed State = iota

	Connecting

	Configuring

	Open

	Closing
)

func (s State) String() string {
	switch s {
	case Connecting:
		return "connecting"
	case Configuring:
		return "configuring"
	case Open:
		return "open"
	case Closing:
		return "closing"
	}
	return "closed"
}

type Channel struct {
	Local  uint16
	Remote uint16

	PSM   uint16
	State State

	Ours bool

	MTU uint16

	theirs bool
	ours   bool
}

type Manager struct {
	channels map[uint16]*Channel
	serving  map[uint16]bool

	asking map[byte]uint16

	next uint16
	id   byte

	mtu uint16
}

func NewManager(mtu uint16) *Manager {
	if mtu == 0 {
		mtu = MTUDefault
	}
	return &Manager{
		channels: map[uint16]*Channel{},
		serving:  map[uint16]bool{},
		asking:   map[byte]uint16{},
		next:     CIDDynamic,
		mtu:      mtu,
	}
}

func (m *Manager) Connect(psm uint16) (Command, error) {
	for _, c := range m.channels {
		if c.PSM == psm && c.State != Closed && c.State != Closing {
			return Command{}, fmt.Errorf("l2cap: channel %#x to %#04x is already %s",
				c.Local, psm, c.State)
		}
	}

	local, err := m.allocate()
	if err != nil {
		return Command{}, err
	}

	m.channels[local] = &Channel{
		Local: local,
		PSM:   psm,
		State: Connecting,
		MTU:   MTUDefault,
		Ours:  true,
	}

	id := m.nextID()
	m.asking[id] = local

	return Connect{PSM: psm, SourceCID: local}.Command(id), nil
}

func (m *Manager) Serve(psm uint16) { m.serving[psm] = true }

func (m *Manager) Channel(cid uint16) *Channel { return m.channels[cid] }

func (m *Manager) Open() []*Channel {
	var out []*Channel
	for _, c := range m.channels {
		if c.State == Open {
			out = append(out, c)
		}
	}
	return out
}

func (m *Manager) allocate() (uint16, error) {
	for range 0xffff - CIDDynamic {
		cid := m.next
		m.next++
		if m.next < CIDDynamic {
			m.next = CIDDynamic
		}

		if m.channels[cid] == nil {
			return cid, nil
		}
	}
	return 0, fmt.Errorf("l2cap: no channel numbers left")
}

// Zero is not a valid identifier.
func (m *Manager) nextID() byte {
	m.id++
	if m.id == 0 {
		m.id = 1
	}
	return m.id
}

func (m *Manager) Handle(f Frame) ([]Command, error) {
	if f.CID != CIDSignalling {
		return nil, fmt.Errorf("l2cap: %#x is not the signalling channel", f.CID)
	}

	commands, err := ParseCommands(f.Payload)
	if err != nil {
		return nil, err
	}

	var out []Command
	for _, c := range commands {
		answers, err := m.one(c)
		if err != nil {
			return out, err
		}
		out = append(out, answers...)
	}
	return out, nil
}

func (m *Manager) one(c Command) ([]Command, error) {
	switch c.Code {
	case CodeConnectRequest:
		return m.connect(c)
	case CodeConfigRequest:
		return m.configure(c)
	case CodeConfigResponse:
		return m.configured(c)
	case CodeDisconnectRequest:
		return m.disconnect(c)
	case CodeEchoRequest:
		return []Command{{Code: CodeEchoResponse, ID: c.ID}}, nil

	case CodeInfoRequest:
		// "Not supported" is a legal information response.
		if len(c.Data) < 2 {
			return []Command{Reject(c.ID, RejectNotUnderstood)}, nil
		}
		data := []byte{c.Data[0], c.Data[1], 0x01, 0x00} // type, result: not supported
		return []Command{{Code: CodeInfoResponse, ID: c.ID, Data: data}}, nil

	case CodeConnectResponse:
		return m.connected(c)

	case CodeDisconnectResponse, CodeEchoResponse, CodeInfoResponse:
		return nil, nil
	}

	return []Command{Reject(c.ID, RejectNotUnderstood)}, nil
}

func (m *Manager) connect(c Command) ([]Command, error) {
	req, err := ParseConnect(c)
	if err != nil {
		return []Command{Reject(c.ID, RejectNotUnderstood)}, nil
	}

	if !m.serving[req.PSM] {
		return []Command{Connected{
			SourceCID: req.SourceCID,
			Result:    ConnectBadPSM,
		}.Command(c.ID)}, nil
	}

	local, err := m.allocate()
	if err != nil {
		return []Command{Connected{
			SourceCID: req.SourceCID,
			Result:    ConnectNoResources,
		}.Command(c.ID)}, nil
	}

	ch := &Channel{
		Local:  local,
		Remote: req.SourceCID,
		PSM:    req.PSM,
		State:  Configuring,
		MTU:    MTUDefault,
	}
	m.channels[local] = ch

	return []Command{
		Connected{
			DestinationCID: local,
			SourceCID:      ch.Remote,
			Result:         ConnectSuccess,
		}.Command(
			c.ID,
		),
		Configure{
			DestinationCID: ch.Remote,
			Options:        []Option{MTUOption(m.mtu)},
		}.Command(m.nextID()),
	}, nil
}

// In a connection response the destination CID is theirs and the source ours.
func (m *Manager) connected(c Command) ([]Command, error) {
	res, err := ParseConnected(c)
	if err != nil {
		return nil, err
	}

	local, waiting := m.asking[c.ID]
	if !waiting {
		return nil, nil
	}

	ch := m.channels[local]
	if ch == nil || ch.State != Connecting {
		delete(m.asking, c.ID)
		return nil, nil
	}

	// Pending: another response follows on the same identifier.
	if res.Result == ConnectPending {
		return nil, nil
	}
	delete(m.asking, c.ID)

	if res.Result != ConnectSuccess {
		delete(m.channels, local)
		return nil, fmt.Errorf("l2cap: %#04x refused a channel: %#04x", ch.PSM, res.Result)
	}

	if res.SourceCID != local {
		delete(m.channels, local)
		return nil, fmt.Errorf("l2cap: an answer for channel %#x arrived on the request for %#x",
			res.SourceCID, local)
	}

	ch.Remote = res.DestinationCID
	ch.State = Configuring

	return []Command{Configure{
		DestinationCID: ch.Remote,
		Options:        []Option{MTUOption(m.mtu)},
	}.Command(m.nextID())}, nil
}

func (m *Manager) configure(c Command) ([]Command, error) {
	req, err := ParseConfigure(c)
	if err != nil {
		return []Command{Reject(c.ID, RejectNotUnderstood)}, nil
	}

	ch := m.channels[req.DestinationCID]
	if ch == nil {
		return []Command{Reject(c.ID, RejectBadCID, req.DestinationCID, 0)}, nil
	}

	if mode, asked := req.Mode(); asked && mode != ModeBasic {
		return []Command{Configured{
			SourceCID: ch.Remote,
			Result:    ConfigUnacceptable,
			Options:   []Option{BasicMode()},
		}.Command(c.ID)}, nil
	}

	if unknown := req.Unknown(); len(unknown) > 0 {
		options := make([]Option, 0, len(unknown))
		for _, t := range unknown {
			options = append(options, Option{Type: t})
		}
		return []Command{Configured{
			SourceCID: ch.Remote,
			Result:    ConfigUnknown,
			Options:   options,
		}.Command(c.ID)}, nil
	}

	ch.MTU = req.MTU()

	if req.Continues() {
		return []Command{Configured{
			SourceCID: ch.Remote,
			Flags:     ConfigFlagContinues,
			Result:    ConfigSuccess,
		}.Command(c.ID)}, nil
	}

	ch.theirs = true
	m.settle(ch)

	return []Command{Configured{
		SourceCID: ch.Remote,
		Result:    ConfigSuccess,
	}.Command(c.ID)}, nil
}

func (m *Manager) configured(c Command) ([]Command, error) {
	res, err := ParseConfigured(c)
	if err != nil {
		return nil, nil
	}

	// A configuration response names our CID.
	ch := m.channels[res.SourceCID]
	if ch == nil {
		return nil, nil
	}

	if res.Result != ConfigSuccess {
		return nil, fmt.Errorf("l2cap: channel %#x refused our configuration: %#x",
			ch.Local, res.Result)
	}
	if res.Flags&ConfigFlagContinues != 0 {
		return nil, nil
	}

	ch.ours = true
	m.settle(ch)
	return nil, nil
}

func (m *Manager) settle(ch *Channel) {
	if ch.theirs && ch.ours && ch.State == Configuring {
		ch.State = Open
	}
}

func (m *Manager) disconnect(c Command) ([]Command, error) {
	req, err := ParseDisconnect(c)
	if err != nil {
		return []Command{Reject(c.ID, RejectNotUnderstood)}, nil
	}

	ch := m.channels[req.DestinationCID]
	if ch == nil {
		return []Command{Reject(c.ID, RejectBadCID, req.DestinationCID, req.SourceCID)}, nil
	}

	delete(m.channels, ch.Local)

	return []Command{Disconnect{
		DestinationCID: req.DestinationCID,
		SourceCID:      req.SourceCID,
	}.Command(CodeDisconnectResponse, c.ID)}, nil
}

func (m *Manager) Drop(local uint16) { delete(m.channels, local) }

func (m *Manager) Forget() {
	clear(m.channels)
	clear(m.asking)
}
