package adb

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tmpDir = "/data/local/tmp"

func liveDevice(t *testing.T) *Device {
	t.Helper()
	if os.Getenv("ADB_LIVE") == "" {
		t.Skip("set ADB_LIVE=1 to run against a connected device")
	}
	d, err := Attach(os.Getenv("ANDROID_SERIAL"))
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	return d
}

func TestLiveShellExitStatus(t *testing.T) {
	d := liveDevice(t)
	t.Logf("serial=%s", d.Serial())

	if _, err := d.Shell("true"); err != nil {
		t.Errorf("true: unexpected error: %v", err)
	}

	_, err := d.Shell("false")
	var derr *Error
	if !errors.As(err, &derr) {
		t.Fatalf("false: want *Error, got %v", err)
	}
	if derr.Code == 0 {
		t.Errorf("false: exit code = 0, want non-zero")
	}

	out, code, err := d.ShellCode("ls /definitely-not-here")
	if err != nil {
		t.Fatalf("ShellCode: %v", err)
	}
	if code == 0 {
		t.Errorf("missing path: exit code = 0, want non-zero")
	}
	if strings.Contains(out, "\r") {
		t.Errorf("output still contains CR: %q", out)
	}
}

func TestLiveUptime(t *testing.T) {
	d := liveDevice(t)

	up, err := d.Uptime()
	if err != nil || up <= 0 {
		t.Errorf("Uptime() = %v, %v; want > 0", up, err)
	}
}

func TestLivePushPull(t *testing.T) {
	d := liveDevice(t)

	local := filepath.Join(t.TempDir(), "probe")
	want := []byte("adb push/pull probe\n")
	if err := os.WriteFile(local, want, 0o644); err != nil {
		t.Fatal(err)
	}

	remote := tmpDir + "/adb-probe"
	if err := d.PushFile(local, remote, 0o755); err != nil {
		t.Fatalf("PushFile: %v", err)
	}
	t.Cleanup(func() { _, _ = d.Shell("rm -f " + quote(remote)) })

	got, err := d.ReadFile(remote)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("round trip = %q, want %q", got, want)
	}
}
