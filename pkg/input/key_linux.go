package input

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// EVIOCGBIT(EV_KEY, n) = _IOR('E', 0x20+EV_KEY, n), EVIOCGKEY(n) = _IOR('E', 0x18, n), KEY_MAX = 0x2ff.
const (
	keyBytes     = 96
	eviocgbitKey = 2<<30 | keyBytes<<16 | 'E'<<8 | (0x20 + EvKey)
	eviocgkey    = 2<<30 | keyBytes<<16 | 'E'<<8 | 0x18
)

func (d *Device) keyBits(req uintptr) ([keyBytes]byte, error) {
	var b [keyBytes]byte
	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		d.f.Fd(),
		req,
		uintptr(unsafe.Pointer(&b[0])),
	); errno != 0 {
		return b, errno
	}
	return b, nil
}

func bitSet(b [keyBytes]byte, code uint16) bool {
	return int(code>>3) < keyBytes && b[code>>3]&(1<<(code&7)) != 0
}

func (d *Device) HasKey(code uint16) bool {
	b, err := d.keyBits(eviocgbitKey)
	return err == nil && bitSet(b, code)
}

// An evdev key reports only when it changes; EVIOCGKEY reads the level now.
func (d *Device) Key(code uint16) (bool, error) {
	b, err := d.keyBits(eviocgkey)
	if err != nil {
		return false, err
	}
	return bitSet(b, code), nil
}
