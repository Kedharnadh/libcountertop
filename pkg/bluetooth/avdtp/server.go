package avdtp

import "fmt"

// AVDTP: a command in the wrong state is refused with BAD_STATE.

type State int

const (
	Idle State = iota

	Configured

	Opened

	Streaming
)

func (s State) String() string {
	switch s {
	case Configured:
		return "configured"
	case Opened:
		return "open"
	case Streaming:
		return "streaming"
	}
	return "idle"
}

type Endpoint struct {
	SEP   SEP
	State State

	Config []Capability

	Remote byte
}

func (e *Endpoint) Codec() (SBC, bool) {
	c, ok := Find(e.Config, CatMediaCodec)
	if !ok {
		return SBC{}, false
	}
	s, err := ParseSBC(c)
	if err != nil {
		return SBC{}, false
	}
	return s, true
}

type Server struct {
	endpoints []*Endpoint

	Started func(*Endpoint)
	Stopped func(*Endpoint)
}

func NewServer(seid byte) *Server {
	return &Server{
		endpoints: []*Endpoint{{
			SEP: SEP{SEID: seid, Media: MediaAudio, TSEP: Sink},
		}},
	}
}

func (s *Server) Endpoint(seid byte) *Endpoint {
	for _, e := range s.endpoints {
		if e.SEP.SEID == seid {
			return e
		}
	}
	return nil
}

func (s *Server) Handle(m Message) Message {
	if m.Type != Command {
		return Message{}
	}

	switch m.Signal {
	case SignalDiscover:
		return m.Accept(s.discover()...)

	case SignalGetCapabilities, SignalGetAllCaps:
		return s.capabilities(m)

	case SignalSetConfig:
		return s.setConfig(m)

	case SignalGetConfig:
		return s.getConfig(m)

	case SignalOpen:
		return s.transition(m, Configured, Opened, nil)

	case SignalStart:
		return s.startStop(m, Opened, Streaming, s.Started)

	case SignalSuspend:
		return s.startStop(m, Streaming, Opened, s.Stopped)

	case SignalClose:
		return s.close(m)

	case SignalAbort:
		return s.abort(m)
	}

	// AVDTP: an unrecognised signal gets a general reject.
	return Message{Label: m.Label, Type: GeneralReject, Signal: m.Signal}
}

func (s *Server) discover() []byte {
	var out []byte
	for _, e := range s.endpoints {
		sep := e.SEP
		sep.InUse = e.State != Idle
		out = append(out, sep.Marshal()...)
	}
	return out
}

func (s *Server) capabilities(m Message) Message {
	seid, ok := SEID(m.Data)
	if !ok {
		return m.Reject(ErrorBadLength)
	}

	e := s.Endpoint(seid)
	if e == nil {
		return m.Reject(ErrorBadSEID)
	}
	return m.Accept(MarshalCapabilities(SinkCapabilities())...)
}

func (s *Server) setConfig(m Message) Message {
	// Our endpoint, then theirs, then the capabilities they chose.
	if len(m.Data) < 2 {
		return m.Reject(ErrorBadLength)
	}

	e := s.Endpoint(m.Data[0] >> 2)
	if e == nil {
		return m.Reject(ErrorBadSEID)
	}
	if e.State != Idle {
		return m.Reject(ErrorSEPInUse)
	}

	chosen, err := ParseCapabilities(m.Data[2:])
	if err != nil {
		return m.RejectConfig(0, ErrorBadPayloadFormat)
	}

	if _, ok := Find(chosen, CatMediaTransport); !ok {
		return m.RejectConfig(CatMediaTransport, ErrorUnsupportedConfig)
	}

	codec, ok := Find(chosen, CatMediaCodec)
	if !ok {
		return m.RejectConfig(CatMediaCodec, ErrorUnsupportedConfig)
	}

	sbc, err := ParseSBC(codec)
	if err != nil {
		return m.RejectConfig(CatMediaCodec, ErrorUnsupportedConfig)
	}
	if !sbc.Chosen() {
		return m.RejectConfig(CatMediaCodec, ErrorBadPayloadFormat)
	}
	if !sbc.Within(SinkSBC()) {
		return m.RejectConfig(CatMediaCodec, ErrorUnsupportedConfig)
	}

	for _, c := range chosen {
		switch c.Category {
		case CatMediaTransport, CatMediaCodec, CatDelayReporting:
		default:
			return m.RejectConfig(c.Category, ErrorBadCategory)
		}
	}

	e.Config = chosen
	e.Remote = m.Data[1] >> 2
	e.State = Configured

	return m.Accept()
}

