package cameracmd

import (
	"fmt"
	"os"
	"time"
)

const DefaultStill = "/data/local/tmp/still.jpg"

func Still(take func() ([]byte, error), args []string) (string, error) {
	at := DefaultStill
	if len(args) > 0 && args[0] != "" {
		at = args[0]
	}

	began := time.Now()
	pic, err := take()
	if err != nil {
		return "", err
	}
	took := time.Since(began)

	if err := os.WriteFile(at, pic, 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("  %-9s %d bytes in %s\n  %-9s %s\n",
		"picture", len(pic), took.Round(time.Millisecond), "wrote", at), nil
}
