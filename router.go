package metrics

import (
	"crypto/tls"

	metrics "github.com/rcrowley/go-metrics"
)

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
func (rm *RouterMetrics) Connection(TLS *tls.ConnectionState) {
	rm.connected.Inc(1)
	if TLS == nil {
		return
	}

	rm.Counter("tls_version", tlsVersion[TLS.Version], "count").Inc(1)
	rm.Counter("tls_cipher", tlsCipherSuite[TLS.CipherSuite], "count").Inc(1)
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

var (
	tlsVersion = map[uint16]string{
		tls.VersionTLS10: "VersionTLS10",
		tls.VersionTLS11: "VersionTLS11",
		tls.VersionTLS12: "VersionTLS12",
		tls.VersionTLS13: "VersionTLS13",
		tls.VersionSSL30: "VersionSSL30",
	}

	tlsCipherSuite = map[uint16]string{
		tls.TLS_RSA_WITH_RC4_128_SHA:                      "TLS_RSA_WITH_RC4_128_SHA",
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:                 "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		tls.TLS_RSA_WITH_AES_128_CBC_SHA:                  "TLS_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_RSA_WITH_AES_256_CBC_SHA:                  "TLS_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256:               "TLS_RSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256:               "TLS_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384:               "TLS_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:              "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:          "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:          "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:                "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:           "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:            "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:            "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256:       "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256:         "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:         "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:       "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:         "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:       "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256:   "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256: "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
		tls.TLS_AES_128_GCM_SHA256:                        "TLS_AES_128_GCM_SHA256",
		tls.TLS_AES_256_GCM_SHA384:                        "TLS_AES_256_GCM_SHA384",
		tls.TLS_CHACHA20_POLY1305_SHA256:                  "TLS_CHACHA20_POLY1305_SHA256",
		tls.TLS_FALLBACK_SCSV:                             "TLS_FALLBACK_SCSV",
	}
)
