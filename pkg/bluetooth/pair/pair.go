package pair

import (
	"encoding/binary"
	"fmt"
	"maps"
	"slices"
)

type Store interface {
	Key(Addr) (Key, bool)

	Save(Addr, Key) error

	Forget(Addr) error
}

type Memory map[Addr]Key

func (m Memory) Key(a Addr) (Key, bool) { k, ok := m[a]; return k, ok }

func (m Memory) Save(a Addr, k Key) error {
	m[a] = k
	return nil
}

func (m Memory) Forget(a Addr) error {
	delete(m, a)
	return nil
}

type Peer struct {
	Addr Addr

	// Zero is a valid handle.
	Handle    uint16
	Connected bool

	Bonded bool

	Encrypted bool
}

type Manager struct {
	Keys Store

	IO byte

	Auth byte

	Accept func(Addr) bool

	Confirm func(Addr, uint32) bool

	Bonded func(*Peer)
	Lost   func(*Peer)

	peers map[Addr]*Peer
}

func New(keys Store) *Manager {
	if keys == nil {
		keys = Memory{}
	}
	return &Manager{
		Keys:  keys,
		IO:    NoInputNoOutput,
		Auth:  GeneralBonding,
		peers: map[Addr]*Peer{},
	}
}

func (m *Manager) Peer(a Addr) *Peer { return m.peers[a] }

func (m *Manager) Peers() []*Peer {
	out := slices.Collect(maps.Values(m.peers))
	slices.SortFunc(out, func(a, b *Peer) int {
		return slices.Compare(a.Addr[:], b.Addr[:])
	})
	return out
}

func (m *Manager) Connected(handle uint16) *Peer {
	for _, p := range m.peers {
		if p.Connected && p.Handle == handle {
			return p
		}
	}
	return nil
}

func (m *Manager) peer(a Addr) *Peer {
	p := m.peers[a]
	if p == nil {
		p = &Peer{Addr: a}
		m.peers[a] = p
	}
	return p
}

func (m *Manager) Handle(e Event) ([]Command, error) {
	switch e.Code {
	case EventConnectionRequest:
		return m.requested(e)
	case EventConnectionComplete:
		return nil, m.connected(e)
	case EventDisconnectionComplete:
		return nil, m.disconnected(e)
	case EventLinkKeyRequest:
		return m.linkKey(e)
	case EventLinkKeyNotification:
		return nil, m.remember(e)
	case EventIOCapabilityRequest:
		return m.capability(e)
	case EventUserConfirmation:
		return m.confirm(e)
	case EventPINCodeRequest:
		return m.legacy(e)
	case EventUserPasskeyRequest:
		return m.passkey(e)
	case EventSimplePairingComplete:
		return nil, m.paired(e)
	case EventEncryptionChange:
		return nil, m.encrypted(e)
	}
	return nil, nil
}

func (m *Manager) requested(e Event) ([]Command, error) {
	// Address, class of device, link type.
	if len(e.Params) < 10 {
		return nil, fmt.Errorf("pair: a connection request of %d bytes", len(e.Params))
	}

	a, _ := ParseAddr(e.Params)

	if link := e.Params[9]; link != LinkACL {
		return []Command{rejectWith(a, RejectUnacceptableAddr)}, nil
	}

	if m.Accept != nil && !m.Accept(a) {
		return []Command{rejectWith(a, RejectSecurity)}, nil
	}

	m.peer(a)

	// Role 0x01 is staying the peripheral.
	return []Command{{
		Opcode: OpAcceptConnection,
		Params: append(append([]byte(nil), a[:]...), 0x01),
	}}, nil
}

func rejectWith(a Addr, reason byte) Command {
	return Command{
		Opcode: OpRejectConnection,
		Params: append(append([]byte(nil), a[:]...), reason),
	}
}

func (m *Manager) connected(e Event) error {
	// Status, handle, address, link type, encryption mode.
	if len(e.Params) < 11 {
		return fmt.Errorf("pair: a connection complete of %d bytes", len(e.Params))
	}

	a, _ := ParseAddr(e.Params[3:])

	if status := e.Params[0]; status != 0 {
		delete(m.peers, a)
		return fmt.Errorf("pair: %v did not connect, status %#02x", a, status)
	}

	p := m.peer(a)
	p.Handle = binary.LittleEndian.Uint16(e.Params[1:]) & 0x0fff
	p.Connected = true
	p.Encrypted = false
	return nil
}

