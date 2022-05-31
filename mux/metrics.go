// Package mux defines a set of basic building blocks for instrumenting KrakenD gateways built using
// the mux router
package mux

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	"github.com/luraproject/lura/v2/router/mux"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"

	krakendmetrics "github.com/krakendio/krakend-metrics/v2"
)

// New creates a new metrics producer with support for the mux router
func New(ctx context.Context, e config.ExtraConfig, l logging.Logger) *Metrics {
	metricsCollector := Metrics{krakendmetrics.New(ctx, e, l)}
	if metricsCollector.Config != nil && !metricsCollector.Config.EndpointDisabled {
		metricsCollector.RunEndpoint(ctx, metricsCollector.NewEngine(), l)
	}
	return &metricsCollector
}

// Metrics is the component that manages all the metrics for the mux-based gateways
type Metrics struct {
	*krakendmetrics.Metrics
}

// RunEndpoint runs the *gin.Engine (that should have the stats endpoint) with the logger
func (m *Metrics) RunEndpoint(ctx context.Context, s *http.ServeMux, l logging.Logger) {
	server := &http.Server{
		Addr:    m.Config.ListenAddr,
		Handler: s,
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

// NewEngine returns a *http.ServeMux with the stats endpoint (no logger)
func (m *Metrics) NewEngine() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/__stats", m.NewExpHandler())
	return mux
}

// NewExpHandler creates an http.Handler ready to expose all the collected metrics as a JSON
func (m *Metrics) NewExpHandler() http.Handler {
	return NewExpHandler(m.Registry)
}

// NewHTTPHandler wraps an http.Handler adding some simple instrumentation to the handler
func (m *Metrics) NewHTTPHandler(name string, h http.Handler) http.HandlerFunc {
	return NewHTTPHandler(name, h, m.Router)
}

func (m *Metrics) NewHTTPHandlerFactory(defaultHandlerFactory mux.HandlerFactory) mux.HandlerFactory {
	if m.Config == nil || m.Config.RouterDisabled {
		return defaultHandlerFactory
	}
	return func(cfg *config.EndpointConfig, p proxy.Proxy) http.HandlerFunc {
		return m.NewHTTPHandler(cfg.Endpoint, defaultHandlerFactory(cfg, p))
	}
}

// NewExpHandler creates an http.Handler ready to expose all the collected metrics as a JSON
func NewExpHandler(parent *metrics.Registry) http.Handler {
	return exp.ExpHandler(*parent)
}

// NewHTTPHandler wraps an http.Handler adding some simple instrumentation to the handler
func NewHTTPHandler(name string, h http.Handler, rm *krakendmetrics.RouterMetrics) http.HandlerFunc {
	rm.RegisterResponseWriterMetrics(name)
	return func(w http.ResponseWriter, r *http.Request) {
		rm.Connection(r.TLS)
		rw := newHTTPResponseWriter(name, w, rm)
		h.ServeHTTP(rw, r)
		rw.end()
		rm.Disconnection()
	}
}

func newHTTPResponseWriter(name string, rw http.ResponseWriter, rm *krakendmetrics.RouterMetrics) *responseWriter {
	return &responseWriter{
		ResponseWriter: rw,
		begin:          time.Now(),
		name:           name,
		rm:             rm,
		status:         200,
	}
}

type responseWriter struct {
	http.ResponseWriter
	begin        time.Time
	name         string
	rm           *krakendmetrics.RouterMetrics
	responseSize int
	status       int
}

// WriteHeader implementes the http.ResponseWriter interface
func (w *responseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	w.status = code
}

// Write implementes the http.ResponseWriter interface
func (w *responseWriter) Write(data []byte) (i int, err error) {
	i, err = w.ResponseWriter.Write(data)
	w.responseSize += i
	return
}

func (w responseWriter) end() {
	duration := time.Since(w.begin)
	w.rm.Counter("response", w.name, "status", strconv.Itoa(w.status), "count").Inc(1)
	w.rm.Histogram("response", w.name, "size").Update(int64(w.responseSize))
	w.rm.Histogram("response", w.name, "time").Update(int64(duration))
}
