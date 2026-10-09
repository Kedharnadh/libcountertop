package tessera

import (
	"encoding/json"
	"math"
	"testing"
)

func TestSamplesKeepGapsAndStopAtADay(t *testing.T) {
	values := make([]any, 30)
	for i := range values {
		values[i] = i
	}
	values[1] = nil
	raw, _ := json.Marshal(map[string]any{"hours": 24, "values": values})
	got := Tile{History: raw}.Samples()
	if len(got) != samplesMost || got[0] != 0 || !math.IsNaN(got[1]) || got[23] != 23 {
		t.Errorf("samples %v", got)
	}
	if s := (Tile{}).Samples(); s != nil {
		t.Errorf("no history gave %v", s)
	}
}
