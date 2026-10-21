package rtc

import (
	"context"
	"log/slog"
	"sync"

	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
)

var watchedH264 = []string{
	"level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f",
	"level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=4d001f",
	"level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=640032",
}

var heardAudio = []webrtc.RTPCodecParameters{
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   PlayRate,
			Channels:    PlayChannels,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		PayloadType: opusType,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypePCMU,
			ClockRate: g711Rate,
		},
		PayloadType: 0,
	},
	{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypePCMA,
			ClockRate: g711Rate,
		},
		PayloadType: 8,
	},
}

type Viewer struct {
	pc      *webrtc.PeerConnection
	count   counters
	workers sync.WaitGroup
	closed  sync.Once
}

func NewViewer(
	show func(Frame),
	play func([]int16),
	report func(webrtc.PeerConnectionState),
) (*Viewer, error) {
	built, err := viewerAPI()
	if err != nil {
		return nil, err
	}
	pc, err := built.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, err
	}
	kinds := []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo}
	if play != nil {
		kinds = append(kinds, webrtc.RTPCodecTypeAudio)
	}
	for _, kind := range kinds {
		if _, err := pc.AddTransceiverFromKind(
			kind,
			webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
		); err != nil {
			_ = pc.Close()
			return nil, err
		}
	}
	v := &Viewer{pc: pc}
	pc.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		switch {
		case t.Kind() == webrtc.RTPCodecTypeVideo:
			v.workers.Go(func() { follow(t, pc, &v.count, show) })
		case t.Kind() == webrtc.RTPCodecTypeAudio && play != nil:
			slog.Info("camera audio", "codec", t.Codec().MimeType, "rate", t.Codec().ClockRate)
			v.workers.Go(func() { hear(t, &v.count, play) })
		}
	})
	if report != nil {
		pc.OnConnectionStateChange(report)
	}
	return v, nil
}

func (v *Viewer) Offer(ctx context.Context) (string, error) {
	offer, err := v.pc.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	gathered := webrtc.GatheringCompletePromise(v.pc)
	if err := v.pc.SetLocalDescription(offer); err != nil {
		return "", err
	}
	select {
	case <-gathered:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return v.pc.LocalDescription().SDP, nil
}

func (v *Viewer) Answered(answer string) error {
	return v.pc.SetRemoteDescription(
		webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: answer},
	)
}

func (v *Viewer) Received() (frames, keys uint64) {
	return v.count.videoRecv.Load(), v.count.videoKeys.Load()
}

func (v *Viewer) Close() error {
	var err error
	v.closed.Do(func() {
		err = v.pc.Close()
		v.workers.Wait()
	})
	return err
}

func viewerAPI() (*webrtc.API, error) {
	m := &webrtc.MediaEngine{}
	for i, fmtp := range watchedH264 {
		if err := m.RegisterCodec(webrtc.RTPCodecParameters{
			RTPCodecCapability: webrtc.RTPCodecCapability{
				MimeType:    webrtc.MimeTypeH264,
				ClockRate:   videoClock,
				SDPFmtpLine: fmtp,
			},
			PayloadType: webrtc.PayloadType(h264Type + 2*i),
		}, webrtc.RTPCodecTypeVideo); err != nil {
			return nil, err
		}
	}
	for _, c := range heardAudio {
		if err := m.RegisterCodec(c, webrtc.RTPCodecTypeAudio); err != nil {
			return nil, err
		}
	}
	reg := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, reg); err != nil {
		return nil, err
	}
	return webrtc.NewAPI(
		webrtc.WithMediaEngine(m),
		webrtc.WithInterceptorRegistry(reg),
		webrtc.WithSettingEngine(settings()),
	), nil
}
