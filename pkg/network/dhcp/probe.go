package dhcp

import (
	"encoding/binary"
	"net"
)

func Probe(iface string, ip net.IP) (bool, error) { return probe(iface, ip) }

func arpProbe(mac net.HardwareAddr, ip net.IP) []byte {
	ip = ip.To4()
	if ip == nil || len(mac) != 6 {
		return nil
	}

	frame := make([]byte, ethHeader+arpPayload)

	copy(frame[0:6], broadcast)
	copy(frame[6:12], mac)
	binary.BigEndian.PutUint16(frame[12:14], 0x0806) // ARP

	arp := frame[ethHeader:]
	binary.BigEndian.PutUint16(arp[0:2], 1)      // ethernet
	binary.BigEndian.PutUint16(arp[2:4], 0x0800) // IPv4
	arp[4] = 6                                   // hardware address length
	arp[5] = 4                                   // protocol address length
	binary.BigEndian.PutUint16(arp[6:8], 1)      // request

	copy(arp[8:14], mac) // sender hardware address
	copy(arp[24:28], ip) // target protocol address

	return frame
}

func taken(frame []byte, mine net.HardwareAddr, want net.IP) bool {
	want = want.To4()
	if want == nil {
		return false
	}

	got, ok := readARP(frame)
	if !ok || got.ours(mine) {
		return false
	}

	if got.senderIP.Equal(want) {
		return true
	}
	return got.senderIP.Equal(net.IPv4zero) && got.targetIP.Equal(want)
}
