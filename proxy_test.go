package metrics

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/devopsfaith/krakend/proxy"
	"github.com/rcrowley/go-metrics"
)

func TestNewProxyMiddleware(t *testing.T) {
	URL, _ := url.Parse("http://example.com/12345")
	request := &proxy.Request{URL: URL}
	response := &proxy.Response{Data: map[string]interface{}{}, IsComplete: true}
	assertion := func(_ context.Context, req *proxy.Request) (*proxy.Response, error) {
		if request != req {
			t.Errorf("unexpected request! want [%s], have [%s]\n", request, req)
		}
		time.Sleep(time.Millisecond)
		return response, nil
	}

	registry := metrics.NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxyMetric := NewProxyMetrics(&registry)

	mw := NewProxyMiddleware("some", "none", proxyMetric)

	for i := 0; i < 100; i++ {
		resp, err := mw(assertion)(ctx, request)
		if err != nil {
			t.Error("unexpected error:", err)
			return
		}
		if resp != response {
			t.Errorf("unexpected response! want [%v], have [%v]\n", response, resp)
			return
		}
	}

	expected := map[string]struct{}{
		"proxy.latency.layer.some.name.none.complete.true.error.false":  {},
		"proxy.requests.layer.some.name.none.complete.true.error.false": {},
	}
	tracked := []string{}
	proxyMetric.register.Each(func(k string, v interface{}) {
		tracked = append(tracked, k)
	})
	if len(tracked) != len(expected) {
		t.Error("unexpected size of the tracked list", tracked)
	}
	for _, k := range tracked {
		if _, ok := expected[k]; !ok {
			t.Error("the key", k, " has not been tracked")
		}
	}
}
