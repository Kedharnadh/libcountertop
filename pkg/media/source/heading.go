package source

// Heading is the track title or its current playback state.
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
