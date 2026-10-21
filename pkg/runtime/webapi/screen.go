package webapi

import (
	"bytes"
	"image/jpeg"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const (
	screenFresh   = time.Second
	screenQuality = 80
)

type screenCache struct {
	mu    sync.Mutex
	taken time.Time
	jpeg  []byte
}

func (a *API) screen(w http.ResponseWriter, r *http.Request) {
	if a.o.Screen == nil {
		http.NotFound(w, r)
		return
	}
	a.shot.mu.Lock()
	defer a.shot.mu.Unlock()
	if now := a.o.Now(); now.Sub(a.shot.taken) >= screenFresh {
		img, err := a.o.Screen()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		var b bytes.Buffer
		if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: screenQuality}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		a.shot.taken, a.shot.jpeg = now, b.Bytes()
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write(a.shot.jpeg); err != nil {
		slog.Debug("writing the screen failed", "err", err)
	}
}
