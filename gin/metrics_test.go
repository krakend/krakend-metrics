package gin

import (
	"bytes"
	"context"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	krakendgin "github.com/devopsfaith/krakend/router/gin"
	"github.com/gin-gonic/gin"
)

func TestNew(t *testing.T) {
	rand.Seed(time.Now().Unix())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := logging.NewLogger("DEBUG", buf, "")
	metric := New(ctx, 100*time.Millisecond, l)

	engine := gin.New()

	response := proxy.Response{Data: map[string]interface{}{}, IsComplete: true}
	max := 1000
	min := 1
	p := func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		time.Sleep(time.Microsecond * time.Duration(rand.Intn(max-min)+min))
		return &response, nil
	}
	hf := metric.NewHTTPHandlerFactory(krakendgin.EndpointHandler)
	cfg := &config.EndpointConfig{
		Endpoint: "/test/{var}",
		Timeout:  10 * time.Second,
		CacheTTL: time.Second,
	}
	engine.GET("/test/:var", hf(cfg, p))
	engine.GET("/__stats", metric.NewExpHandler())

	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/a", nil)
		engine.ServeHTTP(w, req)
	}

	metric.Router.Aggregate()
	snapshot := metric.TakeSnapshot()

	expected := map[string]int64{
		"krakend.router.response./test/{var}.status.200.count": 100,
		"krakend.router.connected":                             0,
		"krakend.router.disconnected":                          0,
		"krakend.router.connected-total":                       100,
		"krakend.router.disconnected-total":                    100,
	}
	for k, v := range snapshot.Counters {
		if exp, ok := expected[k]; !ok || int(exp) != int(v) {
			t.Errorf("unexpected metric: got [%s: %d] want [%s: %d]", k, v, k, exp)
		}
	}

	if _, ok := snapshot.Histograms["krakend.router.response./test/{var}.size"]; !ok {
		t.Error("expected histogram not present")
	}

	expected = map[string]int64{
		"krakend.router.connected-gauge":    100,
		"krakend.router.disconnected-gauge": 100,
	}
	for k, exp := range expected {
		if v, ok := snapshot.Gauges[k]; !ok || int(exp) != int(v) {
			t.Errorf("unexpected metric: got [%s: %d] want [%s: %d]", k, v, k, exp)
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__stats", nil)
	engine.ServeHTTP(w, req)

	if w.Result().StatusCode != 200 {
		t.Errorf("unexpected status code: %d\n", w.Result().StatusCode)
	}
}
