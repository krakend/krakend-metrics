package mux

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	metrics "github.com/krakend/krakend-metrics/v2"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/encoding"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	"github.com/luraproject/lura/v2/transport/http/client"
)

var defaultCfg = map[string]interface{}{metrics.Namespace: map[string]interface{}{"collection_time": "100ms"}}

func Example_router() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := logging.NewLogger("DEBUG", buf, "")
	metricProducer := New(ctx, defaultCfg, l)

	h := metricProducer.NewHTTPHandler("test", http.HandlerFunc(dummyHTTPHandler))
	ts1 := httptest.NewServer(h)
	defer ts1.Close()

	for i := 0; i < 10; i++ {
		resp, err := http.Get(ts1.URL)
		if err != nil {
			fmt.Println(err)
		}
		if resp.Header.Get("x-test") != "ok" {
			fmt.Println("unexpected header:", resp.Header.Get("x-test"))
		}
		if resp.StatusCode != 200 {
			fmt.Println("unexpected status code:", resp.StatusCode)
		}
	}
	metricProducer.Router.Aggregate()

	ts2 := httptest.NewServer(metricProducer.NewExpHandler())
	defer ts2.Close()

	resp, err := http.Get(ts2.URL)
	if err != nil {
		fmt.Println(err)
	}
	data, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if len(data) < 5000 {
		fmt.Println("unexpected response size:", len(data))
	}

	// Uncomment to see what's been logged
	// fmt.Println(buf.String())

	// Uncomment the next line and remove the # to see the stats report as the test fails
	// fmt.Println(string(data))
	// #Output: golly

	// Output:
}

func Example_proxy() {
	ctx := context.Background()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := logging.NewLogger("DEBUG", buf, "")
	cfg := &config.EndpointConfig{
		Endpoint: "/test/endpoint",
	}

	metricProducer := New(ctx, defaultCfg, l)

	response := proxy.Response{Data: map[string]interface{}{}, IsComplete: true}
	fakeFactory := proxy.FactoryFunc(func(_ *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) { return &response, nil }, nil
	})
	pf := metricProducer.ProxyFactory("proxy_layer", fakeFactory)

	p, err := pf.New(cfg)
	if err != nil {
		fmt.Println("Error:", err)
	}
	req := proxy.Request{}
	for i := 0; i < 10; i++ {
		resp, err := p(ctx, &req)
		if err != nil {
			fmt.Println("Error:", err)
		}
		if resp != &response {
			fmt.Println("Unexpected response:", *resp)
		}
	}

	ts := httptest.NewServer(metricProducer.NewExpHandler())
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		fmt.Println("Error:", err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
	}
	resp.Body.Close()
	if len(data) < 5000 {
		fmt.Println("unexpected response size:", len(data))
	}

	// Uncomment to see what's been logged
	// fmt.Println(buf.String())

	// Uncomment the next line and remove the # to see the stats report as the test fails
	// fmt.Println(string(data))
	// #Output: golly

	// Output:
}

func Example_backend() {
	ctx := context.Background()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := logging.NewLogger("DEBUG", buf, "")

	metricProducer := New(ctx, defaultCfg, l)

	bf := metricProducer.BackendFactory("backend_layer", proxy.CustomHTTPProxyFactory(client.NewHTTPClient))

	p := bf(&config.Backend{URLPattern: "/some/{url}", Decoder: encoding.JSONDecoder})

	h := metricProducer.NewHTTPHandler("test", http.HandlerFunc(dummyHTTPHandler))
	ts1 := httptest.NewServer(h)
	defer ts1.Close()

	parsedURL, _ := url.Parse(ts1.URL)
	req := proxy.Request{URL: parsedURL, Method: "GET", Body: ioutil.NopCloser(strings.NewReader(""))}
	for i := 0; i < 10; i++ {
		resp, err := p(ctx, &req)
		if err != nil {
			fmt.Println("Error:", err)
		}
		if !resp.IsComplete {
			fmt.Println("Unexpected response:", *resp)
		}
	}

	ts2 := httptest.NewServer(metricProducer.NewExpHandler())
	defer ts2.Close()

	resp, err := http.Get(ts2.URL)
	if err != nil {
		fmt.Println("Error:", err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
	}
	resp.Body.Close()
	if len(data) < 5000 {
		fmt.Println("unexpected response size:", len(data))
	}
	// Uncomment to see what's been logged
	// fmt.Println(buf.String())

	// Uncomment the next line and remove the # to see the stats report as the test fails
	// fmt.Println(string(data))
	// #Output: golly

	// Output:
}

func dummyHTTPHandler(w http.ResponseWriter, _ *http.Request) {
	time.Sleep(time.Millisecond)
	w.Header().Set("x-test", "ok")
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(`{"status":true}`))
}
