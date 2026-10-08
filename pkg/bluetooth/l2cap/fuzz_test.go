package l2cap

import "testing"

func FuzzParseFrame(f *testing.F) {
	f.Add(Frame{CID: CIDSignalling, Payload: []byte{1, 2, 3}}.Marshal())
	f.Add([]byte{0xff, 0xff, 0x40, 0x00})
	f.Add([]byte{0x00, 0x00})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, raw []byte) {
		fr, err := ParseFrame(raw)
		if err != nil {
			return
		}

		if len(fr.Payload) > len(raw) {
			t.Fatalf("a payload of %d bytes out of a frame of %d", len(fr.Payload), len(raw))
		}
		if _, err := ParseFrame(fr.Marshal()); err != nil {
			t.Fatalf("a frame this parsed would not parse again: %v", err)
		}
	})
}

func FuzzSignalling(f *testing.F) {
	f.Add([]byte{0x02, 0x01, 0x04, 0x00, 0x19, 0x00, 0x40, 0x00})
	f.Add([]byte{0x04, 0x01, 0x08, 0x00, 0x40, 0x00, 0x00, 0x00, 0x01, 0x02, 0xff, 0xff})
	f.Add([]byte{0x01, 0x01, 0xff, 0xff})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, payload []byte) {
		commands, err := ParseCommands(payload)
		if err != nil {
			return
		}

		for _, c := range commands {
			_, _ = ParseConnect(c)
			_, _ = ParseConnected(c)
			_, _ = ParseDisconnect(c)
			_, _ = ParseConfigure(c)
			_, _ = ParseConfigured(c)
		}
	})
}

func FuzzParseOptions(f *testing.F) {
	f.Add([]byte{0x01, 0x02, 0xa0, 0x02})
	f.Add([]byte{0x01, 0xff})
	f.Add([]byte{0x00, 0x00, 0x00})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, raw []byte) {
		opts, err := ParseOptions(raw)
		if err != nil {
			return
		}

		for _, o := range opts {
			if len(o.Value) > len(raw) {
				t.Fatalf("an option carrying %d bytes out of %d", len(o.Value), len(raw))
			}
		}
	})
}
