package call

import (
	sharedstatus "github.com/ygelfand/libcountertop/pkg/media/callstatus"
	"github.com/ygelfand/libcountertop/pkg/media/rtc"
)

func (c *Calls) statusController() sharedstatus.Controller[*session] {
	return sharedstatus.Controller[*session]{Options: sharedstatus.Options[*session]{
		Snapshot: func(s *session) (sharedstatus.Link, sharedstatus.Size) {
			c.mu.Lock()
			defer c.mu.Unlock()
			if s.link == nil {
				return nil, sharedstatus.Size{}
			}
			return s.link, sharedstatus.Size(s.own)
		},
		Own: func(s *session, sz sharedstatus.Size) (func(), bool) {
			c.mu.Lock()
			if c.now != s || s.own == Size(sz) {
				c.mu.Unlock()
				return nil, false
			}
			s.own = Size(sz)
			c.mu.Unlock()
			return func() { c.show(s) }, true
		},
		Remote: func(s *session, sz sharedstatus.Size) (func(), bool) {
			c.mu.Lock()
			if c.now != s {
				c.mu.Unlock()
				return nil, false
			}
			s.remote = Size(sz)
			c.mu.Unlock()
			return func() { c.show(s) }, true
		},
		Status: func(s *session, st sharedstatus.Status) (func(), bool) {
			c.mu.Lock()
			if c.now != s {
				c.mu.Unlock()
				return nil, false
			}
			s.RemoteMuted, s.RemoteCameraOff = st.Muted, st.CameraOff
			c.mu.Unlock()
			return func() { c.show(s) }, true
		},
		Changed: c.changed,
	}}
}
func (c *Calls) tell(s *session)     { c.statusController().Tell(s) }
func (c *Calls) tellSize(s *session) { c.statusController().TellSize(s) }
func (c *Calls) ownSized(s *session, sz Size) {
	c.statusController().OwnSized(s, sharedstatus.Size(sz))
}
func (c *Calls) received(s *session) func(rtc.Message) { return c.statusController().Received(s) }
