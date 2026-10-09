package menu

type Steps int

func (n Steps) Index(level int) int {
	last := int(n) - 1
	if last <= 0 {
		return 0
	}
	return min(max((level*last+50)/100, 0), last)
}

func (n Steps) Level(i int) int {
	last := int(n) - 1
	if last <= 0 {
		return 0
	}
	return min(max(i, 0), last) * 100 / last
}

func (n Steps) Snap(level int) int { return n.Level(n.Index(level)) }
