package airplay

import "image"

func fit(w, h int, most image.Point) (int, int) {
	boxW, boxH := max(most.X, most.Y), min(most.X, most.Y)
	if h > w {
		boxW, boxH = boxH, boxW
	}
	scale := min(1, float64(boxW)/float64(w), float64(boxH)/float64(h))
	return int(float64(w)*scale) &^ 1, int(float64(h)*scale) &^ 1
}
