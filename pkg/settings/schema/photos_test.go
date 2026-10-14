package schema

import "testing"

func TestPhotosTakeOverThePostersServer(t *testing.T) {
	for _, c := range []struct {
		name string
		data string
		want Photos
	}{
		{
			"old poster settings",
			`{"poster":{"server":"http://immich","key":"k","albums":"Kitchen","tags":"cats"}}`,
			Photos{Server: "http://immich", Key: "k", Albums: "Kitchen", Tags: "cats"},
		},
		{
			"photos already set",
			`{"photos":{"server":"http://new"},"poster":{"server":"http://old"}}`,
			Photos{},
		},
		{"nothing to move", `{"poster":{"enabled":true}}`, Photos{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var got Photos
			if err := MigratePhotos([]byte(c.data), &got); err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}
