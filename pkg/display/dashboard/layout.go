package dashboard

import (
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

// positionShare is how much of the height a face gets when it is not centered.
//
// Less than all of it, because top and bottom only mean anything against something: a face given
// the whole screen and told to sit at the top has nowhere to go. The face centers itself in what it
// is handed, so this share is what decides how far up or down that lands.
const positionShare = 0.72

// Box is the part of the screen a face is given.
//
// The whole of it when the clock is centered and full size, a band against one edge otherwise, and
// a share of that band once the clock is asked to be smaller. The face is not told which it got: it
// centers itself in whatever it is handed, which is the same arithmetic in every case and the
// reason a face takes a box at all.
func Box(at config.Position, size config.Size, w, h int) ui.Rect {
	return Place(at, config.AlignCenter, size, w, h)
}

func Place(at config.Position, align config.Align, size config.Size, w, h int) ui.Rect {
	box := ui.Rect{W: w, H: h}

	switch at {
	case config.PositionTop:
		box.H = int(float64(h) * positionShare)
	case config.PositionBottom:
		box.H = int(float64(h) * positionShare)
		box.Y = h - box.H
	}

	switch align {
	case config.AlignLeft:
		box.W = int(float64(w) * positionShare)
	case config.AlignRight:
		box.W = int(float64(w) * positionShare)
		box.X = w - box.W
	}

	return shrink(box, at, align, size.Share())
}

// shrink takes a share of a box, against the edge the position names.
//
// Against the edge rather than about the middle, because the two settings would otherwise fight:
// scaling a top band about its center puts a small clock a third of the way down the screen, which
// is neither the top the position asked for nor anywhere somebody would point at. Top stays against
// the top, bottom against the bottom, and centered stays centered — which is what all three words
// mean whatever size the clock is.
func shrink(in ui.Rect, at config.Position, align config.Align, of float64) ui.Rect {
	if of >= 1 {
		return in
	}

	out := ui.Rect{W: int(float64(in.W) * of), H: int(float64(in.H) * of)}

	switch align {
	case config.AlignLeft:
		out.X = in.X
	case config.AlignRight:
		out.X = in.X + in.W - out.W
	default:
		out.X = in.X + (in.W-out.W)/2
	}

	switch at {
	case config.PositionTop:
		out.Y = in.Y
	case config.PositionBottom:
		out.Y = in.Y + in.H - out.H
	default:
		out.Y = in.Y + (in.H-out.H)/2
	}
	return out
}
