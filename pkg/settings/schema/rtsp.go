package schema

type RTSP struct {
	Enabled bool `json:"enabled"`
}

func DefaultRTSP() RTSP { return RTSP{} }
