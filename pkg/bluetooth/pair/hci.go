package pair

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	EventConnectionComplete    = 0x03
	EventConnectionRequest     = 0x04
	EventDisconnectionComplete = 0x05
	EventAuthComplete          = 0x06
	EventEncryptionChange      = 0x08
	EventPINCodeRequest        = 0x16
	EventLinkKeyRequest        = 0x17
	EventLinkKeyNotification   = 0x18
	EventRemoteName            = 0x07
	EventIOCapabilityRequest   = 0x31
	EventIOCapabilityResponse  = 0x32
	EventUserConfirmation      = 0x33
	EventUserPasskeyRequest    = 0x34
	EventSimplePairingComplete = 0x36
)

const (
	OpAcceptConnection   = 0x0409
	OpRejectConnection   = 0x040a
	OpLinkKeyReply       = 0x040b
	OpLinkKeyNegative    = 0x040c
	OpPINCodeNegative    = 0x040e
	OpIOCapabilityReply  = 0x042b
	OpConfirmReply       = 0x042c
	OpConfirmNegative    = 0x042d
	OpPasskeyNegative    = 0x042f
	OpIOCapabilityNegate = 0x0434
	OpRemoteName         = 0x0419
)

// NoInputNoOutput gives Just Works; DisplayYesNo gives numeric comparison.
const (
	DisplayOnly     = 0x00
	DisplayYesNo    = 0x01
	KeyboardOnly    = 0x02
	NoInputNoOutput = 0x03
)

// Without bonding the controller generates no link key.
const (
	NoBonding              = 0x00
	NoBondingMITM          = 0x01
	DedicatedBonding       = 0x02
	DedicatedBondingMITM   = 0x03
	GeneralBonding         = 0x04
	GeneralBondingWithMITM = 0x05
)

const (
	LinkSCO  = 0x00
	LinkACL  = 0x01
	LinkESCO = 0x02
)

const (
	RejectLimitedResources = 0x0d
	RejectSecurity         = 0x0e
	RejectUnacceptableAddr = 0x0f
)

const (
	KeyCombination         = 0x00
	KeyLocalUnit           = 0x01
	KeyRemoteUnit          = 0x02
	KeyDebug               = 0x03
	KeyUnauthenticated     = 0x04
	KeyAuthenticated       = 0x05
	KeyChangedCombination  = 0x06
	KeyUnauthenticatedP256 = 0x07
	KeyAuthenticatedP256   = 0x08
)

// Wire order: least significant byte first.
type Addr [6]byte

func ParseAddr(buf []byte) (Addr, bool) {
	var a Addr
	if len(buf) < len(a) {
		return a, false
	}
	copy(a[:], buf)
	return a, true
}

func (a Addr) String() string {
	var b strings.Builder
	for i := len(a) - 1; i >= 0; i-- {
		if b.Len() > 0 {
			b.WriteByte(':')
		}
		fmt.Fprintf(&b, "%02x", a[i])
	}
	return b.String()
}

type Key [16]byte

type Event struct {
	Code   byte
	Params []byte
}

func (e Event) Addr() (Addr, bool) { return ParseAddr(e.Params) }

type Command struct {
	Opcode uint16
	Params []byte
}

func (c Command) String() string {
	return fmt.Sprintf("%#04x %x", c.Opcode, c.Params)
}

func reply(opcode uint16, a Addr) Command {
	return Command{Opcode: opcode, Params: append([]byte(nil), a[:]...)}
}

// The address follows a status and a handle.
func Connected(params []byte) (Addr, bool) {
	const at = 1 + 2
	if len(params) < at+6 || params[0] != 0 {
		return Addr{}, false
	}
	return ParseAddr(params[at:])
}

// Then page scan repetition mode, a reserved byte and a clock offset.
func RemoteName(a Addr) Command {
	params := append([]byte(nil), a[:]...)
	return Command{Opcode: OpRemoteName, Params: append(params, 0x01, 0x00, 0x00, 0x00)}
}

// The name is 248 bytes, zero padded.
func ParseRemoteName(params []byte) (Addr, string, error) {
	const size = 1 + 6 + 248
	if len(params) < 7 {
		return Addr{}, "", fmt.Errorf("pair: a remote name of %d bytes", len(params))
	}

	if status := params[0]; status != 0 {
		return Addr{}, "", fmt.Errorf("pair: the far end would not give its name: %#02x", status)
	}

	a, ok := ParseAddr(params[1:])
	if !ok {
		return Addr{}, "", fmt.Errorf("pair: a remote name with no address")
	}

	name := params[7:]
	if len(params) > size {
		name = params[7:size]
	}
	return a, string(bytes.TrimRight(name, "\x00")), nil
}

// The top four bits are reserved.
func Handle(params []byte) (uint16, bool) {
	if len(params) < 3 {
		return 0, false
	}
	return binary.LittleEndian.Uint16(params[1:]) & 0x0fff, true
}
