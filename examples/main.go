package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	viper "github.com/krakendio/krakend-viper/v2"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	krakendgin "github.com/luraproject/lura/v2/router/gin"
	"github.com/luraproject/lura/v2/router/gorilla"
	"github.com/luraproject/lura/v2/router/mux"

	metricsgin "github.com/krakendio/krakend-metrics/v2/gin"
	metricsmux "github.com/krakendio/krakend-metrics/v2/mux"
)

func main() {
	port := flag.Int("p", 0, "Port of the service")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("d", false, "Enable the debug")
	useGorilla := flag.Bool("gorilla", false, "Use the gorilla router (gin is used by default)")
	configFile := flag.String("c", "/etc/krakend/configuration.json", "Path to the configuration filename")
	flag.Parse()

	if *useGorilla {
		config.RoutingPattern = config.BracketsRouterPatternBuilder
	}
	parser := viper.New()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
	serviceConfig.Debug = serviceConfig.Debug || *debug
	if *port != 0 {
		serviceConfig.Port = *port
	}

	ctx := context.Background()

	logger, err := logging.NewLogger(*logLevel, os.Stdout, "[KRAKEND]")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	if *useGorilla {

		metric := metricsmux.New(ctx, serviceConfig.ExtraConfig, logger)

		// create a new proxy factory wrapping an instrumented HTTP backend factory
		pf := proxy.NewDefaultFactory(metric.DefaultBackendFactory(), logger)

		// inject the instrumented proxy factory over the previously created one
		routerCfg := gorilla.DefaultConfig(metric.ProxyFactory("pipe", pf), logger)
		defaultHandlerFactory := routerCfg.HandlerFactory
		// declare the instrumented router handler
		routerCfg.HandlerFactory = metric.NewHTTPHandlerFactory(defaultHandlerFactory)
		routerFactory := mux.NewFactory(routerCfg)

		routerFactory.NewWithContext(ctx).Run(serviceConfig)

	} else {

		metric := metricsgin.New(ctx, serviceConfig.ExtraConfig, logger)

		// create a new proxy factory wrapping an instrumented HTTP backend factory
		pf := proxy.NewDefaultFactory(metric.DefaultBackendFactory(), logger)

		engine := gin.Default()
		routerFactory := krakendgin.NewFactory(krakendgin.Config{
			// declare the instrumented router handler
			HandlerFactory: metric.NewHTTPHandlerFactory(krakendgin.EndpointHandler),
			// inject the instrumented proxy factory over the previously created one
			ProxyFactory: metric.ProxyFactory("pipe", pf),
			// other boring stuff...
			Engine:      engine,
			Middlewares: []gin.HandlerFunc{},
			Logger:      logger,
		})

		routerFactory.NewWithContext(ctx).Run(serviceConfig)
	}
}
