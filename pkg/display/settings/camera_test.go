package settings

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
)

func TestCameraNavigationReusesPreviewPages(t *testing.T) {
	table := setting.NewTable[struct{}]("camera", []setting.Group{"capture"}, nil, setting.Messages{})
	wrapped := 0
	p := NewCameraPages(CameraOptions[struct{}]{Table: func() *setting.Table[struct{}] { return table }, Wrap: func(page *shell.Page) *shell.Page { wrapped++; return page }})
	if p.Page() != p.Page() {
		t.Fatal("camera root was recreated")
	}
	if p.Section("capture") != p.Section("capture") {
		t.Fatal("camera section was recreated")
	}
	knob := setting.Setting[struct{}]{Name: "whitebalance"}
	if p.Choice(knob) != p.Choice(knob) {
		t.Fatal("camera choice was recreated")
	}
	if wrapped != 3 {
		t.Fatalf("registered %d preview pages, want 3", wrapped)
	}
}
