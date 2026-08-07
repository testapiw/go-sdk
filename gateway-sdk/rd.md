# gateway-sdk

Abstract layer for switching the communication method between the WP service
and the collector. Used on the Go side.

```
WP ──► Gateway ──► Adapter ──► (socket | nginx | ...) ──► Collector
```

## Structure

```
gateway-sdk/
├── go.mod
├── gateway.go      // Adapter interface + Gateway wrapper (client)
├── server.go       // Handler + Server interface (server)
├── request.go      // Request, NewRequest
├── response.go     // Response
├── errors.go       // sentinel errors
│
└── adapter/
    ├── unix/
    │   └── unix.go   // Unix Domain Socket client
    └── socket/
        └── server/   // Unix Domain Socket server (SOLID)
            ├── config.go   // Config, defaults, validation
            ├── server.go   // Server struct, New, Run, Close (lifecycle)
            ├── router.go   // Subscribe (handler registration)
            ├── handler.go  // handle (per-connection dispatch)
            ├── codec.go    // readRequest, writeResponse, writeError
            ├── retry.go    // retry policy
            └── resolve.go  // user/group resolution
```

## Contract

Client side (sending data):

```go
type Adapter interface {
    Call(ctx context.Context, req Request) (Response, error)
}
```

Server side (receiving data):

```go
type Handler func(ctx context.Context, payload json.RawMessage) error

type Server interface {
    Run(ctx context.Context) error
    Subscribe(action string, handler Handler) error
}
```

- `Request`  — `{"action":"...","payload":...}`
- `Response` — `{"status":"ok"|"error","message":"..."}`

## Adapters

| Adapter | Side | Transport | When used |
|---------|------|-----------|-----------|
| `adapter/unix` | client | Unix Domain Socket | WP and collector on the same server |
| `adapter/socket/server` | server | Unix Domain Socket | WP and collector on the same server |
| `adapter/http` (TODO) | client/server | HTTP via nginx | WP and collector on different servers |

## Switching

Choosing an adapter is a configuration concern, not a code change:

```go
// Now: same server
adapter, _ := unix.New(unix.Config{SocketPath: "/run/collector.sock"})

// Future: different servers
// adapter, _ := http.New(http.Config{URL: "https://collector.example.com"})

gw, _ := gateway.New(adapter)
resp, err := gw.Call(ctx, req)
```

Server side (ingress):

```go
var srv gateway.Server
srv, _ = server.New(server.Config{SocketPath: "/run/collector.sock"})
srv.Subscribe("coins:list", handler)
srv.Run(ctx)
```

## Queue

Data sent by WP is written to a queue (NATS). gateway-sdk is responsible only
for transport; business logic and queue writes remain outside its scope.