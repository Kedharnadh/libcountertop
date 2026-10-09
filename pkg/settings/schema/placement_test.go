package schema

import "testing"

func TestOnlyATopRightPlacementIsTopRight(t *testing.T) {
	tests := []struct {
		at   Placement
		want bool
	}{
		{Placement{Position: PositionTop, Align: AlignRight}, true},
		{Placement{Position: PositionTop, Align: AlignLeft}, false},
		{Placement{Position: PositionBottom, Align: AlignRight}, false},
	}
	for _, tc := range tests {
		if got := tc.at.TopRight(); got != tc.want {
			t.Errorf("%+v.TopRight() = %v, want %v", tc.at, got, tc.want)
		}
	}
}
