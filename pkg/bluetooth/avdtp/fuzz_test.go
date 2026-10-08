package avdtp

import "testing"

func FuzzParseMessage(f *testing.F) {
	f.Add([]byte{0x00, SignalDiscover})
	f.Add([]byte{0x02, SignalGetCapabilities, 0x01, 0x00})
	f.Add([]byte{0x00})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, raw []byte) {
		m, err := ParseMessage(raw)
		if err != nil {
			return
		}
		if len(m.Data) > len(raw) {
			t.Fatalf("a message carrying %d bytes out of %d", len(m.Data), len(raw))
		}

		_, _ = ParseSEPs(m.Data)

		caps, err := ParseCapabilities(m.Data)
		if err != nil {
			return
		}
		for _, c := range caps {
			if len(c.Data) > len(raw) {
				t.Fatalf("a capability carrying %d bytes out of %d", len(c.Data), len(raw))
			}
			_, _ = ParseSBC(c)
		}
	})
}

func FuzzParseSBC(f *testing.F) {
	f.Add([]byte{0x21, 0x15, 0x02, 0x35})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseSBC(Capability{Category: CatMediaCodec, Data: data})
	})
}

func FuzzParseMedia(f *testing.F) {
	f.Add([]byte{0x80, 0x60, 0x00, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0x01, 0x9c})
	f.Add([]byte{0x80, 0x60, 0x00, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0x80})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, raw []byte) {
		m, err := ParseMedia(raw)
		if err != nil {
			return
		}
		if len(m.Payload) > len(raw) {
			t.Fatalf("a payload of %d bytes out of a packet of %d", len(m.Payload), len(raw))
		}

		r := &Reassembler{}
		for range 3 {
			r.Push(m)
		}
	})
}
