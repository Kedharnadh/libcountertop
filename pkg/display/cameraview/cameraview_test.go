package cameraview

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLayoutFitsTheScaleAndTheCount(t *testing.T) {
	cases := []struct {
		name       string
		n, scale   int
		w, h       float64
		cols, rows int
	}{
		{"landscape, mid scale, many", 20, 3, 1920, 1200, 3, 3},
		{"landscape, largest scale", 20, 5, 1920, 1200, 1, 1},
		{"landscape, smallest scale", 40, 1, 1920, 1200, 5, 5},
		{"few cameras shrink the grid", 2, 1, 1920, 1200, 2, 1},
		{"portrait, mid scale", 20, 3, 1200, 1920, 3, 7},
		{"one camera", 1, 3, 1280, 800, 1, 1},
	}
	for _, c := range cases {
		cols, rows := Layout(c.n, c.scale, c.w, c.h)
		if cols != c.cols || rows != c.rows {
			t.Errorf("%s: %dx%d, want %dx%d", c.name, cols, rows, c.cols, c.rows)
		}
	}
}

func jpegOf(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestASnapshotIsFittedToTheTile(t *testing.T) {
	shot := jpegOf(t, 640, 360)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/camera_proxy/camera.door" || r.URL.Query().Get("token") != "abc" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(shot)
	}))
	defer srv.Close()

	img, err := snapshot(
		context.Background(),
		srv.Client(),
		srv.URL,
		"/api/camera_proxy/camera.door?token=abc",
		320,
		320,
	)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 320 || b.Dy() != 180 {
		t.Errorf("fitted to %v, want 320x180", b.Size())
	}
}

func TestNoAddressIsAnError(t *testing.T) {
	if _, err := snapshot(
		context.Background(),
		http.DefaultClient,
		"",
		"/x",
		10,
		10,
	); err != ErrNoSite {
		t.Errorf("an unknown address said %v, want ErrNoSite", err)
	}
}

func TestTheLiveStreamYieldsEachFrame(t *testing.T) {
	shot := jpegOf(t, 160, 90)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/camera_proxy_stream/camera.door" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "multipart/x-mixed-replace;boundary=frame")
		for range 3 {
			fmt.Fprintf(
				w,
				"--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n",
				len(shot),
			)
			_, _ = w.Write(shot)
			_, _ = w.Write([]byte("\r\n"))
		}
		_, _ = w.Write([]byte("--frame--\r\n"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	frames := 0
	_ = stream(ctx, srv.Client(), srv.URL, "/api/camera_proxy/camera.door?token=abc", 80, 80,
		func(img *image.NRGBA) {
			frames++
			if b := img.Bounds(); b.Dx() != 80 || b.Dy() != 45 {
				t.Errorf("frame fitted to %v, want 80x45", b.Size())
			}
		})
	if frames != 3 {
		t.Errorf("got %d frames, want 3", frames)
	}
}
