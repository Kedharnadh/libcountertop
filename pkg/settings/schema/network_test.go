package schema

import (
	"encoding/json"
	"testing"
)

func TestTLSMigrationAndCanonicalPrecedence(t *testing.T) {
	for _, test := range []struct {
		data string
		want bool
	}{
		{`{}`, true},
		{`{"diag":{"insecure_tls":true}}`, false},
		{`{"diag":{"insecure_tls":false}}`, true},
		{`{"network":{"verify":false},"diag":{"insecure_tls":false}}`, false},
	} {
		document := struct {
			Network Network `json:"network"`
		}{Network: DefaultNetwork()}
		if err := json.Unmarshal([]byte(test.data), &document); err != nil {
			t.Fatal(err)
		}
		n := document.Network
		if err := MigrateNetwork([]byte(test.data), &n); err != nil {
			t.Fatal(err)
		}
		if n.Verify != test.want {
			t.Errorf("%s: verify=%v", test.data, n.Verify)
		}
	}
}
