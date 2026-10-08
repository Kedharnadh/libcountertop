package preferences

import "sync"

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
	Read            func() Settings
	SaveScreens     func(device, music, video string) error
	RequestIDPrefix string
}

var mu sync.RWMutex
var current Provider

func Configure(provider Provider) { mu.Lock(); current = provider; mu.Unlock() }
func snapshot() Provider          { mu.RLock(); defer mu.RUnlock(); return current }
func Get() Settings {
	if read := snapshot().Read; read != nil {
		return read()
	}
	return Settings{YouTube: YouTube{LiveDelay: 10, Skip: []string{"sponsor"}}}
}
func SaveScreens(device, music, video string) error {
	if save := snapshot().SaveScreens; save != nil {
		return save(device, music, video)
	}
	return nil
}
func RequestIDPrefix() string {
	if prefix := snapshot().RequestIDPrefix; prefix != "" {
		return prefix
	}
	return "countertop"
}
