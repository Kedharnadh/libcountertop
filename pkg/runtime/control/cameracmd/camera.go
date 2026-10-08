package cameracmd

import (
	"fmt"
	"os"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/live"
	"github.com/ygelfand/libcountertop/pkg/runtime/control"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const DefaultStill = "/data/local/tmp/still.jpg"

func Still(take func() ([]byte, error), args []string) (string, error) {
	at := DefaultStill
	if len(args) > 0 && args[0] != "" {
		at = args[0]
	}

	began := time.Now()
	pic, err := take()
	if err != nil {
		return "", err
	}
	took := time.Since(began)

	if err := os.WriteFile(at, pic, 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("  %-9s %d bytes in %s\n  %-9s %s\n",
		"picture", len(pic), took.Round(time.Millisecond), "wrote", at), nil
}

func Rows(cam *live.Camera) []control.Setting[schema.Camera] {
	knobs := cam.Table().Rows()
	rows := make([]control.Setting[schema.Camera], 0, len(knobs))
	for _, s := range knobs {
		rows = append(rows, control.Setting[schema.Camera]{
			Name:  "camera." + s.Name,
			Field: "Camera.Settings",
			Read: func(c schema.Camera) string {
				k, _ := cam.Table().Configured(cam.Defaults(), c.Settings)
				return s.Read(&k)
			},
			Write: func(v string) error {
				k := cam.Defaults()
				if err := s.Put(&k, v); err != nil {
					return err
				}
				return cam.Set(s.Name, v)
			},
		})
	}
	return rows
}
