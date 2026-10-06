package ui

import (
	"log/slog"
	"sync"
)

const keptPictures = 32

var pictures struct {
	mu    sync.Mutex
	by    map[uint64]*Image
	order []uint64
}

// Picture is an encoded image as something to draw, decoded once per id and nil where there is none.
func Picture(raw []byte, id uint64) *Image {
	if len(raw) == 0 {
		return nil
	}

	pictures.mu.Lock()
	defer pictures.mu.Unlock()

	if img, ok := pictures.by[id]; ok {
		return img
	}
	img, err := Decode(raw)
	if err != nil {
		slog.Warn("a picture would not decode", "bytes", len(raw), "err", err)
	}
	if pictures.by == nil {
		pictures.by = map[uint64]*Image{}
	}
	pictures.by[id] = img
	pictures.order = append(pictures.order, id)
	if len(pictures.order) > keptPictures {
		delete(pictures.by, pictures.order[0])
		pictures.order = pictures.order[1:]
	}
	return img
}
