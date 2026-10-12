package radar

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func tileServer(t *testing.T, missing string) *httptest.Server {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, tileSize, tileSize))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}
	var body bytes.Buffer
	if err := png.Encode(&body, img); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, missing) {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write(body.Bytes())
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestALayerSurvivesAMissingTile(t *testing.T) {
	srv := tileServer(t, "/0.png")
	tl := &tiles{client: srv.Client(), limit: 2, cache: newCache("")}
	v := View{Lat: 41.3, Lon: -73.06, Zoom: 6.5, Size: image.Pt(640, 400)}
	url := func(z, x, y int) string { return fmt.Sprintf("%s/%d/%d/%d.png", srv.URL, z, x, y%2) }

	img, err := tl.layer(context.Background(), v, 6, lifetime{}, url, nil)
	if err != nil {
		t.Fatalf("one missing tile failed the layer: %v", err)
	}
	clear := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] == 0 {
			clear++
		}
	}
	if clear == 0 || clear == len(img.Pix)/4 {
		t.Errorf(
			"%d of %d pixels clear, want the missing tiles blank and the rest drawn",
			clear,
			len(img.Pix)/4,
		)
	}
	if got := img.RGBAAt(0, 0); got != (color.RGBA{}) && got.A != 0xff {
		t.Errorf("corner pixel %v is neither drawn nor blank", got)
	}
}

func TestALayerFailsWhenEveryTileDoes(t *testing.T) {
	srv := tileServer(t, ".png")
	tl := &tiles{client: srv.Client(), limit: 2, cache: newCache("")}
	v := View{Lat: 41.3, Lon: -73.06, Zoom: 6.5, Size: image.Pt(640, 400)}
	url := func(z, x, y int) string { return fmt.Sprintf("%s/%d/%d/%d.png", srv.URL, z, x, y) }

	if _, err := tl.layer(context.Background(), v, 6, lifetime{}, url, nil); err == nil {
		t.Error("a layer with no tiles reported no error")
	}
}
