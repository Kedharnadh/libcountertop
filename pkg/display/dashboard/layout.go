package dashboard

import (
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const positionShare = 0.72

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
