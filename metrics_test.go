package metrics

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/devopsfaith/krakend/logging"
	"github.com/rcrowley/go-metrics"
)

func TestMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := logging.NewLogger("DEBUG", buf, "")
	m := New(ctx, 100*time.Millisecond, l)
	stats1 := m.Snapshot()
	time.Sleep(100 * time.Millisecond)
	stats2 := m.Snapshot()

	if stats1.Time > stats2.Time {
		t.Error("the later stat must have a higher timestamp")
		return
	}
	// sleep some time so the producer is able to collect some logs
	time.Sleep(200 * time.Millisecond)
	lines := len(strings.Split(buf.String(), "\n"))
	if lines < 50 {
		t.Error("unexpected log size. got:", lines)
	}
}

func TestMetrics_process(t *testing.T) {
	p := metrics.NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := false
	m := Metrics{Registry: &p, Router: NewRouterMetrics(&p), latestSnapshot: NewStats()}
	m.processMetrics(ctx, time.Millisecond, customLogger{&called})
	time.Sleep(50 * time.Millisecond)
	totalMetrics := 0
	tm := &totalMetrics
	expected := map[string]bool{
		"router.disconnected-total": true,
		"router.disconnected":       true,
		"router.disconnected-gauge": true,
		"router.connected-total":    true,
		"router.connected":          true,
		"router.connected-gauge":    true,
	}
	p.Each(func(name string, _ interface{}) {
		_, isRouterMetric := expected[name]
		if !strings.HasPrefix(name, "service.") && !isRouterMetric {
			t.Errorf("Unexpected metric: %s", name)
		}
		*tm = *tm + 1
	})
	if totalMetrics < 31 {
		t.Error("Not enough metrics")
	}
}

type customLogger struct {
	called *bool
}

func (l customLogger) Printf(format string, v ...interface{}) {
	*(l.called) = true
}