func (s *Server) getConfig(m Message) Message {
	seid, ok := SEID(m.Data)
	if !ok {
		return m.Reject(ErrorBadLength)
	}

	e := s.Endpoint(seid)
	if e == nil {
		return m.Reject(ErrorBadSEID)
	}
	if e.State == Idle {
		return m.Reject(ErrorSEPNotInUse)
	}
	return m.Accept(MarshalCapabilities(e.Config)...)
}

func (s *Server) transition(m Message, from, to State, after func(*Endpoint)) Message {
	seid, ok := SEID(m.Data)
	if !ok {
		return m.Reject(ErrorBadLength)
	}

	e := s.Endpoint(seid)
	if e == nil {
		return m.Reject(ErrorBadSEID)
	}
	if e.State != from {
		return m.Reject(ErrorBadState)
	}

	e.State = to
	if after != nil {
		after(e)
	}
	return m.Accept()
}

// The refusal names the SEID that failed.
func (s *Server) startStop(m Message, from, to State, after func(*Endpoint)) Message {
	if len(m.Data) == 0 {
		return m.Reject(ErrorBadLength)
	}

	for _, b := range m.Data {
		seid := b >> 2

		e := s.Endpoint(seid)
		if e == nil {
			return Message{Label: m.Label, Type: ResponseReject, Signal: m.Signal,
				Data: []byte{b, ErrorBadSEID}}
		}
		if e.State != from {
			return Message{Label: m.Label, Type: ResponseReject, Signal: m.Signal,
				Data: []byte{b, ErrorBadState}}
		}
	}

	for _, b := range m.Data {
		e := s.Endpoint(b >> 2)
		e.State = to
		if after != nil {
			after(e)
		}
	}
	return m.Accept()
}

func (s *Server) close(m Message) Message {
	seid, ok := SEID(m.Data)
	if !ok {
		return m.Reject(ErrorBadLength)
	}

	e := s.Endpoint(seid)
	if e == nil {
		return m.Reject(ErrorBadSEID)
	}
	if e.State == Idle {
		return m.Reject(ErrorBadState)
	}

	streaming := e.State == Streaming
	s.release(e)

	if streaming && s.Stopped != nil {
		s.Stopped(e)
	}
	return m.Accept()
}

func (s *Server) abort(m Message) Message {
	seid, ok := SEID(m.Data)
	if !ok {
		return m.Reject(ErrorBadLength)
	}

	e := s.Endpoint(seid)
	if e == nil {
		return m.Reject(ErrorBadSEID)
	}

	streaming := e.State == Streaming
	s.release(e)

	if streaming && s.Stopped != nil {
		s.Stopped(e)
	}
	return m.Accept()
}

func (s *Server) release(e *Endpoint) {
	e.State = Idle
	e.Config = nil
	e.Remote = 0
}

func (s *Server) Forget() {
	for _, e := range s.endpoints {
		if e.State == Streaming && s.Stopped != nil {
			s.Stopped(e)
		}
		s.release(e)
	}
}

func (s *Server) String() string {
	if len(s.endpoints) == 0 {
		return "no endpoints"
	}
	e := s.endpoints[0]
	return fmt.Sprintf("endpoint %d %v", e.SEP.SEID, e.State)
}
