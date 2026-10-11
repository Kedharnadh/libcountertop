package input

import (
	"encoding/binary"
	"unsafe"

	"golang.org/x/sys/unix"
)

// EVIOCGABS(abs) = _IOR('E', 0x40+abs, struct input_absinfo), six s32.
const absInfoBytes = 24

// An evdev axis reports only when it moves.
func (d *Device) Abs(code uint16) (int32, error) {
	var b [absInfoBytes]byte
	req := uintptr(2<<30 | absInfoBytes<<16 | 'E'<<8 | (0x40 + uint32(code)))
	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		d.f.Fd(),
		req,
		uintptr(unsafe.Pointer(&b[0])),
	); errno != 0 {
		return 0, errno
	}
	return int32(binary.LittleEndian.Uint32(b[:4])), nil
}
