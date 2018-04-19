// Package gin defines a set of basic building blocks for instrumenting KakenD gateways built using
// the gin router
package gin

import (
	"context"
	"strconv"
	"time"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	krakendgin "github.com/devopsfaith/krakend/router/gin"
	"github.com/gin-gonic/gin"

	"github.com/devopsfaith/krakend-metrics"
	"github.com/devopsfaith/krakend-metrics/mux"
)

// New creates a new metrics producer with support for the gin router
func New(ctx context.Context, d time.Duration, l logging.Logger) *Metrics {
	m := metrics.New(ctx, d, l)
	return &Metrics{m}
}

// Metrics is the component that manages all the metrics for the gin-based gateways
type Metrics struct {
	*metrics.Metrics
}

// NewExpHandler creates an http.Handler ready to expose all the collected metrics as a JSON
func (m *Metrics) NewExpHandler() gin.HandlerFunc {
	return gin.WrapH(mux.NewExpHandler(m.Registry))
}

// NewHTTPHandlerFactory wraps a handler factory adding some simple instrumentation to the generated handlers
func (m *Metrics) NewHTTPHandlerFactory(hf krakendgin.HandlerFactory) krakendgin.HandlerFactory {
	return NewHTTPHandlerFactory(m.Router, hf)
}

// NewHTTPHandlerFactory wraps a handler factory adding some simple instrumentation to the generated handlers
func NewHTTPHandlerFactory(rm *metrics.RouterMetrics, hf krakendgin.HandlerFactory) krakendgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		next := hf(cfg, p)
		rm.RegisterResponseWriterMetrics(cfg.Endpoint)
		return func(c *gin.Context) {
			rw := &ginResponseWriter{c.Writer, cfg.Endpoint, time.Now(), rm}
			c.Writer = rw
			rm.Connection()

			next(c)

			rw.end()
			rm.Disconnection()
		}
	}
}

type ginResponseWriter struct {
	gin.ResponseWriter
	name  string
	begin time.Time
	rm    *metrics.RouterMetrics
}

func (w *ginResponseWriter) end() {
	duration := time.Since(w.begin)
	w.rm.Counter("response", w.name, "status", strconv.Itoa(w.Status()), "count").Inc(1)
	w.rm.Histogram("response", w.name, "size").Update(int64(w.Size()))
	w.rm.Histogram("response", w.name, "time").Update(int64(duration))
}
