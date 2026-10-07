package native

import "testing"

func TestSharedArtifactsRejectIncompatibleBoardAPIs(t *testing.T) {
	targets := []Target{{Name: "speaker"}, {Name: "older", API: 25}, {Name: "newer", API: 27}}
	if _, err := Select(targets, ""); err == nil {
		t.Fatal("mixed native ABIs silently selected one API")
	}
	for _, target := range targets[1:] {
		api, err := Select(targets, target.Name)
		if err != nil || api != target.API {
			t.Fatalf("%s: API %d, %v", target.Name, api, err)
		}
	}
	for _, name := range []string{"speaker", "missing"} {
		if _, err := Select(targets, name); err == nil {
			t.Fatalf("%s unexpectedly has a native target", name)
		}
	}
}
