package dhcp

import (
	"encoding/binary"
	"net"
)

type arpFrame struct {
	op uint16

	senderMAC net.HardwareAddr
	senderIP  net.IP
	targetIP  net.IP
}

const (
	arpRequest = 1
	arpReply   = 2
)

func readARP(frame []byte) (arpFrame, bool) {
	if len(frame) < ethHeader+arpPayload {
		return arpFrame{}, false
	}
	if binary.BigEndian.Uint16(frame[12:14]) != 0x0806 {
		return arpFrame{}, false
	}

	body := frame[ethHeader:]
	if binary.BigEndian.Uint16(body[0:2]) != 1 || binary.BigEndian.Uint16(body[2:4]) != 0x0800 {
		return arpFrame{}, false
	}
	if body[4] != 6 || body[5] != 4 {
		return arpFrame{}, false
	}

	op := binary.BigEndian.Uint16(body[6:8])
	if op != arpRequest && op != arpReply {
		return arpFrame{}, false
	}

	return arpFrame{
		op:        op,
		senderMAC: net.HardwareAddr(body[8:14]),
		senderIP:  net.IP(body[14:18]),
		targetIP:  net.IP(body[24:28]),
	}, true
}

func (a arpFrame) ours(mine net.HardwareAddr) bool {
	return len(mine) == 6 && a.senderMAC.String() == mine.String()
}
