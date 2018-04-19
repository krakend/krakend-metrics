package metrics

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/rcrowley/go-metrics"
)

// NewProxyMiddleware creates a proxy middleware ready to be injected in the pipe as instrumentation point
func (m *Metrics) NewProxyMiddleware(layer, name string) proxy.Middleware {
	return NewProxyMiddleware(layer, name, m.Proxy)
}

// ProxyFactory creates an instrumented proxy factory
func (m *Metrics) ProxyFactory(segmentName string, next proxy.Factory) proxy.FactoryFunc {
	return proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		next, err := next.New(cfg)
		if err != nil {
			return proxy.NoopProxy, err
		}
		return m.NewProxyMiddleware(segmentName, cfg.Endpoint)(next), nil
	})
}

// BackendFactory creates an instrumented backend factory
func (m *Metrics) BackendFactory(segmentName string, next proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		return m.NewProxyMiddleware(segmentName, cfg.URLPattern)(next(cfg))
	}
}

// DefaultBackendFactory creates an instrumented default HTTP backend factory
func (m *Metrics) DefaultBackendFactory() proxy.BackendFactory {
	return m.BackendFactory("backend", proxy.CustomHTTPProxyFactory(proxy.NewHTTPClient))
}

// NewProxyMetrics creates a ProxyMetrics using the injected registry
func NewProxyMetrics(parent *metrics.Registry) *ProxyMetrics {
	m := metrics.NewPrefixedChildRegistry(*parent, "proxy.")
	return &ProxyMetrics{m}
}

// NewProxyMiddleware creates a proxy middleware ready to be injected in the pipe as instrumentation point
func NewProxyMiddleware(layer, name string, pm *ProxyMetrics) proxy.Middleware {
	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
			begin := time.Now()
			resp, err := next[0](ctx, request)

			go func(duration int64, resp *proxy.Response, err error) {
				lvs := []string{
					"requests",
					"layer", layer,
					"name", name,
					"complete", strconv.FormatBool(resp != nil && resp.IsComplete),
					"error", strconv.FormatBool(err != nil),
				}
				pm.Counter(lvs...).Inc(1)
				lvs[0] = "latency"
				pm.Histogram(lvs...).Update(duration)
			}(time.Since(begin).Nanoseconds(), resp, err)

			return resp, err
		}
	}
}

// ProxyMetrics is the metrics collector for the proxy package
type ProxyMetrics struct {
	register metrics.Registry
}

// Histogram gets or register a histogram
func (rm *ProxyMetrics) Histogram(labels ...string) metrics.Histogram {
	return metrics.GetOrRegisterHistogram(strings.Join(labels, "."), rm.register, defaultSample)
}

// Counter gets or register a counter
func (rm *ProxyMetrics) Counter(labels ...string) metrics.Counter {
	return metrics.GetOrRegisterCounter(strings.Join(labels, "."), rm.register)
}
