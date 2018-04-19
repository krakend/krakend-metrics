package metrics

import "github.com/rcrowley/go-metrics"

// NewRouterMetrics creates a RouterMetrics using the injected registry
func NewRouterMetrics(parent *metrics.Registry) *RouterMetrics {
	r := metrics.NewPrefixedChildRegistry(*parent, "router.")

	return &RouterMetrics{
		ProxyMetrics{r},
		metrics.NewRegisteredCounter("connected", r),
		metrics.NewRegisteredCounter("disconnected", r),
		metrics.NewRegisteredCounter("connected-total", r),
		metrics.NewRegisteredCounter("disconnected-total", r),
		metrics.NewRegisteredGauge("connected-gauge", r),
		metrics.NewRegisteredGauge("disconnected-gauge", r),
	}
}

// RouterMetrics is the metrics collector for the router package
type RouterMetrics struct {
	ProxyMetrics
	connected         metrics.Counter
	disconnected      metrics.Counter
	connectedTotal    metrics.Counter
	disconnectedTotal metrics.Counter
	connectedGauge    metrics.Gauge
	disconnectedGauge metrics.Gauge
}

// Connection adds one to the internal connected counter
func (rm *RouterMetrics) Connection() {
	rm.connected.Inc(1)
}

// Disconnection adds one to the internal disconnected counter
func (rm *RouterMetrics) Disconnection() {
	rm.disconnected.Inc(1)
}

func (rm *RouterMetrics) Aggregate() {
	con := rm.connected.Count()
	rm.connectedGauge.Update(con)
	rm.connectedTotal.Inc(con)
	rm.connected.Clear()
	discon := rm.disconnected.Count()
	rm.disconnectedGauge.Update(discon)
	rm.disconnectedTotal.Inc(discon)
	rm.disconnected.Clear()
}

func (rm *RouterMetrics) RegisterResponseWriterMetrics(name string) {
	rm.Counter("response", name, "status")

	rm.Histogram("response", name, "size")
	rm.Histogram("response", name, "time")
}
