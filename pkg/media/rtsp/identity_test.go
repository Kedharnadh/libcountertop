package rtsp

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestProductIdentityIsPreservedInHeadersSDPAndSenderReports(t *testing.T) {
	for _, id := range []Identity{{"Panel", "panel-"}, {"Speaker", "speaker-"}} {
		t.Run(id.Name, func(t *testing.T) {
			srv := NewServerWithIdentity(id)
			st := NewStream()
			st.Write([]byte{0, 0, 0, 1, 0x67, 0x42, 0, 0x1f, 0, 0, 0, 1, 0x68, 1}, 0)
			sdp, ok := st.sdpNamed(1, "127.0.0.1", time.Second, srv.identity.Name)
			if !ok || !strings.Contains(sdp, "s="+id.Name+"\r\n") {
				t.Fatal(sdp)
			}
			left, right := net.Pipe()
			defer left.Close()
			defer right.Close()
			done := make(chan error, 1)
			go func() { done <- (&conn{srv: srv, c: left}).reply(&request{header: map[string]string{}}, 200, nil, "") }()
			reader := bufio.NewReader(right)
			var header strings.Builder
			for {
				line, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					t.Fatal(err)
				}
				header.WriteString(line)
				if line == "\r\n" {
					break
				}
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(header.String(), "Server: "+id.Name+"\r\n") {
				t.Fatal(header.String())
			}
			// SETUP is where the server's identity is propagated to its RTP session.
			c := &conn{srv: srv, c: left}
			srv.Handle("main", st)
			status, _, _ := c.handle(&request{method: "SETUP", url: "rtsp://127.0.0.1/main/trackID=0", header: map[string]string{"transport": "RTP/AVP/TCP;unicast;interleaved=0-1"}})
			if status != 200 || c.ses.cnamePrefix != id.CNAMEPrefix {
				t.Fatalf("status=%d session=%+v", status, c.ses)
			}
			c.ses.close()
		})
	}
}
