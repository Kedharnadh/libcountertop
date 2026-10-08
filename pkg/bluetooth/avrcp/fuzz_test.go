package avrcp

import "testing"

func FuzzReceive(f *testing.F) {
	f.Add([]byte{0x00, 0x11, 0x0e, 0x00, 0x48, 0x00, 0x00, 0x19, 0x58})
	f.Add([]byte{0x00, 0x11, 0x0e, 0x01, 0x48, 0x00, 0x00, 0x19, 0x58,
		PDUGetElementAttributes, 0x00, 0x00, 0x00})
	f.Add([]byte{0x00})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, raw []byte) {
		tr, err := ParseTransport(raw)
		if err != nil {
			return
		}
		if len(tr.Payload) > len(raw) {
			t.Fatalf("a payload of %d bytes out of %d", len(tr.Payload), len(raw))
		}

		avc, err := ParseAVC(tr.Payload)
		if err != nil {
			return
		}
		if len(avc.Operands) > len(raw) {
			t.Fatalf("operands of %d bytes out of %d", len(avc.Operands), len(raw))
		}

		pdu, err := ParsePDU(avc.Operands)
		if err != nil {
			return
		}
		if len(pdu.Params) > len(raw) {
			t.Fatalf("parameters of %d bytes out of %d", len(pdu.Params), len(raw))
		}

		params(pdu.Params)
	})
}

func FuzzParams(f *testing.F) {
	f.Add([]byte{0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x6a, 0x00, 0x04, 'S', 'o', 'n', 'g'})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{0x00})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, raw []byte) { params(raw) })
}

func params(p []byte) {
	_, _ = ParseElementAttributes(p)
	_, _ = ParsePlayStatus(p)
	_, _ = ParseNotification(p, true)
	_, _ = ParseNotification(p, false)
	_, _ = ParseCapabilities(p)
}
