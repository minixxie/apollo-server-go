# mock-apollo-go
This project is developed to mock Apollo (Ctrip) server APIs in golang.
It serves kv config values from a local file with support for hot reloading config changes.

The main purpose for this project is to serve as a config sidecar in a kubernetes cluster,
utilizing Kubernetes native ConfigMaps as a source of truth. This aids in migrating applications from the non-Kubernetes world while keeping dependencies to a minimum.

## Feature support
This project currently supports 3 APIs for fetching config:
* GET /configs/:appId/:cluster/:namespace
* GET /configfiles/json/:appId/:cluster/:namespace
* GET /notifications/v2 _(long polling)_

# Usage Guide
## Building it locally
`$ make`\
`$ ./mock-apollo-go --help`
```
Usage of ./mock-apollo-go:
  -config-port int
        config HTTP server port (default 8070)
  -file string
        config filepath (default "./configs/example.yaml")
  -internal-port int
        internal HTTP server port (default 9090)
  -poll-timeout duration
        long poll timeout (default 1m0s)
```

## Health check
There is a health check endpoint on the config HTTP server:\
`$ curl "HTTP://localhost:8070/healthz"`

## Ctrl interface
This is used for controlling certain features/abilities of this process via the internal HTTP server.

### Logging
Dynamically changing the logging level:\
`$ curl -X PATCH "HTTP://localhost:9090/ctrl/logging?level=debug"`

Supported logging levels are:
* debug
* info _(default)_
* warn
* error

## Golang pprof
Golang pprof APIs are served via the internal HTTP server at `/debug/pprof*`