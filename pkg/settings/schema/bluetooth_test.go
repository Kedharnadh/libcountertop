package schema

import "testing"

func TestAnUnknownScanShareScansAtTheMost(t *testing.T) {
	for _, s := range []ScanShare{"", "50", "nonsense"} {
		if s.Percent() != DefaultScanShare.Percent() || s.Label() != DefaultScanShare.Label() {
			t.Errorf("%q reads as %d%% %q, want the default", s, s.Percent(), s.Label())
		}
	}
	if ScanShares()[0] != DefaultScanShare {
		t.Errorf("the default %s is not the most the scan takes", DefaultScanShare)
	}
}
