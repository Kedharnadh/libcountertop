package screencmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ygelfand/libcountertop/pkg/homeassistant/nws"
)

const warningSpan = 0.15

func Warning(alerts func() *nws.NWS) *cobra.Command {
	return &cobra.Command{
		Use:   "warning [EVENT] | warning clear",
		Short: "Put a sample severe weather alert around home on the panel (a Tornado Warning unless EVENT names another), or take it away",
		RunE: func(cmd *cobra.Command, args []string) error {
			n := alerts()
			event := strings.Join(args, " ")
			switch event {
			case "clear":
				return n.Load(strings.NewReader(`{"features":[]}`))
			case "":
				event = "Tornado Warning"
			}
			lat, lon, err := n.Home(cmd.Context())
			if err != nil {
				return err
			}
			doc, err := sampleWarning(event, lat, lon, time.Now())
			if err != nil {
				return err
			}
			return n.Load(bytes.NewReader(doc))
		},
	}
}

func sampleWarning(event string, lat, lon float64, now time.Time) ([]byte, error) {
	ring := [][2]float64{
		{lon - warningSpan, lat - warningSpan},
		{lon + warningSpan, lat - warningSpan},
		{lon + warningSpan, lat + warningSpan},
		{lon - warningSpan, lat + warningSpan},
		{lon - warningSpan, lat - warningSpan},
	}
	ends := now.Add(30 * time.Minute)
	return json.Marshal(map[string]any{
		"features": []any{map[string]any{
			"geometry": map[string]any{"type": "Polygon", "coordinates": [][][2]float64{ring}},
			"properties": map[string]any{
				"id":    fmt.Sprintf("urn:oid:ctl.warning.%d", now.Unix()),
				"event": event,
				"headline": event + " issued " + now.Format("January 2 at 3:04PM MST") +
					" until " + ends.Format("3:04PM MST") + " by NWS",
				"areaDesc": "Home",
				"description": "The National Weather Service has issued a " + event + " until " + ends.Format(
					"3:04 PM MST",
				) + ".",
				"instruction": "",
				"severity":    "Extreme",
				"status":      "Actual",
				"expires":     ends.Format(time.RFC3339),
				"ends":        ends.Format(time.RFC3339),
			},
		}},
	})
}
