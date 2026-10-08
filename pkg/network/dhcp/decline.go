package dhcp

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

const (
	clientPort = 68
	serverPort = 67
)

const (
	ipHeader  = 20
	udpHeader = 8
)

const ttl = 64

// RFC 2131 3.1.5: the declined address goes in requested-address; ciaddr stays zero.
func declineMessage(mac net.HardwareAddr, addr, server net.IP, why string) ([]byte, error) {
	if addr.To4() == nil {
		return nil, fmt.Errorf("dhcp: %s is not an IPv4 address", addr)
	}

	msg, err := dhcpv4.New(
		dhcpv4.WithHwAddr(mac),
		dhcpv4.WithMessageType(dhcpv4.MessageTypeDecline),
		dhcpv4.WithOption(dhcpv4.OptRequestedIPAddress(addr)),
		dhcpv4.WithOption(dhcpv4.OptMessage(why)),
	)
	if err != nil {
		return nil, fmt.Errorf("dhcp: building a decline: %w", err)
	}

	if server.To4() != nil {
		msg.UpdateOption(dhcpv4.OptServerIdentifier(server))
	}
	return msg.ToBytes(), nil
}

func declineFrame(mac net.HardwareAddr, payload []byte) []byte {
	if len(mac) != 6 {
		return nil
	}

	frame := make([]byte, ethHeader+ipHeader+udpHeader+len(payload))

	copy(frame[0:6], broadcast)
	copy(frame[6:12], mac)
	binary.BigEndian.PutUint16(frame[12:14], 0x0800) // IPv4

	ip := frame[ethHeader:]
	udp := ip[ipHeader:]
	copy(udp[udpHeader:], payload)

	from, to := net.IPv4zero.To4(), net.IPv4bcast.To4()

	ip[0] = 0x45 // version 4, five words of header
	binary.BigEndian.PutUint16(ip[2:4], uint16(ipHeader+udpHeader+len(payload)))
	ip[8] = ttl
	ip[9] = 17 // UDP
	copy(ip[12:16], from)
	copy(ip[16:20], to)
	binary.BigEndian.PutUint16(ip[10:12], checksum(ip[:ipHeader]))

	binary.BigEndian.PutUint16(udp[0:2], clientPort)
	binary.BigEndian.PutUint16(udp[2:4], serverPort)
	binary.BigEndian.PutUint16(udp[4:6], uint16(udpHeader+len(payload)))
	binary.BigEndian.PutUint16(udp[6:8], udpChecksum(from, to, udp))

	return frame
}

// RFC 768: a computed checksum of zero is sent as all ones.
func udpChecksum(from, to net.IP, datagram []byte) uint16 {
	pseudo := make([]byte, 12)
	copy(pseudo[0:4], from)
	copy(pseudo[4:8], to)
	pseudo[9] = 17 // UDP
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(len(datagram)))

	if sum := checksum(pseudo, datagram); sum != 0 {
		return sum
	}
	return 0xffff
}

// RFC 1071 internet checksum.
func checksum(parts ...[]byte) uint16 {
	var sum uint32
	var odd bool

	for _, part := range parts {
		for i, b := range part {
			if (i%2 == 0) != odd {
				sum += uint32(b) << 8
			} else {
				sum += uint32(b)
			}
		}
		if len(part)%2 == 1 {
			odd = !odd
		}
	}

	for sum>>16 != 0 {
		sum = sum&0xffff + sum>>16
	}
	return ^uint16(sum)
}
