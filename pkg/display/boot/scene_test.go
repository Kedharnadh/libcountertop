package boot

import (
	"fmt"
	gui "github.com/go-gui-org/go-gui/gui"
	backend "github.com/ygelfand/libcountertop/pkg/display/gogui"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
	"strings"
	"testing"
)

type sink struct {
	glyphs   int
	uploaded bool
}

func (*sink) Program(int, string, string) error { return nil }
func (s *sink) Texture(_ uint32, _ int, _ int, _ int, _ int, _ int, _ int, pix []byte) error {
	for _, v := range pix {
		if v != 0 {
			s.uploaded = true
			break
		}
	}
	return nil
}
func (s *sink) Frame(_ int, _ [4]float32, _ []float32, runs []uint32) (uint32, uint32, error) {
	s.glyphs = 0
	for i := 0; i < len(runs); i += 40 {
		if runs[i] == 1 {
			s.glyphs += int(runs[i+7])
		}
	}
	return 0, 0, nil
}
func TestProgressFramesContainRowsVersionAndSkip(t *testing.T) {
	for _, size := range [][2]int{{1920, 1200}, {1200, 1920}, {960, 480}} {
		s := &Scene{width: size[0], height: size[1], palette: theme.Default(), version: "test-build", skip: "Skip"}
		s.w = gui.SimpleWindow("test", size[0], size[1], s, func(w *gui.Window) { w.SetView(s.view) })
		r, err := backend.New(&sink{}, s.w)
		if err != nil {
			t.Fatal(err)
		}
		s.r = r
		s.w.Config.OnInit(s.w)
		if err := s.Draw([]startup.Progress{{Name: "Wi-Fi", Doing: "joining"}, {Name: "Audio", Failed: true, Doing: "unavailable"}}, true); err != nil {
			t.Fatal(err)
		}
		var labels strings.Builder
		for _, cmd := range s.w.Renderers() {
			if cmd.Kind == gui.RenderText {
				labels.WriteString(cmd.Text)
				labels.WriteByte('|')
			}
		}
		for _, want := range []string{"Wi-Fi", "joining", "Audio", "unavailable", "test-build", "Skip"} {
			if !strings.Contains(labels.String(), want) {
				t.Errorf("%v frame missing %q: %s", size, want, labels.String())
			}
		}
		if err := s.Draw([]startup.Progress{{Name: "Wi-Fi", Done: true}}, true); err != nil {
			t.Fatal(err)
		}
		for _, cmd := range s.w.Renderers() {
			if cmd.Kind == gui.RenderText && cmd.Text == "Skip" {
				t.Errorf("%v settled frame retained skip", size)
			}
		}
	}
}

func TestBootTextUsesBundledFontAndPaletteBeforeMainGUI(t *testing.T) {
	for _, dark := range []bool{false, true} {
		p := theme.Default().On(dark)
		s := &Scene{width: 1024, height: 600, palette: p, version: "build", skip: "Skip"}
		s.w = gui.SimpleWindow("test", 1024, 600, s, func(w *gui.Window) { w.SetView(s.view) })
		out := &sink{}
		r, err := backend.New(out, s.w)
		if err != nil {
			t.Fatal(err)
		}
		s.r = r
		s.w.Config.OnInit(s.w)
		if err := s.Draw([]startup.Progress{{Name: "wifi", Doing: "joining"}}, true); err != nil {
			t.Fatal(err)
		}
		if out.glyphs == 0 || !out.uploaded {
			t.Errorf("dark=%t: no rasterized glyphs: batches=%d atlas=%t", dark, out.glyphs, out.uploaded)
		}
		found := false
		for _, cmd := range s.w.Renderers() {
			if cmd.Kind != gui.RenderText {
				continue
			}
			if cmd.FontName != "Go" {
				t.Errorf("dark=%t: %q uses %q, want bundled Go font", dark, cmd.Text, cmd.FontName)
			}
			if cmd.Text == "Wi-Fi" {
				found = true
				if cmd.Color != color(p.Text) {
					t.Errorf("dark=%t label color=%v, want %v", dark, cmd.Color, color(p.Text))
				}
			}
		}
		if !found {
			t.Fatalf("dark=%t: no Wi-Fi label", dark)
		}
	}
}

