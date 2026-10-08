package capture

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Dump struct {
	mu                sync.Mutex
	frames, pre, post *os.File
	until             time.Time
	timer             *time.Timer
	directory         string
}

func New(directory string) *Dump { return &Dump{directory: directory} }

func (d *Dump) Start(within time.Duration) (string, error) {
	if within <= 0 {
		return "", fmt.Errorf("bluetooth capture duration must be positive")
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	d.shut()
	dumpFrames, dumpPre, dumpPost := d.paths()

	made := make([]*os.File, 0, 3)
	for _, path := range []string{dumpFrames, dumpPre, dumpPost} {
		f, err := os.Create(path)
		if err != nil {
			for _, o := range made {
				o.Close()
			}
			return "", err
		}
		made = append(made, f)
	}

	d.frames, d.pre, d.post = made[0], made[1], made[2]
	d.until = time.Now().Add(within)
	d.timer = time.AfterFunc(within, func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		if !time.Now().Before(d.until) {
			d.shut()
		}
	})

	return fmt.Sprintf("capturing %s to\n  %s  frames as they arrived\n"+
		"  %s  decoded, stereo\n  %s  resampled, stereo 48k\n",
		within, dumpFrames, dumpPre, dumpPost), nil
}

func (d *Dump) Frame(b []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.frames == nil {
		return
	}
	if time.Now().After(d.until) {
		d.shut()
		return
	}
	if _, err := d.frames.Write(b); err != nil {
		slog.Warn("bluetooth capture write failed", "err", err)
		d.shut()
	}
}

func (d *Dump) Write(pre, post []int16) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.pre == nil {
		return
	}
	if time.Now().After(d.until) {
		d.shut()
		return
	}

	if _, err := d.pre.Write(raw(pre)); err != nil {
		slog.Warn("bluetooth capture write failed", "err", err)
		d.shut()
		return
	}
	if _, err := d.post.Write(raw(post)); err != nil {
		slog.Warn("bluetooth capture write failed", "err", err)
		d.shut()
	}
}

func (d *Dump) shut() {
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	if d.pre == nil {
		return
	}

	d.frames.Close()
	d.pre.Close()
	d.post.Close()
	d.frames, d.pre, d.post = nil, nil, nil

	dumpFrames, dumpPre, dumpPost := d.paths()
	slog.Info("bluetooth capture finished",
		"frames", dumpFrames, "decoded", dumpPre, "resampled", dumpPost)
}

func raw(s []int16) []byte {
	b := make([]byte, len(s)*2)
	for i, v := range s {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(v))
	}
	return b
}

func (d *Dump) paths() (string, string, string) {
	return filepath.Join(
			d.directory,
			"bt-frames.sbc",
		), filepath.Join(
			d.directory,
			"bt-decoded.pcm",
		), filepath.Join(
			d.directory,
			"bt-resampled.pcm",
		)
}
