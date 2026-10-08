package firmware

import (
	"context"
	"log/slog"
	"slices"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/runtime/component"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	EventInstalled  = "installed"
	EventRolledBack = "rolled_back"
	EventFailed     = "failed"
)

type Release struct {
	Version    string
	Notes      string
	ReleaseURL string
}

type Source[M any] interface {
	Fetch(ctx context.Context, channel string) (M, error)
	Describe(found M) Release
	Install(ctx context.Context, found M, progress func(float32)) error
	Restart(why string)
}

type Trial interface {
	RolledBack() string
	Commit()
}

type Settings interface {
	Read() schema.Update
	Channel(label string) error
	LastVersion(version string) error
}

type Activity interface {
	Refresh()
	Done()
}

type Indicator interface{ Begin() Activity }

type Dependencies[M any] struct {
	Source    Source[M]
	Settings  Settings
	Product   string
	Version   string
	Channels  []string
	Busy      Indicator
	OnFailure func()
}

type Upgrade struct {
	Version string
	At      float32
}

func (up Upgrade) Active() bool { return up.Version != "" }

type Firmware[M any] struct {
	Upgrading hook.Hook[Upgrade]

	deps Dependencies[M]

	entity  *esphome.Update
	channel *esphome.Select
	look    *esphome.Button
	status  *esphome.TextSensor
	events  *esphome.Event

	mu      sync.Mutex
	found   M
	upgrade Upgrade

	announced     sync.Once
	rolledBack    string
	rebootPending bool
}

func New[M any](deps Dependencies[M]) *Firmware[M] {
	u := &Firmware[M]{deps: deps}

	u.entity = &esphome.Update{
		Base: esphome.Base{
			ObjectID: "firmware",
			Name:     "Firmware",
			Icon:     "mdi:package-up",
		},
		DeviceClass: "firmware",
		OnCommand:   u.command,
	}
	u.publish(Release{Version: deps.Version})

	u.channel = &esphome.Select{
		Base: esphome.Base{
			ObjectID: "update_channel",
			Name:     "Update channel",
			Icon:     "mdi:source-branch",
			Category: esphome.CategoryDiagnostic,
		},
		Options:   slices.Clone(deps.Channels),
		OnCommand: u.choose,
	}
	u.channel.Set(u.Channel())

	// Home Assistant asks for an update check only through homeassistant.update_entity.
	u.look = &esphome.Button{
		Base: esphome.Base{
			ObjectID: "check_for_updates",
			Name:     "Check for updates",
			Icon:     "mdi:cloud-search",
			Category: esphome.CategoryDiagnostic,
		},
		OnPress: u.checkLater,
	}

	u.status = &esphome.TextSensor{
		Base: esphome.Base{
			ObjectID: "update_status",
			Name:     "Update status",
			Icon:     "mdi:package-variant",
			Category: esphome.CategoryDiagnostic,
		},
	}

	types := []string{EventInstalled, EventFailed}
	if t, ok := deps.Source.(Trial); ok {
		u.rolledBack = t.RolledBack()
		types = []string{EventInstalled, EventRolledBack, EventFailed}
	}
	u.events = &esphome.Event{
		Base: esphome.Base{
			ObjectID: "update_outcome",
			Name:     "Update outcome",
			Icon:     "mdi:package-up",
		},
		Types: types,
	}
	return u
}

func (u *Firmware[M]) Name() string { return "firmware" }

func (u *Firmware[M]) Entities() []esphome.Entity {
	return []esphome.Entity{u.entity, u.channel, u.look, u.status, u.events}
}

func (u *Firmware[M]) Restore() { u.channel.Set(u.Channel()) }

func (u *Firmware[M]) RebootPending(pending bool) {
	u.mu.Lock()
	u.rebootPending = pending
	u.mu.Unlock()
}

func (u *Firmware[M]) Announce() {
	u.announced.Do(func() {
		u.mu.Lock()
		pending := u.rebootPending
		u.mu.Unlock()

		version := u.deps.Version
		finish := "installed " + version + ", reboot to finish"
		if u.deps.Settings.Read().LastVersion == version {
			if pending {
				u.status.Set(component.Fit(finish))
			}
			return
		}

		event, detail := EventInstalled, "installed "+version
		if u.rolledBack != "" {
			event, detail = EventRolledBack, "rolled back from "+u.rolledBack
		} else if pending {
			detail = finish
		}
		u.Settled(event, detail)

		if err := u.deps.Settings.LastVersion(version); err != nil {
			slog.Error("saving the reported version failed", "err", err)
		}
	})
}

