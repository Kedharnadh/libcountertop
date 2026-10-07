package capture

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestCaptureClosesWithoutMoreAudio(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := New(t.TempDir())
		if _, err := d.Start(time.Second); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		synctest.Wait()
		d.mu.Lock()
		defer d.mu.Unlock()
		if d.frames != nil || d.pre != nil || d.post != nil {
			t.Fatal("capture files remained open after expiry")
		}
	})
}
