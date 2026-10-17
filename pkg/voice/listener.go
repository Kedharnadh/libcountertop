package voice

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/inference/whistle"
	"github.com/ygelfand/libcountertop/pkg/runtime/payload"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

const Group setting.Group = "Voice"

const depth = 3

type Options struct {
	Helper   string
	Binary   []byte
	ModelDir string
	Client   *http.Client
	Source   whistle.Source
	Paused   func() bool
	Settings storage.Section[schema.Voice]
	Device   uint32
	Clips    string
}

type Listener struct {
	*Commands

	opts   Options
	engine *whistle.Client
	knobs  *setting.Controls[schema.Voice]

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewListener(o Options) *Listener {
	if o.Client == nil {
		o.Client = &http.Client{Timeout: 10 * time.Minute}
	}
	engine := whistle.New(whistle.Options{
		Helper: o.Helper,
		Model:  whistle.Whistle.Path(o.ModelDir),
		Depth:  depth,
		Source: o.Source,
		Paused: o.Paused,
		Clips:  o.Clips,
	})
	l := &Listener{Commands: NewCommands(engine), opts: o, engine: engine}
	engine.Text.Listen(func(t whistle.Transcript) { l.Hear(t.Text) })
	l.knobs = &setting.Controls[schema.Voice]{
		Table:  NewTable(),
		Device: o.Device,
		Read:   o.Settings.Read,
		Save:   l.save,
	}
	return l
}

func NewTable() *setting.Table[schema.Voice] {
	return setting.NewTable("voice", []setting.Group{Group}, []setting.Setting[schema.Voice]{
		setting.Flag(
			setting.Setting[schema.Voice]{
				Name:  "commands",
				Group: Group,
				ID:    "voice_commands",
				Icon:  "mdi:account-voice",
			},
			func(c *schema.Voice) *bool { return &c.Commands },
		),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
}

func (l *Listener) Name() string { return "voice" }

func (l *Listener) Entities() []esphome.Entity { return l.knobs.Entities() }

func (l *Listener) Settings() *setting.Controls[schema.Voice] { return l.knobs }

func (l *Listener) Restore(c schema.Voice) {
	l.knobs.PublishFrom(c)
	if c.Commands {
		l.Start()
	}
}

func (l *Listener) Start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.wg.Go(func() { l.run(ctx) })
}

func (l *Listener) Stop() {
	l.mu.Lock()
	cancel := l.cancel
	l.cancel = nil
	l.mu.Unlock()
	if cancel != nil {
		cancel()
		l.wg.Wait()
	}
}

func (l *Listener) save(s setting.Setting[schema.Voice], v string) error {
	if err := setting.Store(l.opts.Settings, s, v); err != nil {
		return err
	}
	if l.opts.Settings.Read().Commands {
		l.Start()
	} else {
		l.Stop()
	}
	return nil
}

func (l *Listener) run(ctx context.Context) {
	if err := payload.Install(l.opts.Helper, l.opts.Binary); err != nil {
		slog.Error("voice helper not installed", "path", l.opts.Helper, "err", err)
		return
	}
	backoff := 5 * time.Second
	for {
		_, err := whistle.Whistle.Fetch(ctx, l.opts.Client, l.opts.ModelDir)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			return
		}
		slog.Warn("voice model not fetched", "err", err, "retry_in", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, 10*time.Minute)
	}
	if err := l.engine.Run(ctx); err != nil {
		slog.Error("voice commands stopped", "err", err)
	}
}