func (u *Firmware[M]) Settled(event, status string) {
	u.status.Set(component.Fit(status))
	u.events.Trigger(event)
}

func (u *Firmware[M]) Channel() string {
	saved := u.deps.Settings.Read().Channel
	if len(u.deps.Channels) == 0 || slices.Contains(u.deps.Channels, saved) {
		return saved
	}
	return u.deps.Channels[0]
}

func (u *Firmware[M]) choose(label string) {
	if !slices.Contains(u.deps.Channels, label) {
		slog.Warn("unknown option", "setting", u.channel.ObjectID, "value", label)
		return
	}
	u.channel.Set(label)
	if err := u.deps.Settings.Channel(label); err != nil {
		slog.Error("saving a setting failed", "setting", u.channel.ObjectID, "err", err)
	}
	slog.Info("setting changed", "setting", u.channel.ObjectID, "using", label)
	u.checkLater()
}

func (u *Firmware[M]) Check(ctx context.Context) {
	channel := u.Channel()

	found, err := u.deps.Source.Fetch(ctx, channel)
	if err != nil {
		slog.Error("checking for an update failed", "channel", channel, "err", err)
		return
	}

	u.mu.Lock()
	u.found = found
	u.mu.Unlock()

	release := u.deps.Source.Describe(found)
	slog.Info(
		"update check",
		"channel",
		channel,
		"running",
		u.deps.Version,
		"offered",
		release.Version,
	)
	u.publish(release)
}

func (u *Firmware[M]) checkLater() {
	safe.Go("update check", func() { u.Check(context.Background()) })
}

func (u *Firmware[M]) command(cmd esphome.UpdateCommand) {
	switch cmd {
	case esphome.UpdateCheck:
		u.checkLater()

	case esphome.UpdateInstall:
		safe.Go("update install", func() { u.Install(context.Background()) })
	}
}

func (u *Firmware[M]) Install(ctx context.Context) {
	found, err := u.deps.Source.Fetch(ctx, u.Channel())
	u.mu.Lock()
	if err != nil {
		slog.Warn("re-reading the channel failed, using the last check", "err", err)
		found = u.found
	} else {
		u.found = found
	}
	u.mu.Unlock()

	release := u.deps.Source.Describe(found)
	if release.Version == "" || release.Version == u.deps.Version {
		slog.Warn("an install was asked for with nothing to install", "running", u.deps.Version)
		return
	}

	refresh := func() {}
	if u.deps.Busy != nil {
		working := u.deps.Busy.Begin()
		defer working.Done()
		refresh = working.Refresh
	}

	u.progress(release, 0)
	err = u.deps.Source.Install(ctx, found, func(at float32) {
		refresh()
		u.progress(release, at)
	})

	// The update entity has no way to report a failed install.
	if err != nil {
		slog.Error("installing an update failed", "version", release.Version, "err", err)
		u.upgrading(Upgrade{})
		u.publish(release)
		u.Settled(EventFailed, "installing "+release.Version+" failed: "+err.Error())
		if u.deps.OnFailure != nil {
			u.deps.OnFailure()
		}
		return
	}
	u.deps.Source.Restart("update to " + release.Version)
}

func (u *Firmware[M]) progress(release Release, at float32) {
	state := u.state(release)
	state.InProgress, state.Progress = true, at*100
	u.entity.Set(state)
	u.upgrading(Upgrade{Version: release.Version, At: at})
}

func (u *Firmware[M]) upgrading(up Upgrade) {
	u.mu.Lock()
	u.upgrade = up
	u.mu.Unlock()
	u.Upgrading.Emit(up)
}

func (u *Firmware[M]) Upgrade() Upgrade {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.upgrade
}

func (u *Firmware[M]) publish(release Release) { u.entity.Set(u.state(release)) }

func (u *Firmware[M]) state(release Release) esphome.UpdateState {
	return esphome.UpdateState{
		CurrentVersion: u.deps.Version,
		LatestVersion:  release.Version,
		Title:          u.deps.Product + " (" + u.Channel() + " channel)",
		ReleaseSummary: release.Notes,
		ReleaseURL:     release.ReleaseURL,
	}
}
