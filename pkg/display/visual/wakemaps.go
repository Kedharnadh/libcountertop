package visual

import (
	"image"
	"math"
	"sync"
)

const (
	cloudSpan  = 8
	cloudWidth = 1024
	cloudRows  = 512
	hillSpan   = 2
	hillWidth  = 1024
)

var sunriseMaps = sync.OnceValue(func() map[int]*image.RGBA {
	return map[int]*image.RGBA{1: cloudMap(), 2: hillMap()}
})

func cloudMap() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, cloudWidth, cloudRows))
	for y := range cloudRows {
		for x := range cloudWidth {
			n := tiledFBM(
				float64(x)/cloudWidth*cloudSpan,
				float64(y)/cloudRows*cloudSpan,
				cloudSpan,
				5,
			)
			v := uint8(math.Round(clamp01(n) * 255))
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = v, v, v, 255
		}
	}
	return img
}

func hillMap() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, hillWidth, 1))
	for x := range hillWidth {
		px := (float64(x) + 0.5) / hillWidth * hillSpan
		i := img.PixOffset(x, 0)
		img.Pix[i] = unit8(shaderFBM(px*1.8+3, 1))
		img.Pix[i+1] = unit8(shaderNoise(px*5, 2))
		img.Pix[i+2] = unit8(shaderFBM(px*1.2+9, 4))
		img.Pix[i+3] = 255
	}
	return img
}

func unit8(v float64) uint8 { return uint8(math.Round(clamp01(v) * 255)) }

func shaderHash(x, y float64) float64 {
	fract := func(v float64) float64 { return v - math.Floor(v) }
	a, b, c := fract(x*0.1031), fract(y*0.1031), fract(x*0.1031)
	d := a*(b+33.33) + b*(c+33.33) + c*(a+33.33)
	a, b, c = a+d, b+d, c+d
	return fract((a + b) * c)
}

func smooth(f float64) float64 { return f * f * (3 - 2*f) }

func lattice(x, y float64, at func(ix, iy float64) float64) float64 {
	ix, iy := math.Floor(x), math.Floor(y)
	fx, fy := smooth(x-ix), smooth(y-iy)
	a, b := at(ix, iy), at(ix+1, iy)
	c, d := at(ix, iy+1), at(ix+1, iy+1)
	return a + (b-a)*fx + (c-a)*fy + (a-b-c+d)*fx*fy
}

func shaderNoise(x, y float64) float64 { return lattice(x, y, shaderHash) }

func shaderFBM(x, y float64) float64 {
	v, amp := 0.0, 0.5
	for range 5 {
		v += amp * shaderNoise(x, y)
		x, y = x*2.03+17.1, y*2.03+9.7
		amp *= 0.5
	}
	return v
}

func tiledFBM(x, y float64, period, octaves int) float64 {
	v, amp := 0.0, 0.5
	for range octaves {
		p := float64(period)
		v += amp * lattice(x, y, func(ix, iy float64) float64 {
			return shaderHash(math.Mod(math.Mod(ix, p)+p, p), iy)
		})
		x, y, period = x*2, y*2+17.1, period*2
		amp *= 0.5
	}
	return v
}