func TestBootLayoutFitsPortraitAndLandscapeWithOversizedTheme(t *testing.T) {
	for _, size := range [][2]int{{1024, 600}, {1200, 1920}, {800, 1280}, {960, 480}} {
		for _, dark := range []bool{false, true} {
			p := theme.Default().On(dark)
			s := &Scene{width: size[0], height: size[1], palette: p, version: "build", skip: "Skip"}
			s.w = gui.SimpleWindow("test", size[0], size[1], s, func(w *gui.Window) { w.SetView(s.view) })
			cfg := bootTheme(p).Cfg
			cfg.TextStyleDef.Size = 200
			cfg.SpacingSmall = 100
			cfg.SpacingMedium = 200
			s.w.SetTheme(gui.ThemeMaker(cfg))
			r, err := backend.New(&sink{}, s.w)
			if err != nil {
				t.Fatal(err)
			}
			s.r = r
			s.w.Config.OnInit(s.w)
			progress := make([]startup.Progress, 18)
			for i := range progress {
				progress[i] = startup.Progress{Name: fmt.Sprintf("part-%02d", i), Doing: "waiting for a deliberately very long network and hardware status"}
			}
			if err := s.Draw(progress, true); err != nil {
				t.Fatal(err)
			}
			skip := Skip(size[0], size[1])
			area := progressArea(size[0], size[1])
			if skip.Y < size[1]*3/4 || skip.X+skip.W > size[0] || skip.Y+skip.H > size[1] {
				t.Errorf("%v: skip outside bottom footer: %+v", size, skip)
			}
			if area.Y+area.H >= skip.Y {
				t.Errorf("%v: progress overlaps footer: area=%+v skip=%+v", size, area, skip)
			}
			found := 0
			for _, cmd := range s.w.Renderers() {
				if cmd.Kind != gui.RenderText {
					continue
				}
				if cmd.Text == "Skip" || cmd.Text == "build" {
					continue
				}
				found++
				if cmd.FontSize > 22 {
					t.Errorf("%v dark=%t: %q font size %v exceeds cap", size, dark, cmd.Text, cmd.FontSize)
				}
				if cmd.Y+cmd.H > float32(skip.Y) || cmd.X < 0 || cmd.X+cmd.W > float32(size[0])+1 {
					t.Errorf("%v dark=%t: text %q outside progress area: x=%v y=%v w=%v h=%v", size, dark, cmd.Text, cmd.X, cmd.Y, cmd.W, cmd.H)
				}
			}
			if found != 36 {
				t.Errorf("%v dark=%t: %d row lines, want 36", size, dark, found)
			}
		}
	}
}

func TestEchoBootKeepsReadableFontSizes(t *testing.T) {
	s := &Scene{width: 960, height: 480, palette: theme.Default(), version: "build", skip: "Skip"}
	s.w = gui.SimpleWindow("test", s.width, s.height, s, func(w *gui.Window) { w.SetView(s.view) })
	r, err := backend.New(&sink{}, s.w)
	if err != nil {
		t.Fatal(err)
	}
	s.r = r
	s.w.Config.OnInit(s.w)
	progress := make([]startup.Progress, 12)
	for i := range progress {
		progress[i] = startup.Progress{Name: fmt.Sprintf("part-%02d", i), Doing: "starting"}
	}
	if err := s.Draw(progress, true); err != nil {
		t.Fatal(err)
	}
	for _, cmd := range s.w.Renderers() {
		if cmd.Kind != gui.RenderText {
			continue
		}
		if strings.HasPrefix(cmd.Text, "part-") && cmd.FontSize < 20 {
			t.Errorf("Echo label %q uses %v px, want at least 20", cmd.Text, cmd.FontSize)
		}
		if cmd.Text == "starting" && cmd.FontSize < 14 {
			t.Errorf("Echo status uses %v px, want at least 14", cmd.FontSize)
		}
	}
}
