package logd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const reader = "/dev/socket/logdr"

// logger_entry fields every version shares: len, hdr_size, pid, tid, sec, nsec.
const entryPrefix = 20

// LOGGER_ENTRY_MAX_LEN.
const maxEntry = 5 * 1024

const dumpTimeout = 10 * time.Second

func Dump(tag string) ([]Line, error) {
	c, err := net.Dial("unixpacket", reader)
	if err != nil {
		return nil, fmt.Errorf("logd: dial %s: %w", reader, err)
	}
	defer c.Close()

	if err := c.SetDeadline(time.Now().Add(dumpTimeout)); err != nil {
		return nil, err
	}

	cmd := fmt.Sprintf("dumpAndClose lids=%d", idMain)
	if _, err := c.Write(append([]byte(cmd), 0)); err != nil {
		return nil, fmt.Errorf("logd: %s: %w", cmd, err)
	}

	var out []Line
	buf := make([]byte, maxEntry+entryPrefix+64)
	for {
		n, err := c.Read(buf)
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return out, fmt.Errorf("logd: reading %s: %w", reader, err)
		}

		l, t, ok := entry(buf[:n])
		if !ok || t != tag {
			continue
		}
		out = append(out, l)
	}
}

func entry(b []byte) (Line, string, bool) {
	if len(b) < entryPrefix {
		return Line{}, "", false
	}

	size := int(binary.LittleEndian.Uint16(b[0:2]))
	// The oldest layout has no hdr_size: those two bytes are padding.
	head := int(binary.LittleEndian.Uint16(b[2:4]))
	if head == 0 {
		head = entryPrefix
	}
	if head < entryPrefix || head > len(b) {
		return Line{}, "", false
	}

	msg := b[head:]
	if size < len(msg) {
		msg = msg[:size]
	}
	if len(msg) < 2 {
		return Line{}, "", false
	}

	rest := msg[1:]
	i := bytes.IndexByte(rest, 0)
	if i < 0 {
		return Line{}, "", false
	}

	return Line{
		Level: Level(msg[0]),
		Text:  strings.TrimRight(string(rest[i+1:]), "\x00\n"),
	}, string(rest[:i]), true
}
