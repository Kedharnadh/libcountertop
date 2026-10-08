package preferences

type YouTube struct {
	Device, Music, Video string
	Skip                 []string
	OnDemand             bool
	LiveDelay            int
}
type Prime struct{ Persist, SkipIntro bool }
type Settings struct {
	YouTube YouTube
	Prime   Prime
}
type Provider struct {
	Read   func() Settings
	Save   func(device, music, video string) error
	Prefix string
}

func (p Provider) Settings() Settings {
	if p.Read != nil {
		return p.Read()
	}
	return Settings{YouTube: YouTube{LiveDelay: 10, Skip: []string{"sponsor"}}}
}

func (p Provider) SaveScreens(device, music, video string) error {
	if p.Save != nil {
		return p.Save(device, music, video)
	}
	return nil
}

func (p Provider) RequestIDPrefix() string {
	if p.Prefix != "" {
		return p.Prefix
	}
	return "countertop"
}
