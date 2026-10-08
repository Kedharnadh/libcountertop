package cast

import "testing"

func FuzzUnmarshal(f *testing.F) {
	f.Add(Message{Source: "sender-0", Destination: "receiver-0", Namespace: NSHeartbeat,
		Payload: Ping()}.Marshal())
	f.Add(Message{Namespace: NSDeviceAuth, Binary: []byte{1, 2, 3}}.Marshal())
	f.Add([]byte{})
	f.Add([]byte{0x08})

	f.Fuzz(func(t *testing.T, raw []byte) {
		m, err := Unmarshal(raw)
		if err != nil {
			return
		}

		if _, err := Unmarshal(m.Marshal()); err != nil {
			t.Fatalf("a message this parsed would not parse again: %v", err)
		}
	})
}

func FuzzNext(f *testing.F) {
	f.Add(Frame(Message{Namespace: NSReceiver, Payload: `{"type":"GET_STATUS"}`}))
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{0x00, 0x00, 0x00, 0x05, 0x01})

	f.Fuzz(func(t *testing.T, raw []byte) {
		m, n, err := Next(raw)
		if err != nil {
			return
		}
		if n <= 0 || n > len(raw) {
			t.Fatalf("a message of %d bytes was taken out of %d", n, len(raw))
		}

		_ = m
	})
}

func FuzzKind(f *testing.F) {
	f.Add(`{"type":"LAUNCH","requestId":1}`)
	f.Add(`{}`)
	f.Add(``)

	f.Fuzz(func(t *testing.T, payload string) {
		_, _ = Kind(payload)
	})
}

func FuzzPayloads(f *testing.F) {
	f.Add(`{"type":"LAUNCH","requestId":1,"appId":"CC1AD845"}`)
	f.Add(`{"type":"SET_VOLUME","volume":{"level":0.5}}`)
	f.Add(`{"type":"LOAD","media":{"contentId":"http://x/a.mp3"}}`)
	f.Add(`{"type":"SEEK","currentTime":1e308}`)
	f.Add(`{"media":{"duration":-1}}`)

	f.Fuzz(func(t *testing.T, payload string) {
		_, _ = ParseLaunch(payload)
		_, _ = ParseStop(payload)
		_, _ = ParseSetVolume(payload)
		_, _ = ParseLoad(payload)
		_, _ = ParseMediaRequest(payload)
	})
}

func FuzzReceive(f *testing.F) {
	f.Add(NSReceiver, `{"type":"GET_STATUS","requestId":1}`)
	f.Add(NSMedia, `{"type":"PLAY"}`)
	f.Add(NSConnection, `{"type":"CONNECT"}`)
	f.Add("", "")

	f.Fuzz(func(t *testing.T, namespace, payload string) {
		r := NewReceiver("Kitchen")

		_, _ = r.Receive(Message{Source: SenderID, Namespace: NSConnection, Payload: Connect()})
		_, _ = r.Receive(Message{Source: SenderID, Namespace: NSReceiver,
			Payload: `{"type":"LAUNCH","requestId":1,"appId":"` + DefaultMediaReceiver + `"}`})

		_, _ = r.Receive(Message{Source: SenderID, Namespace: namespace, Payload: payload})
	})
}
