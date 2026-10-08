package bt

import (
	"github.com/ygelfand/libcountertop/pkg/bluetooth/l2cap"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/sdp"
)

const (
	SinkPSM    = l2cap.PSMAVDTP
	ControlPSM = l2cap.PSMAVCTP
	BrowsePSM  = l2cap.PSMAVCTPBrowse
)

const RemoteFeatures = sdp.CategoryPlayer | sdp.CategoryAmplifier |
	sdp.CategoryTuner | sdp.CategoryMenu

const CoverArt = sdp.FeatureImageProperties | sdp.FeatureImage | sdp.FeatureThumbnail

func Records(name string) []sdp.Record {
	return []sdp.Record{
		sdp.AudioSink(0x00010000, name, SinkPSM, sdp.FeatureSpeaker),

		sdp.RemoteControl(0x00010001, name, ControlPSM, BrowsePSM,
			sdp.UUIDAVRemoteControlController, RemoteFeatures|CoverArt),

		sdp.RemoteControl(0x00010002, name, ControlPSM, BrowsePSM,
			sdp.UUIDAVRemoteControlTarget, RemoteFeatures),
	}
}

func Record(name string) sdp.Record { return Records(name)[0] }
