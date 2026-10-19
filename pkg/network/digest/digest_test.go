package digest

import "testing"

func creds(user, password string) Credentials {
	return func() (string, string) { return user, password }
}

func TestRFC2617Example(t *testing.T) {
	c := Challenge{Realm: "testrealm@host.com", Nonce: "dcd98b7102dd2f0e8b11d0f600bfb0c093"}
	auth := `Digest username="Mufasa", realm="testrealm@host.com", ` +
		`nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093", uri="/dir/index.html", qop=auth, ` +
		`nc=00000001, cnonce="0a4f113b", response="6629fae49393a05397450978507c4ef1", ` +
		`opaque="5ccc069c403ebaf9f0171e9517f40e41"`
	if !c.Allows(auth, "GET", creds("Mufasa", "Circle Of Life")) {
		t.Fatal("the RFC 2617 example was refused")
	}
	if c.Allows(auth, "GET", creds("Mufasa", "wrong")) {
		t.Fatal("a wrong password was allowed")
	}
	if c.Allows(auth, "POST", creds("Mufasa", "Circle Of Life")) {
		t.Fatal("a response for another method was allowed")
	}
}

func TestWithoutQop(t *testing.T) {
	c := Challenge{Realm: "LANovo", Nonce: "abc"}
	ha1 := sum("admin:LANovo:pw")
	ha2 := sum("DESCRIBE:rtsp://host/main")
	auth := `Digest username="admin", realm="LANovo", nonce="abc", uri="rtsp://host/main", response="` +
		sum(
			ha1+":abc:"+ha2,
		) + `"`
	if !c.Allows(auth, "DESCRIBE", creds("admin", "pw")) {
		t.Fatal("a valid response without qop was refused")
	}
	stale := Challenge{Realm: "LANovo", Nonce: "other"}
	if stale.Allows(auth, "DESCRIBE", creds("admin", "pw")) {
		t.Fatal("a response to another nonce was allowed")
	}
}

func TestNoPasswordIsOpen(t *testing.T) {
	c := New("LANovo")
	if !c.Allows("", "DESCRIBE", creds("admin", "")) || !c.Allows("", "DESCRIBE", nil) {
		t.Fatal("an empty password should leave the server open")
	}
	if c.Allows("", "DESCRIBE", creds("admin", "pw")) {
		t.Fatal("a missing Authorization header was allowed")
	}
	if c.Allows("Basic YWRtaW46cHc=", "DESCRIBE", creds("admin", "pw")) {
		t.Fatal("Basic was allowed")
	}
}
