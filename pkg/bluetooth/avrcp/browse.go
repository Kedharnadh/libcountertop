package avrcp

import (
	"encoding/binary"
	"fmt"
)

// Browsing PDUs have no AV/C wrapper or company ID: PDU id, big-endian length, parameters.

const (
	BrowseSetPlayer     = 0x70
	BrowseFolderItems   = 0x71
	BrowseChangePath    = 0x72
	BrowseItemAttrs     = 0x73
	BrowseTotalItems    = 0x75
	BrowseSearch        = 0x80
	BrowseAddNowPlaying = 0x90
	BrowseReject        = 0xa0
)

const (
	ScopePlayers    = 0x00
	ScopeFilesystem = 0x01
	ScopeSearch     = 0x02
	ScopeNowPlaying = 0x03
)

// Status 0x04 is success; 0x00 is invalid command.
const (
	BrowseBadCommand   = 0x00
	BrowseBadParameter = 0x01
	BrowseBadContent   = 0x02
	BrowseInternal     = 0x03
	BrowseOK           = 0x04
	BrowseNoPlayers    = 0x15
	BrowsePlayerMoved  = 0x16
)

const browseHeader = 3

type Browse struct {
	ID     byte
	Params []byte
}

func ParseBrowse(buf []byte) (Browse, error) {
	if len(buf) < browseHeader {
		return Browse{}, ErrShort
	}

	n := int(binary.BigEndian.Uint16(buf[1:]))
	if len(buf) < browseHeader+n {
		return Browse{}, fmt.Errorf("avrcp: a browsing pdu says %d bytes and %d arrived",
			n, len(buf)-browseHeader)
	}

	return Browse{ID: buf[0], Params: buf[browseHeader : browseHeader+n]}, nil
}

func (b Browse) Marshal() []byte {
	out := make([]byte, browseHeader+len(b.Params))
	out[0] = b.ID
	binary.BigEndian.PutUint16(out[1:], uint16(len(b.Params)))
	copy(out[browseHeader:], b.Params)
	return out
}

func (b Browse) Response(label byte) Transport {
	return Transport{Label: label, Type: MessageResponse, Payload: b.Marshal()}
}

func Reject(status byte) Browse {
	return Browse{ID: BrowseReject, Params: []byte{status}}
}

const (
	ItemPlayer  = 0x01
	ItemFolder  = 0x02
	ItemElement = 0x03
)

func (b Browse) Request(label byte) Transport {
	return Transport{Label: label, Type: MessageCommand, Payload: b.Marshal()}
}

func SetBrowsedPlayer(player uint16) Browse {
	params := make([]byte, 2)
	binary.BigEndian.PutUint16(params, player)
	return Browse{ID: BrowseSetPlayer, Params: params}
}

// The range is inclusive at both ends.
func FolderItems(scope byte, start, end uint32) Browse {
	params := make([]byte, 0, 10)
	params = append(params, scope)
	params = binary.BigEndian.AppendUint32(params, start)
	params = binary.BigEndian.AppendUint32(params, end)

	// Zero attributes means every attribute.
	params = append(params, 0)

	return Browse{ID: BrowseFolderItems, Params: params}
}

func ParseBrowsedPlayer(params []byte) error {
	if len(params) < 1 {
		return ErrShort
	}
	if status := params[0]; status != BrowseOK {
		return fmt.Errorf("avrcp: a player was refused: %#02x", status)
	}
	return nil
}

func TotalItems(scope byte) Browse {
	return Browse{ID: BrowseTotalItems, Params: []byte{scope}}
}

func ParseTotalItems(params []byte) (uint32, error) {
	const size = 7 // status, uid counter, the count itself
	if len(params) < size {
		return 0, ErrShort
	}
	if status := params[0]; status != BrowseOK {
		return 0, fmt.Errorf("avrcp: counting a scope was refused: %#02x", status)
	}
	return binary.BigEndian.Uint32(params[3:]), nil
}

func ItemAttributes(scope byte, uid uint64, counter uint16, attrs ...uint32) Browse {
	params := make([]byte, 0, 12+len(attrs)*4)
	params = append(params, scope)
	params = binary.BigEndian.AppendUint64(params, uid)
	params = binary.BigEndian.AppendUint16(params, counter)

	params = append(params, byte(len(attrs)))
	for _, a := range attrs {
		params = binary.BigEndian.AppendUint32(params, a)
	}
	return Browse{ID: BrowseItemAttrs, Params: params}
}

func ParseItemAttributes(params []byte) (Track, error) {
	if len(params) < 2 {
		return Track{}, ErrShort
	}
	if status := params[0]; status != BrowseOK {
		return Track{}, fmt.Errorf("avrcp: asking about an entry was refused: %#02x", status)
	}
	return ParseElementAttributes(params[1:])
}

func Counter(params []byte) (uint16, bool) {
	if len(params) < 3 {
		return 0, false
	}
	return binary.BigEndian.Uint16(params[1:]), true
}

type Item struct {
	Kind byte

	UID uint64

	Player uint16

	Name  string
	Track Track
}

func ParseFolderItems(params []byte) ([]Item, error) {
	const header = 5 // status, uid counter
	if len(params) < header {
		return nil, ErrShort
	}

	if status := params[0]; status != BrowseOK {
		return nil, fmt.Errorf("avrcp: a listing was refused: %#02x", status)
	}

	count := int(binary.BigEndian.Uint16(params[3:]))
	rest := params[header:]

	out := make([]Item, 0, count)
	for range count {
		// Kind and the length of everything after it.
		if len(rest) < 3 {
			break
		}

		n := int(binary.BigEndian.Uint16(rest[1:]))
		if len(rest) < 3+n {
			break
		}

		item, body := Item{Kind: rest[0]}, rest[3:3+n]
		rest = rest[3+n:]

		switch item.Kind {
		case ItemFolder, ItemElement:
			// UID, type, playable for a folder, then charset, length and name.
			at := 11
			if item.Kind == ItemFolder {
				at = 12
			}
			if len(body) < at+2 {
				continue
			}

			item.UID = binary.BigEndian.Uint64(body)

			length := int(binary.BigEndian.Uint16(body[at:]))
			at += 2
			if len(body) < at+length {
				continue
			}

			item.Name = string(body[at : at+length])
			at += length

			// An element carries its attributes after the name.
			if item.Kind == ItemElement && at < len(body) {
				item.Track, _ = ParseElementAttributes(body[at:])
			}

		case ItemPlayer:
			if len(body) < 2 {
				continue
			}
			item.Player = binary.BigEndian.Uint16(body)

			// Major type, subtype, play status, a sixteen byte feature mask, then the name.
			const named = 2 + 1 + 4 + 1 + 16
			if len(body) < named+4 {
				continue
			}

			length := int(binary.BigEndian.Uint16(body[named+2:]))
			if len(body) < named+4+length {
				continue
			}
			item.Name = string(body[named+4 : named+4+length])
		}

		out = append(out, item)
	}
	return out, nil
}

func Browsed(t Transport) ([]Transport, error) {
	if t.Type != MessageCommand {
		return nil, nil
	}

	b, err := ParseBrowse(t.Payload)
	if err != nil {
		return []Transport{Reject(BrowseBadParameter).Response(t.Label)}, err
	}

	status := byte(BrowseBadCommand)
	switch b.ID {
	case BrowseFolderItems, BrowseTotalItems, BrowseChangePath, BrowseItemAttrs:
		status = BrowseBadParameter
	case BrowseSetPlayer:
		status = BrowseNoPlayers
	}

	return []Transport{Reject(status).Response(t.Label)}, nil
}
