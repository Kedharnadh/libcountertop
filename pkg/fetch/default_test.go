package fetch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrdinaryClientsInheritTheVerificationSwitch(t *testing.T) {
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original; Verify(true) }()
	UseAsDefault()
	server := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) },
		),
	)
	defer server.Close()
	ordinary := &http.Client{}
	Verify(true)
	if response, err := ordinary.Get(server.URL); err == nil {
		response.Body.Close()
		t.Fatal("untrusted certificate accepted")
	}
	Verify(false)
	response, err := http.DefaultClient.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	Verify(true)
	if response, err := ordinary.Get(server.URL); err == nil {
		response.Body.Close()
		t.Fatal("strict mode reused insecure policy")
	}
}
