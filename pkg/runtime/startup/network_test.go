package startup

import "testing"

func TestNetworkAddressRequiresCurrentAssociation(t *testing.T) {
	tests := []struct {
		name    string
		link    Progress
		address string
		want    Progress
	}{
		{
			"retained lease",
			Progress{Doing: "looking for the network"},
			"10.0.0.2",
			Progress{Doing: "waiting for Wi-Fi"},
		},
		{
			"associated without lease",
			Progress{Done: true},
			"",
			Progress{Doing: "asking for an address"},
		},
		{
			"associated with lease",
			Progress{Done: true},
			"10.0.0.2",
			Progress{Done: true, Doing: "10.0.0.2"},
		},
		{
			"no configured network",
			Progress{Failed: true, Doing: "no network set up"},
			"10.0.0.2",
			Progress{Failed: true, Doing: "no network set up"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NetworkAddress(tt.link, tt.address); got != tt.want {
				t.Errorf(
					"NetworkAddress(%+v, %q) = %+v, want %+v",
					tt.link,
					tt.address,
					got,
					tt.want,
				)
			}
		})
	}
}
