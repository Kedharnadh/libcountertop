package hci

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func blob(kind byte, body int) []byte {
	out := make([]byte, 4+body)
	binary.LittleEndian.PutUint32(out, uint32(body)<<8|uint32(kind))

	for i := range out[4:] {
		out[4+i] = byte(i)
	}
	return out
}

func TestTheHeaderIsATypeAndThreeBytesOfLength(t *testing.T) {
	got, err := ReadBlob(blob(tlvNVM, 2186))
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != tlvNVM {
		t.Errorf("type %d, want %d", got.Type, tlvNVM)
	}
	if got.Patch != nil {
		t.Error("an NVM came back with a patch header")
	}
}

func TestATruncatedBlobIsRefused(t *testing.T) {
	full := blob(tlvPatch, 4000)

	if _, err := ReadBlob(full[:2000]); err == nil {
		t.Error("half a patch was accepted")
	}
	if _, err := ReadBlob(full[:3]); err == nil {
		t.Error("a header that is not even a header was accepted")
	}
}

func TestSomethingThatIsNeitherIsRefused(t *testing.T) {
	if _, err := ReadBlob(blob(7, 100)); err == nil {
		t.Error("TLV type 7 was accepted")
	}
}

func TestAPatchHeaderReadsItsFields(t *testing.T) {
	const body = 30020

	raw := make([]byte, 4+body)
	binary.LittleEndian.PutUint32(raw, body<<8|tlvPatch)

	in := raw[4:]
	binary.LittleEndian.PutUint32(in[0:], 0x7844)
	binary.LittleEndian.PutUint32(in[4:], 0x782c)
	in[8], in[9] = 0x01, 0x02
	binary.LittleEndian.PutUint16(in[12:], 0x000c)
	binary.LittleEndian.PutUint16(in[14:], 256)
	binary.LittleEndian.PutUint16(in[16:], 506)
	binary.LittleEndian.PutUint32(in[20:], 0x0001a06c)

	got, err := ReadBlob(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Patch == nil {
		t.Fatal("a patch came back without its header")
	}

	want := Patch{
		TotalSize: 0x7844, DataLength: 0x782c,
		Format: 1, Signature: 2,
		ProductID: 0x000c, ROMBuild: 256, Version: 506, Entry: 0x0001a06c,
	}
	if *got.Patch != want {
		t.Errorf("got %+v\nwant %+v", *got.Patch, want)
	}
}

func TestTheSegmentsAreTheWholeFileInOrder(t *testing.T) {
	for _, size := range []int{1, segment - 1, segment, segment + 1, 30020, 2186} {
		b, err := ReadBlob(blob(tlvPatch, size))
		if err != nil {
			b = Blob{Raw: blob(tlvPatch, size)}
		}

		var back []byte
		for _, piece := range b.Segments() {
			if len(piece) == 0 {
				t.Fatalf("%d bytes: an empty segment", size)
			}
			if len(piece) > segment {
				t.Fatalf("%d bytes: a segment of %d, over the %d limit", size, len(piece), segment)
			}
			back = append(back, piece...)
		}

		if !bytes.Equal(back, b.Raw) {
			t.Errorf("%d bytes: the segments do not join back into the file", size)
		}
	}
}

func TestTheSegmentCountFollowsTheSize(t *testing.T) {
	for _, tc := range []struct{ size, want int }{
		{1, 1},
		{segment, 1},
		{segment + 1, 2},
		{2 * segment, 2},
		{30024, 124},
	} {
		b := Blob{Raw: make([]byte, tc.size)}
		if got := len(b.Segments()); got != tc.want {
			t.Errorf("%d bytes is %d segments, want %d", tc.size, got, tc.want)
		}
	}
}

func TestASegmentCommandCarriesItsLength(t *testing.T) {
	piece := make([]byte, segment)
	for i := range piece {
		piece[i] = byte(i)
	}

	got, err := download(piece)
	if err != nil {
		t.Fatal(err)
	}

	if got[0] != typeCommand {
		t.Errorf("framed as %#02x", got[0])
	}
	if op := binary.LittleEndian.Uint16(got[1:]); op != edlOpcode {
		t.Errorf("opcode %#04x", op)
	}
	if n := int(got[3]); n != len(piece)+2 {
		t.Errorf("says %d parameters, want %d", n, len(piece)+2)
	}
	if got[4] != edlTLVReq {
		t.Errorf("request %#02x, want %#02x", got[4], edlTLVReq)
	}
	if int(got[5]) != len(piece) {
		t.Errorf("says a %d byte segment, want %d", got[5], len(piece))
	}
	if !bytes.Equal(got[6:], piece) {
		t.Error("the segment did not survive framing")
	}
}

func TestAnOversizedSegmentIsRefused(t *testing.T) {
	if _, err := download(make([]byte, segment+1)); err == nil {
		t.Error("an oversized segment was framed anyway")
	}
}
