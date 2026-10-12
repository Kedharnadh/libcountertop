package whistle

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/vad"
)

const (
	version    = 2
	maxPayload = 1 << 20

	msgClip     = 'C'
	msgKeywords = 'K'
	msgReady    = 'R'
	msgText     = 'T'
	msgFailed   = 'F'
)

var errVersion = errors.New("whistle: helper protocol version mismatch")

type message struct {
	kind    uint32
	payload []byte
}

func writeMessage(w io.Writer, kind uint32, payload []byte) error {
	buf := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(buf[0:], kind)
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(payload)))
	copy(buf[8:], payload)
	_, err := w.Write(buf)
	return err
}

func readMessage(r io.Reader) (message, error) {
	var head [8]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return message{}, err
	}
	m := message{kind: binary.LittleEndian.Uint32(head[0:])}
	n := binary.LittleEndian.Uint32(head[4:])
	if n > maxPayload {
		return message{}, fmt.Errorf("whistle: %d byte message", n)
	}
	m.payload = make([]byte, n)
	if _, err := io.ReadFull(r, m.payload); err != nil {
		return message{}, err
	}
	return m, nil
}

func clip(c vad.Clip) []byte {
	b := make([]byte, 8+2*len(c.Samples))
	binary.LittleEndian.PutUint32(b[0:], uint32(c.Start.Milliseconds()))
	binary.LittleEndian.PutUint32(b[4:], uint32(c.End.Milliseconds()))
	for i, s := range c.Samples {
		binary.LittleEndian.PutUint16(b[8+2*i:], uint16(s))
	}
	return b
}

func ready(m message) (depth int, err error) {
	if m.kind == msgFailed {
		return 0, fmt.Errorf("whistle: helper failed: %s", m.payload)
	}
	if m.kind != msgReady || len(m.payload) < 8 {
		return 0, fmt.Errorf("whistle: helper sent %q before ready", rune(m.kind))
	}
	if binary.LittleEndian.Uint32(m.payload[0:]) != version {
		return 0, errVersion
	}
	return int(binary.LittleEndian.Uint32(m.payload[4:])), nil
}

func transcript(m message) (Transcript, error) {
	if len(m.payload) < 16 {
		return Transcript{}, fmt.Errorf("whistle: %d byte transcript", len(m.payload))
	}
	ms := func(off int) time.Duration {
		return time.Duration(binary.LittleEndian.Uint32(m.payload[off:])) * time.Millisecond
	}
	var body struct {
		Text  string `json:"text"`
		Words []Word `json:"words"`
	}
	if err := json.Unmarshal(m.payload[16:], &body); err != nil {
		return Transcript{}, fmt.Errorf("whistle: transcript: %w", err)
	}
	return Transcript{
		Text:      body.Text,
		Words:     body.Words,
		Start:     ms(0),
		End:       ms(4),
		Took:      ms(8),
		DecodeTPS: float64(binary.LittleEndian.Uint32(m.payload[12:])) / 100,
	}, nil
}
