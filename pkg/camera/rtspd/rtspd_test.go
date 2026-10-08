package rtspd

import (
	"reflect"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/camera/helper"
	"github.com/ygelfand/libcountertop/pkg/camera/onvif"
	"github.com/ygelfand/libcountertop/pkg/camera/session"
)

type sizes []session.Size

func (sizes) Join(int) (*session.Session, <-chan helper.Frame, error) { return nil, nil, nil }
func (sizes) Leave(*session.Session, <-chan helper.Frame)             {}
func (s sizes) Sizes() []session.Size                                 { return s }

func TestEachServedStreamIsAProfile(t *testing.T) {
	s := New(Options{
		Source:  sizes{{Width: 1280, Height: 720}, {Width: 640, Height: 368}},
		Served:  func() []int { return []int{0, 1} },
		FPS:     30,
		Bitrate: func(w, h int) int { return w * h },
	})

	want := []onvif.Profile{
		{
			Token:   "main",
			Name:    "main",
			Width:   1280,
			Height:  720,
			FPS:     30,
			Bitrate: 1280 * 720,
			Path:    "main",
		},
		{
			Token:   "sub",
			Name:    "sub",
			Width:   640,
			Height:  368,
			FPS:     30,
			Bitrate: 640 * 368,
			Path:    "sub",
		},
	}
	if got := s.Profiles(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestOnlyServedStreamsAreProfiles(t *testing.T) {
	s := New(Options{
		Source:  sizes{{Width: 1280, Height: 720}, {Width: 640, Height: 368}},
		Served:  func() []int { return []int{0} },
		Bitrate: func(int, int) int { return 0 },
	})

	if got := s.Profiles(); len(got) != 1 || got[0].Path != "main" {
		t.Errorf("got %+v, want only main", got)
	}
}

func TestStreamNamesOutsideTheRangeAreBlank(t *testing.T) {
	for _, i := range []int{-1, len(paths)} {
		if got := StreamName(i); got != "" {
			t.Errorf("StreamName(%d) = %q, want blank", i, got)
		}
	}
}
