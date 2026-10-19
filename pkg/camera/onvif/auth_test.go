package onvif

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/network/digest"
)

func token(user, password string) string {
	nonce := []byte("0123456789abcdef")
	created := "2026-10-19T12:00:00Z"
	h := sha1.Sum(append(append(append([]byte{}, nonce...), created...), password...))
	return `<wsse:Security xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" ` +
		`xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">` +
		`<wsse:UsernameToken><wsse:Username>` + user + `</wsse:Username>` +
		`<wsse:Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordDigest">` +
		base64.StdEncoding.EncodeToString(
			h[:],
		) + `</wsse:Password>` +
		`<wsse:Nonce>` + base64.StdEncoding.EncodeToString(
		nonce,
	) + `</wsse:Nonce>` +
		`<wsu:Created>` + created + `</wsu:Created></wsse:UsernameToken></wsse:Security>`
}

func md5hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func httpDigest(c digest.Challenge, method, uri, user, password string) string {
	ha1 := md5hex(user + ":" + c.Realm + ":" + password)
	ha2 := md5hex(method + ":" + uri)
	return fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s"`,
		user, c.Realm, c.Nonce, uri, md5hex(ha1+":"+c.Nonce+":"+ha2))
}

func TestCredentialsGuardTheServices(t *testing.T) {
	s := &Service{
		Device: Device{
			Name:     "display",
			RTSPPort: 8554,
			Profiles: func() []Profile { return []Profile{{Token: "main", Name: "Main", Path: "main"}} },
		},
		Auth:      func() (string, string) { return "admin", "pw" },
		challenge: digest.Challenge{Realm: "LANovo", Nonce: "n0nce"},
	}
	mux := http.NewServeMux()
	mux.Handle(MediaPath, s)
	mux.Handle(DevicePath, s)
	mux.HandleFunc(SnapshotPath, snapshot(func() ([]byte, error) { return []byte("jpeg"), nil }, s))
	srv := httptest.NewServer(mux)
	defer srv.Close()
	dev, media := srv.URL+DevicePath, srv.URL+MediaPath

	if code, body := soap(t, dev, "tds:GetSystemDateAndTime", ""); code != 200 {
		t.Errorf("GetSystemDateAndTime needs no credentials, answered %d\n%s", code, body)
	}

	for _, c := range []struct {
		name, header, authorization string
		want                        int
	}{
		{name: "nothing", want: 401},
		{name: "wrong password", header: token("admin", "nope"), want: 401},
		{name: "wrong user", header: token("root", "pw"), want: 401},
		{name: "username token", header: token("admin", "pw"), want: 200},
		{name: "http digest", authorization: httpDigest(s.challenge, "POST", MediaPath, "admin", "pw"), want: 200},
	} {
		code, body := soapWith(t, media, "trt:GetProfiles", "", c.header, c.authorization)
		if code != c.want {
			t.Errorf("%s: GetProfiles answered %d, want %d\n%s", c.name, code, c.want, body)
		}
		if c.want == 401 && !strings.Contains(body, "ter:NotAuthorized") {
			t.Errorf("%s: no NotAuthorized fault\n%s", c.name, body)
		}
	}

	get := func(authorization string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+SnapshotPath, nil)
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		r.Body.Close()
		return r
	}
	if r := get(
		"",
	); r.StatusCode != 401 ||
		!strings.HasPrefix(r.Header.Get("WWW-Authenticate"), "Digest ") {
		t.Errorf(
			"the snapshot without credentials answered %d %q",
			r.StatusCode,
			r.Header.Get("WWW-Authenticate"),
		)
	}
	if r := get(httpDigest(s.challenge, "GET", SnapshotPath, "admin", "pw")); r.StatusCode != 200 {
		t.Errorf("the snapshot with credentials answered %d", r.StatusCode)
	}

	s.Auth = func() (string, string) { return "admin", "" }
	if code, _ := soap(t, media, "trt:GetProfiles", ""); code != 200 {
		t.Errorf("with no password GetProfiles answered %d", code)
	}
	if r := get(""); r.StatusCode != 200 {
		t.Errorf("with no password the snapshot answered %d", r.StatusCode)
	}
}
