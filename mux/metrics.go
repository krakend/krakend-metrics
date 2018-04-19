// Package mux defines a set of basic building blocks for instrumenting KakenD gateways built using
// the mux router
package mux

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/router/mux"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"

	krakendmetrics "github.com/devopsfaith/krakend-metrics"
)

// New creates a new metrics producer with support for the mux router
func New(ctx context.Context, d time.Duration, l logging.Logger) *Metrics {
	return &Metrics{krakendmetrics.New(ctx, d, l)}
}

// Metrics is the component that manages all the metrics for the mux-based gateways
type Metrics struct {
	*krakendmetrics.Metrics
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
		rm.Connection()
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
