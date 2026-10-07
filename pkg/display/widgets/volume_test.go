package widgets

import (
	"testing"

	gogui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/display/interaction"
	"github.com/ygelfand/libcountertop/pkg/display/style"
)

func TestExpandedVolumeCardFitsShortDisplays(t *testing.T) {
	for _, height := range []int{360, 480, 600} {
		presses := interaction.New()
		kit := Toolkit{Kit: func() style.Kit { return style.ByName("standard").Kit }, Pressable: presses.Pressable}
		w := gogui.SimpleWindow("volume", 960, height, &struct{}{}, func(*gogui.Window) {})
		view := kit.VolumeCard(w, VolumeOptions{
			Current: VolumeChoice{Key: "main", Label: "Device", Level: 40}, Open: true,
			Choices: []VolumeChoice{{Key: "main"}, {Key: "media", Label: "Media"}, {Key: "voice", Label: "Voice"}, {Key: "alerts", Label: "Alerts"}, {Key: "feedback", Label: "Feedback"}},
			Grip:    func(v gogui.View) gogui.View { return v }, Expand: func() {}, Pick: func(string) {}, Linger: func() {}, Dismiss: func() {}, Set: func(int) {},
		})
		layout := gogui.GenerateViewLayout(view, w)
		var card, choices *gogui.Shape
		var visit func(*gogui.Layout)
		visit = func(l *gogui.Layout) {
			if l.Shape.ID == "volume" {
				card = l.Shape
			}
			if l.Shape.ID == "volume-choices" {
				choices = l.Shape
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
		if choices == nil || choices.Height <= 0 || choices.Height > card.Height {
			t.Fatalf("stream choices are not bounded on %d-pixel display", height)
		}
	}
}
