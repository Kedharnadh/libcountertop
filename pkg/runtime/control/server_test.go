package control

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func socket(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ctl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "s")
}

func TestCloseRevokesExistingClientsAndCanRestart(t *testing.T) {
	var calls atomic.Int32
	s := NewServer(socket(t), func(context.Context, []string) (string, error) {
		calls.Add(1)
		return "answered", nil
	})
	t.Cleanup(s.Close)
	for range 2 {
		if err := s.Start(); err != nil {
			t.Fatal(err)
		}
		conn, err := net.Dial("unix", s.address)
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		in := bufio.NewScanner(conn)
		if _, err := fmt.Fprintln(conn, "ping"); err != nil {
			t.Fatal(err)
		}
		if err := Answer(in, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
		s.Close()
		s.Wait()
		_, _ = fmt.Fprintln(conn, "ping")
		if in.Scan() {
			t.Errorf("revoked connection remained usable: %q", in.Text())
		}
		_ = conn.Close()
	}
	if calls.Load() != 2 {
		t.Errorf("revoked clients dispatched extra requests: %d calls", calls.Load())
	}
}

func TestCloseCancelsCommandContext(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	s := NewServer(socket(t), func(ctx context.Context, _ []string) (string, error) {
		close(started)
		<-ctx.Done()
		close(stopped)
		return "", ctx.Err()
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	conn, err := net.Dial("unix", s.address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = fmt.Fprintln(conn, "wait")
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("command did not start")
	}
	s.Close()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("closing the harness did not cancel the command")
	}
	s.Wait()
}

func TestSendStopsAfterFailureAndKeepsPartialOutput(t *testing.T) {
	var calls atomic.Int32
	s := NewServer(socket(t), func(context.Context, []string) (string, error) {
		calls.Add(1)
		return "step completed", errors.New("later step failed")
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { s.Close(); s.Wait() }()
	var out bytes.Buffer
	err := Send(t.Context(), s.address, []string{"first", "second"}, &out)
	if err == nil || !strings.Contains(err.Error(), "later step failed") {
		t.Fatalf("Send error = %v", err)
	}
	if out.String() != "step completed\n" || calls.Load() != 1 {
		t.Errorf("partial output or sequencing lost: output=%q calls=%d", out.String(), calls.Load())
	}
}
