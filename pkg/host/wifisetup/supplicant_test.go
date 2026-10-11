package wifisetup

import "testing"

func TestSupplicantClient(t *testing.T) {
	tests := []struct {
		name string
		s    Supplicant
		want string
	}{
		{
			name: "wpa_cli by default",
			s:    Supplicant{Sockets: "/data/misc/wifi/sockets", Iface: "wlan0"},
			want: "wpa_cli -p /data/misc/wifi/sockets -i wlan0",
		},
		{
			name: "named client",
			s: Supplicant{
				Sockets: "/data/misc/wifi/sockets",
				Iface:   "wlan0",
				Client:  "/data/misc/lanovo/bin/lanovod wpa",
			},
			want: "/data/misc/lanovo/bin/lanovod wpa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.client(); got != tt.want {
				t.Errorf("client() = %q, want %q", got, tt.want)
			}
		})
	}
}
