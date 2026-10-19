package theme

import "testing"

func TestCustomOverridesOnlyWhatIsSet(t *testing.T) {
	base := Default()
	got := Custom{Accent: "#ff0000", Text: "nonsense"}.Theme()
	if got.Name != CustomName {
		t.Errorf("name %q, want %q", got.Name, CustomName)
	}
	if got.Accent != rgb(0xff0000) {
		t.Errorf("accent %v, want #ff0000", got.Accent)
	}
	if got.Text != base.Text {
		t.Errorf(
			"text %v, want the default's %v when the value is not a color",
			got.Text,
			base.Text,
		)
	}
	if got.Background != base.Background {
		t.Errorf("background %v, want the default's %v", got.Background, base.Background)
	}
}

func TestCustomDarknessFollowsItsBackground(t *testing.T) {
	if (Custom{Background: "#ffffff"}).Theme().Dark {
		t.Error("a white background should make a light theme")
	}
	if !(Custom{Background: "#101010"}).Theme().Dark {
		t.Error("a near-black background should make a dark theme")
	}
}

func TestResolve(t *testing.T) {
	c := Custom{Accent: "#123456"}
	got, ok := Resolve(CustomName, c)
	if !ok || got.Accent != rgb(0x123456) {
		t.Errorf("Resolve(Custom) = %+v, %v", got, ok)
	}
	if got, ok := Resolve("Ocean", c); !ok || got.Name != "Ocean" {
		t.Errorf("Resolve(Ocean) = %q, %v", got.Name, ok)
	}
	if _, ok := Resolve("nope", c); ok {
		t.Error("Resolve(nope) should fail")
	}
}

func TestParseColor(t *testing.T) {
	for _, s := range []string{"#a1b2c3", "a1b2c3", " #A1B2C3 "} {
		if c, ok := ParseColor(s); !ok || c != rgb(0xa1b2c3) {
			t.Errorf("ParseColor(%q) = %v, %v", s, c, ok)
		}
	}
	for _, s := range []string{"", "#abc", "#gggggg", "#a1b2c3d4"} {
		if _, ok := ParseColor(s); ok {
			t.Errorf("ParseColor(%q) should fail", s)
		}
	}
	if CheckColor("") != nil || CheckColor("#000000") != nil || CheckColor("red") == nil {
		t.Error("CheckColor accepts empty and #rrggbb only")
	}
}
