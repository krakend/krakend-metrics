package metrics

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/transport/http/client"
	"github.com/rcrowley/go-metrics"
)

// NewProxyMiddleware creates a proxy middleware ready to be injected in the pipe as instrumentation point
func (m *Metrics) NewProxyMiddleware(layer, name string) proxy.Middleware {
	return NewProxyMiddleware(layer, name, m.Proxy)
}

// ProxyFactory creates an instrumented proxy factory
func (m *Metrics) ProxyFactory(segmentName string, next proxy.Factory) proxy.FactoryFunc {
	if m.Config == nil || m.Config.ProxyDisabled {
		return next.New
	}
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
	if m.Config == nil || m.Config.BackendDisabled {
		return next
	}
	return func(cfg *config.Backend) proxy.Proxy {
		return m.NewProxyMiddleware(segmentName, cfg.URLPattern)(next(cfg))
	}
}

// DefaultBackendFactory creates an instrumented default HTTP backend factory
func (m *Metrics) DefaultBackendFactory() proxy.BackendFactory {
	return m.BackendFactory("backend", proxy.CustomHTTPProxyFactory(client.NewHTTPClient))
}

// NewProxyMetrics creates a ProxyMetrics using the injected registry
func NewProxyMetrics(parent *metrics.Registry) *ProxyMetrics {
	m := metrics.NewPrefixedChildRegistry(*parent, "proxy.")
	return &ProxyMetrics{m}
}

// NewProxyMiddleware creates a proxy middleware ready to be injected in the pipe as instrumentation point
func NewProxyMiddleware(layer, name string, pm *ProxyMetrics) proxy.Middleware {
	registerProxyMiddlewareMetrics(layer, name, pm)
	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
			begin := time.Now()
			resp, err := next[0](ctx, request)

			go func(duration int64, resp *proxy.Response, err error) {
				errored := strconv.FormatBool(err != nil)
				complete := strconv.FormatBool(resp != nil && resp.IsComplete)
				labels := "layer." + layer + ".name." + name + ".complete." + complete + ".error." + errored
				pm.Counter("requests." + labels).Inc(1)
				pm.Histogram("latency." + labels).Update(duration)
			}(time.Since(begin).Nanoseconds(), resp, err)

			return resp, err
		}
	}
}

func registerProxyMiddlewareMetrics(layer, name string, pm *ProxyMetrics) {
	labels := "layer." + layer + ".name." + name
	for _, complete := range []string{"true", "false"} {
		for _, errored := range []string{"true", "false"} {
			metrics.GetOrRegisterCounter("requests."+labels+".complete."+complete+".error."+errored, pm.register)

			metrics.GetOrRegisterHistogram("latency."+labels+".complete."+complete+".error."+errored, pm.register, defaultSample())
		}
	}
}

// ProxyMetrics is the metrics collector for the proxy package
type ProxyMetrics struct {
	register metrics.Registry
}

// Histogram gets or register a histogram
func (rm *ProxyMetrics) Histogram(labels ...string) metrics.Histogram {
	return metrics.GetOrRegisterHistogram(strings.Join(labels, "."), rm.register, defaultSample())
}

// Counter gets or register a counter
func (rm *ProxyMetrics) Counter(labels ...string) metrics.Counter {
	return metrics.GetOrRegisterCounter(strings.Join(labels, "."), rm.register)
}
