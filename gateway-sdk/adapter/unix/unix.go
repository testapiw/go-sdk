// Package unix provides a gateway.Adapter that communicates with the
// collector over a Unix Domain Socket.
//
// The wire protocol matches the gateway server:
//
//	client ──► {"action":"...","payload":...} ──► server
//	client ◄── {"status":"ok"|"error","message":"..."} ◄── server
//
// Each Call opens a fresh connection, sends exactly one request, reads exactly
// one response and closes the connection.
package unix

import (
	"fmt"
	"time"
)

// DefaultTimeout limits the whole round-trip (dial + write + read).
const DefaultTimeout = 5 * time.Second

// Config holds the Unix socket adapter parameters.
type Config struct {
	// SocketPath is the filesystem path of the Unix Domain Socket.
	// Required.
	SocketPath string

	// Timeout limits the whole round-trip. Defaults to DefaultTimeout.
	Timeout time.Duration
}

// Adapter sends requests to the collector over a Unix Domain Socket.
type Adapter struct {
	cfg Config
}

// New creates a Unix socket adapter.
func New(cfg Config) (*Adapter, error) {
	if cfg.SocketPath == "" {
		return nil, fmt.Errorf("unix: socket path is required")
	}

	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}

	return &Adapter{cfg: cfg}, nil
}

// Call sends req over the Unix socket and returns the response.
//
// Call is safe for concurrent use: every invocation uses its own connection.
//
// NOTE: currently unused — the collector only receives signals from WP
// (server side). This is the client-side abstraction for future use when a
// Go service needs to send data to the collector.
//
// func (a *Adapter) Call(ctx context.Context, req gateway.Request) (gateway.Response, error) {
// 	dialer := net.Dialer{Timeout: a.cfg.Timeout}
//
// 	conn, err := dialer.DialContext(ctx, "unix", a.cfg.SocketPath)
// 	if err != nil {
// 		return gateway.Response{}, fmt.Errorf("%w: dial %s: %v", gateway.ErrTransport, a.cfg.SocketPath, err)
// 	}
// 	defer conn.Close()
//
// 	// Bound the whole round-trip with a single deadline.
// 	deadline := time.Now().Add(a.cfg.Timeout)
// 	if err := conn.SetDeadline(deadline); err != nil {
// 		return gateway.Response{}, fmt.Errorf("%w: set deadline: %v", gateway.ErrTransport, err)
// 	}
//
// 	if err := json.NewEncoder(conn).Encode(req); err != nil {
// 		return gateway.Response{}, fmt.Errorf("%w: encode request: %v", gateway.ErrTransport, err)
// 	}
//
// 	var resp gateway.Response
//
// 	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
// 		return gateway.Response{}, fmt.Errorf("%w: decode response: %v", gateway.ErrTransport, err)
// 	}
//
// 	if !resp.OK() {
// 		return resp, fmt.Errorf("%w: %s", gateway.ErrRemote, resp.Message)
// 	}
//
// 	return resp, nil
// }