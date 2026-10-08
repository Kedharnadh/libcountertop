package widgets

import (
	"strings"
	"testing"

	gogui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/display/interaction"
	"github.com/ygelfand/libcountertop/pkg/display/style"
)

func TestExpandedVolumeCardFitsShortDisplays(t *testing.T) {
	for _, height := range []int{360, 480, 600} {
		presses := interaction.New()
		gestures := Gestures{}
		kit := Toolkit{Kit: func() style.Kit { return style.ByName("standard").Kit }, Pressable: presses.Pressable}
		w := gogui.SimpleWindow("volume", 960, height, &struct{}{}, func(*gogui.Window) {})
		view := kit.VolumeCard(w, VolumeOptions{
			Current: VolumeChoice{Key: "main", Label: "Device", Level: 40}, Open: true,
			Choices: []VolumeChoice{{Key: "main"}, {Key: "media", Label: "Media"}, {Key: "voice", Label: "Voice"}, {Key: "alerts", Label: "Alerts"}, {Key: "feedback", Label: "Feedback"}},
			Grip:    gestures.Grip, Expand: func() {}, Linger: func() {}, Dismiss: func() {}, Set: func(string, int) {},
		})
		layout := gogui.GenerateViewLayout(view, w)
		var card *gogui.Shape
		bars := 0
		var visit func(*gogui.Layout)
		visit = func(l *gogui.Layout) {
			if l.Shape.Scrollable {
				t.Fatalf("volume control must not scroll on %d-pixel display: %s", height, l.Shape.ID)
			}
			if l.Shape.ID == "volume" {
				card = l.Shape
			}
			if strings.HasPrefix(l.Shape.ID, "volume-bar-") {
				bars++
			}
			if strings.HasPrefix(l.Shape.ID, "volume-label-") && card != nil && l.Shape.Y+l.Shape.Height > card.Y+card.Height+1 {
				t.Fatalf("expansion control overflows card at height %d: footer bottom %v, card bottom %v", height, l.Shape.Y+l.Shape.Height, card.Y+card.Height)
			}
			for i := range l.Children {
				visit(&l.Children[i])
			}
		}
		visit(&layout)
		padding := gogui.CurrentTheme().PaddingLarge
		if card == nil || card.Height+padding.Top+padding.Bottom > float32(height) {
			t.Fatalf("expanded volume card overflows %d-pixel display", height)
		}
		if bars != 5 {
			t.Fatalf("expected five volume bars, got %d", bars)
		}
	}
}
