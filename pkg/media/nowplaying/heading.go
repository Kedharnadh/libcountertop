package nowplaying

func Heading(now Now) string {
	if now.Title != "" {
		return now.Title
	}
	if now.Paused {
		return "Paused"
	}
	if now.Playing {
		return "Playing"
	}
	return "Nothing playing"
}
