package hci

import (
	"context"
	"encoding/binary"
	"sync"
	"testing"
	"time"
)

type script struct {
	mu   sync.Mutex
	at   int
	from []packet

	answer  func(opcode uint16) (packet, bool)
	pending []packet
}

func (s *script) next(within time.Duration) (packet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pending) > 0 {
		g := s.pending[0]
		s.pending = s.pending[1:]
		return g, nil
	}
	if len(s.from) == 0 {
		return packet{}, errQuiet
	}

	g := s.from[s.at%len(s.from)]
	s.at++
	return g, nil
}

func (s *script) Send(b []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.answer != nil && len(b) >= 3 && b[0] == typeCommand {
		if g, ok := s.answer(binary.LittleEndian.Uint16(b[1:])); ok {
			s.pending = append(s.pending, g)
		}
	}
	return len(b), nil
}

func (s *script) Wake() error  { return nil }
func (s *script) Asleep() bool { return false }

func eventPacket(code byte, params ...byte) packet {
	return packet{kind: typeEvent, event: event{Code: code, Params: params}}
}

func completePacket(opcode uint16) packet {
	params := []byte{0x01}
	params = binary.LittleEndian.AppendUint16(params, opcode)
	return eventPacket(eventCommandComplete, append(params, 0x00)...)
}

func TestTheReaderIsSharedSafely(t *testing.T) {
	s := newSession(&script{
		from: []packet{
			eventPacket(eventLEMeta, leAdvertisingReport, 0x00),
			eventPacket(0x05, 0x00, 0x05, 0x00, 0x13),
			{kind: typeACL, acl: aclPacket{Handle: 5, Data: []byte{1, 2, 3}}},
		},
		answer: func(op uint16) (packet, bool) { return completePacket(op), true },
	})

	ctx, cancel := context.WithCancel(context.Background())
	go s.serve(ctx)

	var wg sync.WaitGroup

	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				got, done := s.Reports()
				select {
				case <-got:
				case <-time.After(20 * time.Millisecond):
				}
				done()
			}
		}()
	}

	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op := uint16(0x1000 + i)
			for range 20 {
				_, _ = s.Command(op)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		events, release := s.Events()
		defer release()

		for range 50 {
			select {
			case <-events:
			case <-time.After(20 * time.Millisecond):
			}
		}
	}()

	wg.Wait()
	cancel()

	select {
	case <-s.done:
	case <-time.After(2 * time.Second):
		t.Fatal("the reader did not stop")
	}
	if s.Err() == nil {
		t.Error("a stopped reader has no reason recorded")
	}
}

func TestASlowTakerIsDroppedNotWaitedOn(t *testing.T) {
	s := newSession(&script{from: []packet{eventPacket(eventLEMeta, leAdvertisingReport, 0x00)}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.serve(ctx)

	_, stuck := s.Reports()
	defer stuck()

	events, release := s.Events()
	defer release()

	select {
	case <-events:
	case <-time.After(300 * time.Millisecond):
	}

	if err := s.Err(); err != nil {
		t.Fatalf("the reader stopped: %v", err)
	}

	s.p.(*script).mu.Lock()
	s.p.(*script).answer = func(op uint16) (packet, bool) { return completePacket(op), true }
	s.p.(*script).mu.Unlock()

	if _, err := s.Command(hciReadAddress); err != nil {
		t.Fatalf("the line is wedged behind a taker that stopped reading: %v", err)
	}
}
