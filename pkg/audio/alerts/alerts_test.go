package alerts

import (
	"reflect"
	"testing"

	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestTimerTableExcludesMessageControls(t *testing.T) {
	table := TimerTable(
		func() []setting.Option { return []setting.Option{{Value: "tone:bell", Label: "Bell"}} },
	)
	var names []string
	for _, row := range table.Rows() {
		names = append(names, row.Name)
	}
	if !reflect.DeepEqual(names, []string{"timer", "ring_for"}) {
		t.Fatalf("timer controls = %v", names)
	}
	var at schema.Alerts
	if err := table.MustRow("timer").Write(&at, "tone:bell"); err != nil {
		t.Fatal(err)
	}
	if at.Timer != schema.CustomChime("bell") {
		t.Errorf("timer = %q", at.Timer)
	}
	for _, name := range []string{"info", "success", "warning", "alert"} {
		if _, ok := table.Find(name); ok {
			t.Errorf("timer table includes %s", name)
		}
	}
}
