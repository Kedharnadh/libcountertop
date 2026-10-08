package brightness

import (
	"testing"
	"time"
)

const (
	lit      = 190.0
	leastLux = 15
)

func TestFirstReadingIsAlwaysSent(t *testing.T) {
	var sent Told

	if !sent.Worth(lit, leastLux) {
		t.Fatal("the first reading was held back, so Home Assistant would start with nothing")
	}
}

func TestAReadingThatHasNotMovedIsHeldBack(t *testing.T) {
	var sent Told
	sent.Worth(lit, leastLux)

	for _, lux := range []float64{190, 196, 184, 193, 187} {
		if sent.Worth(lux, leastLux) {
			t.Errorf("%.0f lux was reported, which is inside the part's own noise", lux)
		}
	}
}

func TestAReadingThatHasMovedIsSent(t *testing.T) {
	var sent Told
	sent.Worth(lit, leastLux)

	if !sent.Worth(380, leastLux) {
		t.Fatal("the room doubling was not reported")
	}
}

func TestNearDarkIsHeldToTheFloorRatherThanTheFraction(t *testing.T) {
	var sent Told
	sent.Worth(20, leastLux)

	if sent.Worth(28, leastLux) {
		t.Error("8 lux was reported: five percent of a dark room is under the noise")
	}
	if !sent.Worth(40, leastLux) {
		t.Error("20 lux was not reported, which is past the floor")
	}
}

func TestDriftingPastTheThresholdReportsOnce(t *testing.T) {
	var sent Told
	sent.Worth(400, leastLux)

	var reports int
	for _, lux := range []float64{408, 416, 424, 432} {
		if sent.Worth(lux, leastLux) {
			reports++
		}
	}

	if reports != 1 {
		t.Errorf("a slow drift reported %d times, want once", reports)
	}
}

func TestAStillRoomIsStillHeardFromEventually(t *testing.T) {
	var sent Told
	sent.Worth(lit, leastLux)

	if sent.Worth(lit, leastLux) {
		t.Fatal("an unchanged reading was reported straight away")
	}

	sent.at = time.Now().Add(-stale - time.Second)

	if !sent.Worth(lit, leastLux) {
		t.Error("nothing was reported for an afternoon of an unchanging room")
	}
}
