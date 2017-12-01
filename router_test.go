package metrics

import (
	"strings"
	"testing"

	"github.com/rcrowley/go-metrics"
)

func TestRouterMetrics(t *testing.T) {
	p := metrics.NewRegistry()
	rm := NewRouterMetrics(&p)

	rm.Connection()
	rm.Connection()
	rm.Disconnection()
	rm.Connection()
	rm.Connection()
	rm.Connection()

	rm.Aggregate()

	rm.Connection()
	rm.Disconnection()
	rm.Disconnection()
	rm.Disconnection()

	rm.Aggregate()

	rm.Disconnection()

	p.Each(func(name string, v interface{}) {
		if !strings.HasPrefix(name, "router.") {
			t.Errorf("Unexpected metric: %s", name)
		}
	})

	for k, want := range map[string]int64{
		"router.connected":          0,
		"router.disconnected":       1,
		"router.connected-total":    6,
		"router.disconnected-total": 4,
		"router.connected-gauge":    1,
		"router.disconnected-gauge": 3,
	} {
		var have int64
		switch metric := p.Get(k).(type) {
		case metrics.Counter:
			have = metric.Count()
		case metrics.Gauge:
			have = metric.Value()
		}
		if have != want {
			t.Errorf("Unexpected value for %s. Have: %d, want: %d", k, have, want)
		}
	}
}
