package schema

// RTSP is whether the camera is served over RTSP.
type RTSP struct {
	Enabled bool `json:"enabled"`
}

func DefaultRTSP() RTSP { return RTSP{} }

// RTSPWriter changes it.
