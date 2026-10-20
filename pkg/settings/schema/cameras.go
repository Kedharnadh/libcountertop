package schema

const (
	CameraScaleLeast  = 1
	CameraScaleMost   = 5
	CameraRefreshMost = 60
	CameraRotateMost  = 120
)

type Cameras struct {
	Pick    HomePick `json:"pick"`
	Scale   int      `json:"scale"`
	Refresh int      `json:"refresh"`
	Rotate  int      `json:"rotate"`
	URL     string   `json:"url,omitempty"`
}

func DefaultCameras() Cameras {
	return Cameras{Pick: HomePick{All: true}, Scale: 3, Refresh: 5}
}

func (c Cameras) Clone() Cameras {
	c.Pick = c.Pick.Clone()
	return c
}
