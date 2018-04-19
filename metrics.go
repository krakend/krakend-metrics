// Package metrics defines a set of basic building blocks for instrumenting KakenD gateways
//
// Check the "github.com/devopsfaith/krakend-metrics/gin" and "github.com/devopsfaith/krakend-metrics/mux"
// packages for complete implementations
package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/devopsfaith/krakend/logging"
	"github.com/rcrowley/go-metrics"
)

// New creates a new metrics producer
func New(ctx context.Context, d time.Duration, l logging.Logger) *Metrics {
	registry := metrics.NewPrefixedRegistry("krakend.")

	m := Metrics{
		Router:         NewRouterMetrics(&registry),
		Proxy:          NewProxyMetrics(&registry),
		Registry:       &registry,
		latestSnapshot: NewStats(),
	}

	m.processMetrics(ctx, d, logger{l})

	return &m
}

// Metrics is the component that manages all the metrics
type Metrics struct {
	// Proxy is the metrics collector for the proxy package
	Proxy *ProxyMetrics
	// Router is the metrics collector for the router package
	Router *RouterMetrics
	// Registry is the metrics register
	Registry       *metrics.Registry
	latestSnapshot Stats
}

// Snapshot returns the last calculted snapshot
func (m *Metrics) Snapshot() Stats {
	return m.latestSnapshot
}

// TakeSnapshot takes a snapshot of the current state
func (m *Metrics) TakeSnapshot() Stats {
	tmp := NewStats()

	(*m.Registry).Each(func(k string, v interface{}) {
		switch metric := v.(type) {
		case metrics.Counter:
			tmp.Counters[k] = metric.Count()
		case metrics.Gauge:
			tmp.Gauges[k] = metric.Value()
		case metrics.Histogram:
			tmp.Histograms[k] = HistogramData{
				Max:         metric.Max(),
				Min:         metric.Min(),
				Mean:        metric.Mean(),
				Stddev:      metric.StdDev(),
				Variance:    metric.Variance(),
				Percentiles: metric.Percentiles(percentiles),
			}
		}
	})
	return tmp
}

func (m *Metrics) processMetrics(ctx context.Context, d time.Duration, l metrics.Logger) {
	r := metrics.NewPrefixedChildRegistry(*(m.Registry), "service.")

	metrics.RegisterDebugGCStats(r)
	metrics.RegisterRuntimeMemStats(r)

	go metrics.Log(r, d, l)

	go func() {
		ticker := time.NewTicker(d)
		for {
			select {
			case <-ticker.C:
				metrics.CaptureDebugGCStatsOnce(r)
				metrics.CaptureRuntimeMemStatsOnce(r)
				m.Router.Aggregate()
				m.latestSnapshot = m.TakeSnapshot()
			case <-ctx.Done():
				return
			}
		}
	}()
}

var (
	percentiles   = []float64{0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 0.99}
	defaultSample = func() metrics.Sample { return metrics.NewUniformSample(1028) }
)

type logger struct {
	logger logging.Logger
}

func (l logger) Printf(format string, v ...interface{}) {
	l.logger.Debug(strings.TrimRight(fmt.Sprintf(format, v...), "\n"))
}
