package sdp

import "sort"

const (
	UUIDL2CAP      = 0x0100
	UUIDAVDTP      = 0x0019
	UUIDAVCTP      = 0x0017
	UUIDOBEX       = 0x0008
	UUIDAudioSink  = 0x110b
	UUIDA2DP       = 0x110d
	UUIDBrowseRoot = 0x1002

	UUIDAVRemoteControlTarget     = 0x110c
	UUIDAVRemoteControl           = 0x110e
	UUIDAVRemoteControlController = 0x110f
)

// Below 0x0100 is defined for all services; above it is the profile's own.
const (
	AttrRecordHandle   = 0x0000
	AttrServiceClasses = 0x0001
	AttrProtocols      = 0x0004
	AttrBrowseGroups   = 0x0005
	AttrProfiles       = 0x0009
	AttrMoreProtocols  = 0x000d
	AttrName           = 0x0100
	AttrFeatures       = 0x0311
)

const (
	FeatureHeadphone = 1 << 0
	FeatureSpeaker   = 1 << 1
	FeatureRecorder  = 1 << 2
	FeatureAmplifier = 1 << 3
)

const A2DPVersion = 0x0103

const AVDTPVersion = 0x0103

type Attribute struct {
	ID    uint16
	Value Element
}

type Record []Attribute

func (r Record) Attribute(id uint16) (Element, bool) {
	for _, a := range r {
		if a.ID == id {
			return a.Value, true
		}
	}
	return Element{}, false
}

// The spec requires ascending attribute IDs.
func (r Record) Element() Element {
	sorted := make(Record, len(r))
	copy(sorted, r)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	// Attribute IDs are always two bytes.
	out := make([]Element, 0, len(sorted)*2)
	for _, a := range sorted {
		out = append(out, Uint16(a.ID), a.Value)
	}
	return Sequence(out...)
}

func AudioSink(handle uint32, name string, psm uint16, features uint16) Record {
	return Record{
		{ID: AttrRecordHandle, Value: Uint32(handle)},

		{ID: AttrServiceClasses, Value: Sequence(UUID16(UUIDAudioSink))},

		// Innermost protocol last; L2CAP carries the PSM as a parameter.
		{ID: AttrProtocols, Value: Sequence(
			Sequence(UUID16(UUIDL2CAP), Uint16(psm)),
			Sequence(UUID16(UUIDAVDTP), Uint16(AVDTPVersion)),
		)},

		{ID: AttrBrowseGroups, Value: Sequence(UUID16(UUIDBrowseRoot))},

		{ID: AttrProfiles, Value: Sequence(
			Sequence(UUID16(UUIDA2DP), Uint16(A2DPVersion)),
		)},

		{ID: AttrName, Value: Text(name)},
		{ID: AttrFeatures, Value: Uint16(features)},
	}
}

// The spec matches a pattern against any UUID in the record, at any depth.
func (r Record) UUIDs() []uint32 {
	var out []uint32
	for _, a := range r {
		collectUUIDs(a.Value, &out)
	}
	return out
}

func collectUUIDs(e Element, out *[]uint32) {
	if e.Type == TypeUUID {
		if v, ok := e.Uint(); ok {
			*out = append(*out, v)
		}
		return
	}
	for _, c := range e.Children {
		collectUUIDs(c, out)
	}
}

const AVRCPVersion = 0x0106

const AVCTPVersion = 0x0104

// AV/C categories: 1 player/recorder, 2 monitor/amplifier.
const (
	CategoryPlayer    = 1 << 0
	CategoryAmplifier = 1 << 1
	CategoryTuner     = 1 << 2
	CategoryMenu      = 1 << 3
)

// The browsing PSM goes in AttrMoreProtocols.
const FeatureBrowsing = 1 << 6

// Controller only: on a target 0x0080 is multiple players and 0x0100 is cover art served.
const (
	FeatureImageProperties = 1 << 7
	FeatureImage           = 1 << 8
	FeatureThumbnail       = 1 << 9
)

func RemoteControl(
	handle uint32,
	name string,
	psm, browse uint16,
	class uint16,
	features uint16,
) Record {
	classes := []Element{UUID16(class)}

	if class != UUIDAVRemoteControl {
		classes = append(classes, UUID16(UUIDAVRemoteControl))
	}

	if browse != 0 {
		features |= FeatureBrowsing
	}

	r := Record{
		{ID: AttrRecordHandle, Value: Uint32(handle)},
		{ID: AttrServiceClasses, Value: Sequence(classes...)},

		{ID: AttrProtocols, Value: Sequence(
			Sequence(UUID16(UUIDL2CAP), Uint16(psm)),
			Sequence(UUID16(UUIDAVCTP), Uint16(AVCTPVersion)),
		)},

		{ID: AttrBrowseGroups, Value: Sequence(UUID16(UUIDBrowseRoot))},

		{ID: AttrProfiles, Value: Sequence(
			Sequence(UUID16(UUIDAVRemoteControl), Uint16(AVRCPVersion)),
		)},

		{ID: AttrName, Value: Text(name)},
		{ID: AttrFeatures, Value: Uint16(features)},
	}

	if browse == 0 {
		return r
	}

	// A list of protocol descriptor lists, one level deeper than AttrProtocols.
	return append(r, Attribute{ID: AttrMoreProtocols, Value: Sequence(
		Sequence(
			Sequence(UUID16(UUIDL2CAP), Uint16(browse)),
			Sequence(UUID16(UUIDAVCTP), Uint16(AVCTPVersion)),
		),
	)})
}

func PSM(protocols Element) (uint16, bool) {
	if protocols.Type != TypeSequence {
		return 0, false
	}

	for _, layer := range protocols.Children {
		if layer.Type != TypeSequence || len(layer.Children) < 2 {
			continue
		}

		id, ok := layer.Children[0].Uint()
		if !ok || id != UUIDL2CAP {
			continue
		}
		if psm, ok := layer.Children[1].Uint(); ok {
			return uint16(psm), true
		}
	}
	return 0, false
}
