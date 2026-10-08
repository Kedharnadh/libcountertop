package turn

type Phase int

const (
	Idle Phase = iota
	Listening
	Thinking
	Replying
)

func (p Phase) String() string {
	switch p {
	case Listening:
		return "listening"
	case Thinking:
		return "thinking"
	case Replying:
		return "replying"
	}
	return "idle"
}

type Showing struct {
	Phase Phase
	Slot  int

	Said  string
	Reply string
}
