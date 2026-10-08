package obex

// AVRCP's cover art UUID, not a BIP one.
var CoverArtTarget = []byte{
	0x71, 0x63, 0xdd, 0x54, 0x4a, 0x7e, 0x11, 0xe2,
	0xb4, 0x7c, 0x00, 0x50, 0xc2, 0x49, 0x00, 0x48,
}

// Null terminated; BIP fixes the thumbnail at 200x200 JPEG.
const TypeThumbnail = "x-bt/img-thm"

const HeaderImgHandle = 0x30

func Thumbnail(connection uint32, handle string) Packet {
	return Get(connection,
		Bytes(HeaderType, terminated(TypeThumbnail)),
		Text(HeaderImgHandle, handle),
	)
}

// ASCII with a trailing zero.
func terminated(s string) []byte { return append([]byte(s), 0) }
