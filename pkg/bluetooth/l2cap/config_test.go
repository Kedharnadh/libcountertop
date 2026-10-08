package l2cap

import (
	"bytes"
	"testing"
)

func TestAnMTUOptionGoesOutAndComesBack(t *testing.T) {
	got, err := ParseOptions(MTUOption(895).Marshal())
	if err != nil {
		t.Fatalf("ParseOptions: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("%d options came back, want one", len(got))
	}

	mtu, ok := got[0].MTU()
	if !ok {
		t.Fatal("the option did not read as an mtu")
	}
	if mtu != 895 {
		t.Errorf("mtu came back %d, want 895", mtu)
	}
	if got[0].Hint {
		t.Error("an mtu we send came back marked a hint")
	}
}

func TestTheHintBitIsNotPartOfTheType(t *testing.T) {
	raw := []byte{OptionFlush | OptionHint, 0x02, 0xff, 0xff}

	got, err := ParseOptions(raw)
	if err != nil {
		t.Fatalf("ParseOptions: %v", err)
	}
	if got[0].Type != OptionFlush {
		t.Errorf("type came back %#x, want the flush timeout with the hint bit off", got[0].Type)
	}
	if !got[0].Hint {
		t.Error("a hinted option did not come back as one")
	}
	if !bytes.Equal(got[0].Marshal(), raw) {
		t.Errorf("writing it back gave %x, want %x", got[0].Marshal(), raw)
	}
}

func TestOptionsThatDoNotAddUp(t *testing.T) {
	for _, tc := range []struct {
		name string
		buf  []byte
	}{
		{"half a header", []byte{OptionMTU}},
		{"says more than it carries", []byte{OptionMTU, 0x04, 0x00, 0x02}},
	} {
		if _, err := ParseOptions(tc.buf); err == nil {
			t.Errorf("%s: was accepted", tc.name)
		}
	}
}

func TestReadingAConfigurationRequest(t *testing.T) {
	want := Configure{
		DestinationCID: 0x0040,
		Options:        []Option{MTUOption(672)},
	}

	got, err := ParseConfigure(want.Command(5))
	if err != nil {
		t.Fatalf("ParseConfigure: %v", err)
	}
	if got.DestinationCID != want.DestinationCID {
		t.Errorf("channel came back %#x", got.DestinationCID)
	}
	if got.MTU() != 672 {
		t.Errorf("mtu came back %d, want 672", got.MTU())
	}
	if got.Continues() {
		t.Error("a single request said more was coming")
	}
}

func TestAConfigurationWithNoMTUIsTheDefault(t *testing.T) {
	c := Configure{DestinationCID: 0x0040}

	if c.MTU() != MTUDefault {
		t.Errorf("mtu with no option came back %d, want the default %d", c.MTU(), MTUDefault)
	}
}

func TestAContinuedConfigurationSaysSo(t *testing.T) {
	want := Configure{
		DestinationCID: 0x0040,
		Flags:          ConfigFlagContinues,
		Options:        []Option{MTUOption(672)},
	}

	got, err := ParseConfigure(want.Command(5))
	if err != nil {
		t.Fatalf("ParseConfigure: %v", err)
	}
	if !got.Continues() {
		t.Error("the continuation flag was lost")
	}
}

func TestUnknownOptionsAreReportedAndHintsAreNot(t *testing.T) {
	c := Configure{
		Options: []Option{
			MTUOption(672),
			{Type: 0x7f, Value: []byte{1}},
			{Type: 0x7e, Hint: true, Value: []byte{1}},
			{Type: OptionQoS, Value: make([]byte, 22)},
		},
	}

	got := c.Unknown()
	if len(got) != 1 || got[0] != 0x7f {
		t.Errorf("unknown options came back %x, want just 0x7f", got)
	}
}

func TestAConfigurationResponseCarriesItsResult(t *testing.T) {
	want := Configured{
		SourceCID: 0x0041,
		Result:    ConfigUnacceptable,
		Options:   []Option{MTUOption(672)},
	}

	got, err := ParseConfigured(want.Command(5))
	if err != nil {
		t.Fatalf("ParseConfigured: %v", err)
	}
	if got.SourceCID != want.SourceCID || got.Result != want.Result {
		t.Errorf("came back %+v, want %+v", got, want)
	}
	if len(got.Options) != 1 {
		t.Fatalf("%d options came back, want one", len(got.Options))
	}
	if mtu, _ := got.Options[0].MTU(); mtu != 672 {
		t.Errorf("the counter-proposed mtu came back %d", mtu)
	}
	if c := want.Command(5); c.Code != CodeConfigResponse {
		t.Errorf("the command came out as %#x", c.Code)
	}
}

func TestAWholeConfigurationRound(t *testing.T) {
	asked := Configure{DestinationCID: 0x0040, Options: []Option{MTUOption(895)}}.Command(1)

	req, err := ParseConfigure(asked)
	if err != nil {
		t.Fatalf("ParseConfigure: %v", err)
	}
	if unknown := req.Unknown(); len(unknown) != 0 {
		t.Fatalf("an mtu came back as unknown: %x", unknown)
	}

	answer := Configured{SourceCID: 0x0041, Result: ConfigSuccess}.Command(asked.ID)
	if answer.ID != asked.ID {
		t.Error("the answer did not carry the question's identifier")
	}

	got, err := ParseConfigured(answer)
	if err != nil {
		t.Fatalf("ParseConfigured: %v", err)
	}
	if got.Result != ConfigSuccess {
		t.Errorf("result came back %#x", got.Result)
	}
	if req.MTU() != 895 {
		t.Errorf("the agreed mtu is %d, want 895", req.MTU())
	}
}
