package schema

type RTSP struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func DefaultRTSP() RTSP { return RTSP{Username: "admin"} }
