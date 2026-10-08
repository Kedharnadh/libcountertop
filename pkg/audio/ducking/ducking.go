package ducking

import "math"

const MinimumDB = -40
const MaximumDB = -3
const DefaultDB = -15

func Valid(db float64) bool {
	return !math.IsNaN(db) && !math.IsInf(db, 0) && db >= MinimumDB && db <= MaximumDB
}
func Gain(db float64) float32 { return float32(math.Pow(10, db/20)) }
