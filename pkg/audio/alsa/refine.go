package alsa

import (
	"encoding/binary"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var ioctlHwRefine = ioc(3, 'A', 0x10, hwParamsSize)

type Range struct {
	Formats                  []int
	MinChannels, MaxChannels uint32
	MinRate, MaxRate         uint32
	MinBits, MaxBits         uint32
}

func (r Range) String() string {
	return fmt.Sprintf("formats %v, channels %d-%d, rate %d-%d, sample bits %d-%d",
		r.Formats, r.MinChannels, r.MaxChannels, r.MinRate, r.MaxRate, r.MinBits, r.MaxBits)
}

func Refine(card, device int, capture bool) (Range, error) {
	dir := "p"
	flags := os.O_WRONLY
	if capture {
		dir, flags = "c", os.O_RDONLY
	}
	f, err := os.OpenFile(
		fmt.Sprintf("/dev/snd/pcmC%dD%d%s", card, device, dir),
		flags|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return Range{}, err
	}
	defer f.Close()

	var p hwParams
	p.init()
	if err := ioctl(f.Fd(), ioctlHwRefine, unsafe.Pointer(&p)); err != nil {
		return Range{}, fmt.Errorf("hw_refine: %w", err)
	}

	var r Range
	off := maskOff + paramFormat*maskSize
	for word := range maskSize / 4 {
		bits := binary.LittleEndian.Uint32(p[off+word*4:])
		for b := range 32 {
			if bits&(1<<uint(b)) != 0 {
				r.Formats = append(r.Formats, word*32+b)
			}
		}
	}
	r.MinChannels, r.MaxChannels = p.bounds(paramChannels)
	r.MinRate, r.MaxRate = p.bounds(paramRate)
	r.MinBits, r.MaxBits = p.bounds(paramSampleBits)
	return r, nil
}

func (p *hwParams) bounds(param int) (lo, hi uint32) {
	off := intervalOff + (param-firstInterval)*intervalLen
	return binary.LittleEndian.Uint32(p[off:]), binary.LittleEndian.Uint32(p[off+4:])
}
