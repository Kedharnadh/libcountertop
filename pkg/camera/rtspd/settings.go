package rtspd

import (
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const Group setting.Group = "RTSP"

func NewTable() *setting.Table[schema.RTSP] {
	return setting.NewTable("rtsp", []setting.Group{Group}, []setting.Setting[schema.RTSP]{
		setting.Flag(
			setting.Setting[schema.RTSP]{
				Name:  "enabled",
				Group: Group,
				ID:    "rtsp",
				Icon:  "mdi:cctv",
			},
			func(c *schema.RTSP) *bool { return &c.Enabled },
		),
	}, setting.Messages{Text: say.T, Missing: say.Missing}).Local()
}

func Controls(
	device uint32,
	read func() schema.RTSP,
	enable func(bool) error,
) *setting.Controls[schema.RTSP] {
	return &setting.Controls[schema.RTSP]{
		Table:  NewTable(),
		Device: device,
		Read:   read,
		Save: func(s setting.Setting[schema.RTSP], v string) error {
			at := read()
			if err := s.Write(&at, v); err != nil {
				return err
			}
			return enable(at.Enabled)
		},
	}
}
