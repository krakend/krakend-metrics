// Package gin defines a set of basic building blocks for instrumenting KrakenD gateways built using
// the gin router
package gin

import (
	"context"
	"net/http"
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
func New(ctx context.Context, e config.ExtraConfig, l logging.Logger) *Metrics {
	metricsCollector := Metrics{metrics.New(ctx, e, l)}
	if metricsCollector.Config != nil && !metricsCollector.Config.EndpointDisabled {
		metricsCollector.RunEndpoint(ctx, metricsCollector.NewEngine(), l)
	}
	return &metricsCollector
}

// Metrics is the component that manages all the metrics for the gin-based gateways
type Metrics struct {
	*metrics.Metrics
}

// RunEndpoint runs the *gin.Engine (that should have the stats endpoint) with the logger
func (m *Metrics) RunEndpoint(ctx context.Context, e *gin.Engine, l logging.Logger) {
	server := &http.Server{
		Addr:    m.Config.ListenAddr,
		Handler: e,
	}
	go func() {
		l.Error(server.ListenAndServe())
	}()

	go func() {
		<-ctx.Done()
		l.Info("shutting down the stats handler")
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		server.Shutdown(ctx)
		cancel()
	}()
}

// NewEngine returns a *gin.Engine with some defaults and the stats endpoint (no logger)
func (m *Metrics) NewEngine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.RedirectTrailingSlash = true
	engine.RedirectFixedPath = true
	engine.HandleMethodNotAllowed = true

	engine.GET("/__stats", m.NewExpHandler())
	return engine
}

// NewExpHandler creates an http.Handler ready to expose all the collected metrics as a JSON
func (m *Metrics) NewExpHandler() gin.HandlerFunc {
	return gin.WrapH(mux.NewExpHandler(m.Registry))
}

// NewHTTPHandlerFactory wraps a handler factory adding some simple instrumentation to the generated handlers
func (m *Metrics) NewHTTPHandlerFactory(hf krakendgin.HandlerFactory) krakendgin.HandlerFactory {
	if m.Config == nil || m.Config.RouterDisabled {
		return hf
	}
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