func (m *Manager) disconnected(e Event) error {
	handle, ok := Handle(e.Params)
	if !ok {
		return fmt.Errorf("pair: a disconnection complete of %d bytes", len(e.Params))
	}

	p := m.Connected(handle)
	if p == nil {
		return nil
	}

	p.Connected = false
	p.Encrypted = false
	if m.Lost != nil {
		m.Lost(p)
	}
	return nil
}

func (m *Manager) linkKey(e Event) ([]Command, error) {
	a, ok := e.Addr()
	if !ok {
		return nil, fmt.Errorf("pair: a link key request of %d bytes", len(e.Params))
	}

	key, ok := m.Keys.Key(a)
	if !ok {
		// A negative reply starts pairing.
		m.peer(a).Bonded = false
		return []Command{reply(OpLinkKeyNegative, a)}, nil
	}

	m.peer(a).Bonded = true
	return []Command{{
		Opcode: OpLinkKeyReply,
		Params: append(append([]byte(nil), a[:]...), key[:]...),
	}}, nil
}

func (m *Manager) remember(e Event) error {
	// Address, key, key type.
	if len(e.Params) < 23 {
		return fmt.Errorf("pair: a link key notification of %d bytes", len(e.Params))
	}

	a, _ := ParseAddr(e.Params)

	// The debug link key is published in the spec.
	if e.Params[22] == KeyDebug {
		return fmt.Errorf("pair: %v produced a debug key, which is not saved", a)
	}

	var key Key
	copy(key[:], e.Params[6:22])

	if err := m.Keys.Save(a, key); err != nil {
		return fmt.Errorf("pair: saving the key for %v: %w", a, err)
	}

	m.peer(a).Bonded = true
	return nil
}

func (m *Manager) capability(e Event) ([]Command, error) {
	a, ok := e.Addr()
	if !ok {
		return nil, fmt.Errorf("pair: an io capability request of %d bytes", len(e.Params))
	}

	// Capability, out of band data present, authentication requirements.
	return []Command{{
		Opcode: OpIOCapabilityReply,
		Params: append(append([]byte(nil), a[:]...), m.IO, 0x00, m.Auth),
	}}, nil
}

func (m *Manager) confirm(e Event) ([]Command, error) {
	a, ok := e.Addr()
	if !ok {
		return nil, fmt.Errorf("pair: a user confirmation request of %d bytes", len(e.Params))
	}

	var number uint32
	if len(e.Params) >= 10 {
		number = binary.LittleEndian.Uint32(e.Params[6:])
	}

	if m.Confirm != nil && !m.Confirm(a, number) {
		return []Command{reply(OpConfirmNegative, a)}, nil
	}
	return []Command{reply(OpConfirmReply, a)}, nil
}

func (m *Manager) legacy(e Event) ([]Command, error) {
	a, ok := e.Addr()
	if !ok {
		return nil, fmt.Errorf("pair: a pin code request of %d bytes", len(e.Params))
	}
	return []Command{reply(OpPINCodeNegative, a)}, nil
}

func (m *Manager) passkey(e Event) ([]Command, error) {
	a, ok := e.Addr()
	if !ok {
		return nil, fmt.Errorf("pair: a passkey request of %d bytes", len(e.Params))
	}
	return []Command{reply(OpPasskeyNegative, a)}, nil
}

func (m *Manager) paired(e Event) error {
	// Status, then the address.
	if len(e.Params) < 7 {
		return fmt.Errorf("pair: a simple pairing complete of %d bytes", len(e.Params))
	}

	a, _ := ParseAddr(e.Params[1:])

	if status := e.Params[0]; status != 0 {
		p := m.peer(a)
		p.Bonded = false
		if err := m.Keys.Forget(a); err != nil {
			return fmt.Errorf("pair: dropping the key for %v: %w", a, err)
		}
		return fmt.Errorf("pair: %v failed to pair, status %#02x", a, status)
	}

	p := m.peer(a)
	p.Bonded = true
	if m.Bonded != nil {
		m.Bonded(p)
	}
	return nil
}

func (m *Manager) encrypted(e Event) error {
	// Status, handle, enabled.
	if len(e.Params) < 4 {
		return fmt.Errorf("pair: an encryption change of %d bytes", len(e.Params))
	}

	handle, _ := Handle(e.Params)
	p := m.Connected(handle)
	if p == nil {
		return nil
	}

	if status := e.Params[0]; status != 0 {
		p.Encrypted = false
		return fmt.Errorf("pair: encryption for %v failed, status %#02x", p.Addr, status)
	}

	p.Encrypted = e.Params[3] != 0
	return nil
}

func (m *Manager) Forget(a Addr) error {
	delete(m.peers, a)
	return m.Keys.Forget(a)
}
