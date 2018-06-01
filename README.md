KrakenD metrics
====

A set of building blocks for instrumenting [KrakenD](http://www.krakend.io) gateways

## Available middlewares

There are the avaliable middlewares to add to the KrakenD pipes.

1. Backend
2. Proxy
3. Router

## Available router flavours

1. [mux](github.com/devopsfaith/krakend-metrics/blob/master/mux) Mux based routers and handlers
2. [gin](github.com/devopsfaith/krakend-metrics/blob/master/gin) Gin based routers and handlers

Check the examples and the documentation for more details

## Configuration

You need to add an ExtraConfig section to the configuration to enable the metrics collector (an empty one will use the defaults).

You can disable metrics by layer by explicitly setting it to true (the default is to be enabled):

- `backend_disabled` bool
- `proxy_disabled` bool
- `router_disabled` bool

Or configure the collection time of metrics:

- `collection_time` (default: 60s) (Ex: "30s", "5m", "500ms", ...)

### Configuration Example

This configuration will set the _collection time_ to 2 minutes and will disable the proxy metrics collector (backend and router metrics will be enabled since the default for all layers is to be enabled).
```
  "extra_config": {
    "github_com/devopsfaith/krakend-metrics": {
      "collection_time": "2m",
      "proxy_disabled": true,
    }
  }
  ```

  or leave the defaults:
  ```
  "extra_config": {
    github_com/devopsfaith/krakend-metrics": {}
  }
  ```
