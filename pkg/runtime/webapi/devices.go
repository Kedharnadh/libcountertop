package webapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ygelfand/libcountertop/pkg/network/peer/pictures"
)

func (a *API) picture(w http.ResponseWriter, r *http.Request) {
	_, svg := pictures.For(chi.URLParam(r, "board"))
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if _, err := w.Write([]byte(svg)); err != nil {
		slog.Debug("writing a device picture failed", "err", err)
	}
}
