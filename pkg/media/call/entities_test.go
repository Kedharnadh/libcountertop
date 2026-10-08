package call

import (
	"errors"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type callSettings struct {
	value schema.Call
	err   error
}

func (s *callSettings) Read() schema.Call { return s.value }
func (s *callSettings) Update(change func(*schema.Call)) error {
	if s.err != nil {
		return s.err
	}
	change(&s.value)
	return nil
}

func TestFailedCallSettingPublishesSavedValue(t *testing.T) {
	settings := &callSettings{value: schema.DefaultCall(), err: errors.New("write failed")}
	c := New(Dependencies{Settings: settings, Shell: shell.New()})
	t.Cleanup(c.Close)
	c.Restore(settings.Read())
	c.SetIncoming(false)
	if !advertised(c, "incoming") || !settings.Read().Incoming {
		t.Fatal("failed save changed the advertised or persisted incoming setting")
	}
	settings.err = nil
	c.SetAutoVideo(true)
	if !settings.Read().Incoming || !advertised(c, "auto_video") {
		t.Fatal("changing auto video lost another setting or was not published")
	}
}

func advertised(c *Calls, name string) bool {
	return c.Controls().Entity(name).(*esphome.Switch).Get()
}
